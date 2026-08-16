package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/iwanhae/kabinet/internal/api"
	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/collector"
	"github.com/iwanhae/kabinet/internal/config"
	"github.com/iwanhae/kabinet/internal/lifecycle"
	"github.com/iwanhae/kabinet/internal/metrics"
	"github.com/iwanhae/kabinet/internal/query"
	"github.com/iwanhae/kabinet/internal/wal"
)

//go:embed all:dist
var distFS embed.FS

func main() {
	cfg := config.Load()

	// Initialize metrics registry and counters
	metrics.Init()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Set up signal handling for graceful shutdown
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopCh
		log.Println("main: received shutdown signal, initiating graceful shutdown...")
		cancel()
	}()

	// --- Data layout ---
	walDir := filepath.Join(cfg.DataDir, "wal")
	archiveDir := filepath.Join(cfg.DataDir, "archive")
	tmpDir := filepath.Join(cfg.DataDir, "tmp")

	// tmp holds only spill files, snapshots, and unpublished compactor
	// outputs; anything left over is garbage from a previous run.
	if err := os.RemoveAll(tmpDir); err != nil {
		log.Printf("main: failed to clear tmp directory: %v", err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		log.Fatalf("main: failed to create tmp directory: %v", err)
	}

	// --- Ingest (WAL) ---
	walWriter, err := wal.NewWriter(ctx, wal.Options{
		Dir:            walDir,
		TempDir:        tmpDir,
		RotateInterval: cfg.WalRotateInterval,
		RotateBytes:    cfg.WalRotateBytes,
	})
	if err != nil {
		log.Fatalf("main: failed to initialize wal: %v", err)
	}

	// --- Catalog ---
	cat, err := catalog.Open(archiveDir)
	if err != nil {
		log.Fatalf("main: failed to open catalog: %v", err)
	}

	// --- Query ---
	executor, err := query.New(cat, walWriter, walDir)
	if err != nil {
		log.Fatalf("main: failed to initialize query executor: %v", err)
	}

	// --- Manage (lifecycle) ---
	manager := lifecycle.New(cat, lifecycle.Config{
		WalDir:            walDir,
		TempDir:           tmpDir,
		StorageLimitBytes: cfg.StorageLimitBytes,
		CompactInterval:   cfg.CompactInterval,
		MergeTargetBytes:  cfg.MergeTargetBytes,
		MemoryLimitMB:     cfg.CompactMemoryLimitMB,
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Run(ctx)
	}()

	// --- API Server ---
	stats := func(ctx context.Context) map[string]any {
		return map[string]any{
			"data_dir": cfg.DataDir,
			"wal":      walWriter.Stats(),
			"archive":  cat.Stats(),
		}
	}
	apiServer := api.New(executor, stats, cfg.ListenPort, distFS)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("main: starting API server on port %s...", cfg.ListenPort)
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("main: API server failed: %v", err)
		}
		log.Println("main: API server closed")
	}()

	// --- Collector ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("main: starting event collector...")
		runCollector(ctx, walWriter)
		log.Println("main: collector finished")
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// --- Graceful Shutdown ---
	if err := apiServer.Shutdown(context.Background()); err != nil {
		log.Printf("main: error during API server shutdown: %v", err)
	}

	log.Println("main: waiting for all background processes to finish...")
	wg.Wait()

	// Flush and seal the active WAL segment, then release DuckDB.
	walWriter.Wait()
	executor.Close()
	log.Println("main: all processes finished. exiting.")
}

func runCollector(ctx context.Context, walWriter *wal.Writer) {
	c, err := collector.ConnectK8s()
	if err != nil {
		log.Printf("collector: error connecting to Kubernetes: %v. collector will not run.", err)
		return
	}

	watcher := collector.WatchEvents(ctx, c)

	log.Println("collector: event collector started.")
	for {
		select {
		case event, ok := <-watcher:
			if !ok {
				log.Println("collector: event watcher channel closed. collector is stopping.")
				return
			}

			// track collected event
			metrics.EventsCollected.Inc()

			// Events are stored as raw JSON; field fallbacks (empty
			// firstTimestamp/count) are applied at read time by the schema
			// projection.
			if err := walWriter.Append(ctx, &event); err != nil {
				log.Printf("collector: failed to append event: %v", err)
			}
		case <-ctx.Done():
			log.Println("collector: context cancelled. stopping event collector.")
			return
		}
	}
}
