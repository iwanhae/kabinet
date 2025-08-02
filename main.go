package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/iwanhae/kube-event-analyzer/internal/api"
	"github.com/iwanhae/kube-event-analyzer/internal/collector"
	"github.com/iwanhae/kube-event-analyzer/internal/config"
	"github.com/iwanhae/kube-event-analyzer/internal/storage"
)

//go:embed all:dist
var distFS embed.FS

func main() {
	cfg := config.Load()

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

	// --- Storage Writer ---
	writer, err := storage.NewWriter("data/events.db")
	if err != nil {
		log.Fatalf("main: failed to initialize storage writer: %v", err)
	}
	defer writer.Close()

	// --- Storage Reader ---
	reader, err := storage.NewReader("data/events.db")
	if err != nil {
		log.Fatalf("main: failed to initialize storage reader: %v", err)
	}
	defer reader.Close()

	// --- API Server ---
	apiServer := api.New(reader, writer, cfg.ListenPort, distFS)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("main: starting API server on port %s...", cfg.ListenPort)
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("main: API server failed: %v", err)
		}
		log.Println("main: API server closed")
	}()

	// --- Collector and Data Lifecycle ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("main: starting data lifecycle manager...")
		writer.LifecycleManager(ctx, cfg.ArchiveInterval, cfg.StorageLimitBytes)
		log.Println("main: data lifecycle manager finished")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("main: starting event collector...")
		runCollector(ctx, writer)
		log.Println("main: collector finished")
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// --- Graceful Shutdown ---
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("main: error during API server shutdown: %v", err)
	}

	log.Println("main: waiting for all background processes to finish...")
	wg.Wait()
	log.Println("main: all processes finished. exiting.")
}

func runCollector(ctx context.Context, writer *storage.Writer) {
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

			// if the event is missing some fields, set them to the creation timestamp
			if event.FirstTimestamp.IsZero() {
				event.FirstTimestamp = event.ObjectMeta.CreationTimestamp
			}
			if event.LastTimestamp.IsZero() {
				event.LastTimestamp = event.FirstTimestamp
			}
			if event.Count == 0 {
				event.Count = 1
			}

			if err := writer.AppendEvent(&event); err != nil {
				log.Printf("collector: failed to append event: %v", err)
			}
		case <-ctx.Done():
			log.Println("collector: context cancelled. stopping event collector.")
			return
		}
	}
}
