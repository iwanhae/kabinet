package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
func New(storage *storage.Storage, port string, distFS embed.FS) *Server {
	s := &Server{
		storage: storage,
	}

	mux := http.NewServeMux()

	// API Handler
	mux.HandleFunc("/query", s.handleQuery)
	mux.HandleFunc("/stats", s.handleStats)

	// Frontend Handler
	staticFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Fatalf("server: failed to create static file system: %v", err)
	}
	fileServer := http.FileServerFS(staticFS)

	mux.Handle("/", fileServer)
	// serve index.html for SPA routing
	mux.Handle("/discover", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := staticFS.Open("index.html")
		if err != nil {
			http.Error(w, "server: could not open index.html", http.StatusInternalServerError)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, file) // Copy index.html content to response
	}))

	s.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	return s
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(s.storage.Stats(r.Context()))
}

// Start runs the API server.
func (s *Server) Start() error {
	log.Printf("server: listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("server: shutting down API server...")
	return s.server.Shutdown(ctx)
}

type queryRequest struct {
	Query string    `json:"query"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type queryResponse struct {
	Results        []map[string]any          `json:"results"`
	DurationMs     int64                     `json:"duration_ms"`
	Files          []storage.ParquetFileInfo `json:"files"`
	TotalFilesSize int64                     `json:"total_files_size_bytes"`
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "server: only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("server: invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Query == "" || req.Start.IsZero() || req.End.IsZero() {
		http.Error(w, "server: missing required fields: query, start, end", http.StatusBadRequest)
		return
	}

	rows, result, err := s.storage.RangeQuery(r.Context(), req.Query, req.Start, req.End)
	if err != nil {
		log.Printf("server: failed to execute query: %v", err)
		http.Error(w, fmt.Sprintf("server: failed to execute query: %v", err), http.StatusInternalServerError)
		return
	}

	var totalSize int64
	for _, f := range result.Files {
		totalSize += f.Size
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(queryResponse{
		Results:        rows,
		DurationMs:     result.Duration.Milliseconds(),
		Files:          result.Files,
		TotalFilesSize: totalSize,
	}); err != nil {
		log.Printf("server: failed to write response: %v", err)
	}
}
