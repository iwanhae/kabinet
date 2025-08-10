package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/iwanhae/kabinet/internal/utils"
	_ "github.com/marcboeker/go-duckdb/v2"
	corev1 "k8s.io/api/core/v1"
)

// Storage represents the main storage interface for Kubernetes events
type Storage struct {
	db *sql.DB

	dbPath  string
	dataDir string

	eventCh chan *corev1.Event

	wg *sync.WaitGroup
}

// New creates a new Storage instance
func New(ctx context.Context, dbPath string) (*Storage, error) {
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Remove the WAL file
	// This is duckdb's bug that failed to recover from a crash.
	os.Remove(path.Join(dataDir, "events.db.wal"))

	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_WRITE")
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	s := &Storage{
		db:      db,
		dbPath:  dbPath,
		dataDir: dataDir,
		eventCh: make(chan *corev1.Event, 2000),
		wg:      &sync.WaitGroup{},
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runBatchInserter(ctx)
	}()

	return s, nil
}

// Stats returns storage statistics
func (s *Storage) Stats(ctx context.Context) map[string]any {
	errs := utils.MultiError{}

	dataDirSize, err := s.dataDirSize()
	if err != nil {
		errs.Add(fmt.Errorf("storage: error getting data directory size: %v", err))
		dataDirSize = 0
	}

	statsTempFiles, err := func() ([]map[string]any, error) {
		rows, err := s.db.QueryContext(ctx, "FROM duckdb_temporary_files();")
		if err != nil {
			return nil, fmt.Errorf("storage: error getting temporary files: %v", err)
		}
		defer rows.Close()
		return serializeRows(rows)
	}()
	if err != nil {
		errs.Add(err)
	}

	statsMemory, err := func() ([]map[string]any, error) {
		rows, err := s.db.QueryContext(ctx, "SELECT * FROM duckdb_memory();")
		if err != nil {
			return nil, fmt.Errorf("storage: error getting memory usage: %v", err)
		}
		defer rows.Close()
		return serializeRows(rows)
	}()
	if err != nil {
		errs.Add(err)
	}

	return map[string]any{
		"db_stats":          s.db.Stats(),
		"event_channel":     len(s.eventCh),
		"data_dir":          s.dataDir,
		"data_dir_size":     dataDirSize,
		"duckdb_temp_files": statsTempFiles,
		"duckdb_memory":     statsMemory,
		"errors":            errs,
	}
}

// Wait waits for all background tasks to complete
func (s *Storage) Wait() {
	s.wg.Wait()
	log.Println("storage: all background tasks finished")
}

// Close closes the storage and database connection
func (s *Storage) Close() {
	log.Println("storage: closing storage...")
	if err := s.db.Close(); err != nil {
		log.Printf("storage: error closing database: %v", err)
	}
	log.Println("storage: closed successfully")
}
