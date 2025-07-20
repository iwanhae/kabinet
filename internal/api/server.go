package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/iwanhae/kube-event-analyzer/internal/storage"
)

// Server holds the dependencies for the API server.
type Server struct {
	storage *storage.Storage
	server  *http.Server
}

// New creates a new API server.
func New(storage *storage.Storage, port string) *Server {
	s := &Server{
		storage: storage,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/query", s.handleQuery)

	s.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	return s
}

// Start runs the API server.
func (s *Server) Start() error {
	log.Printf("API server listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API server...")
	return s.server.Shutdown(ctx)
}

type queryRequest struct {
	Query string    `json:"query"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type queryResponse struct {
	Results []map[string]any `json:"results"`
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Query == "" || req.Start.IsZero() || req.End.IsZero() {
		http.Error(w, "Missing required fields: query, start, end", http.StatusBadRequest)
		return
	}

	rows, err := s.storage.RangeQuery(r.Context(), req.Query, req.Start, req.End)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute query: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	results, err := serializeRows(rows)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize results: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(queryResponse{
		Results: results,
	}); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func serializeRows(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]any
	for rows.Next() {
		rowValues := make([]any, len(columns))
		rowPointers := make([]any, len(columns))
		for i := range rowValues {
			rowPointers[i] = &rowValues[i]
		}

		if err := rows.Scan(rowPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowData := make(map[string]any, len(columns))
		for i, colName := range columns {
			val := rowValues[i]

			// To keep JSON clean, we handle byte slices (like DuckDB structs)
			if b, ok := val.([]byte); ok {
				rowData[colName] = string(b)
			} else {
				rowData[colName] = val
			}
		}
		results = append(results, rowData)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return results, nil
}
