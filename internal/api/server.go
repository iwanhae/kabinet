package api

import (
	"compress/gzip"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/iwanhae/kabinet/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
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
	mux.HandleFunc("/download", s.handleDownload)

	// pprof profiling endpoints
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Frontend Handler
	staticFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Fatalf("server: failed to create static file system: %v", err)
	}
	fileServer := http.FileServerFS(staticFS)

	mux.Handle("/", fileServer)
	// serve index.html for SPA routing under /p/*
	mux.Handle("/p/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// backward compatibility: redirect /discover -> /p/discover
	mux.HandleFunc("/discover", func(w http.ResponseWriter, r *http.Request) {
		target := "/p/discover"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})

	// CORS handler to allow all origins
	corsMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		mux.ServeHTTP(w, r)
	})

	s.server = &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}
	return s
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(s.storage.Stats(r.Context()))
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "server: only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	qs := r.URL.Query()
	where := qs.Get("where")
	fromStr := qs.Get("from")
	toStr := qs.Get("to")
	if fromStr == "" || toStr == "" {
		http.Error(w, "server: missing required parameters: from, to", http.StatusBadRequest)
		return
	}

	start, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("server: invalid from parameter: %v", err), http.StatusBadRequest)
		return
	}
	end, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("server: invalid to parameter: %v", err), http.StatusBadRequest)
		return
	}

	pr, pw := io.Pipe()
	gzw := gzip.NewWriter(w)

	w.Header().Set("Content-Type", "application/octet-stream")
	filename := fmt.Sprintf("events_%s_%s.jsonl.gz", start.Format("20060102T150405"), end.Format("20060102T150405"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	eg, ctx := errgroup.WithContext(r.Context())

	eg.Go(func() error {
		defer pw.Close()
		enc := json.NewEncoder(pw)
		_, err := s.storage.StreamEvents(ctx, where, start, end, func(row map[string]any) error {
			return enc.Encode(row)
		})
		return err
	})

	eg.Go(func() error {
		defer gzw.Close()
		_, err := io.Copy(gzw, pr)
		return err
	})

	if err := eg.Wait(); err != nil {
		log.Printf("server: download error: %v", err)
	}
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
