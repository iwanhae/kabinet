package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/iwanhae/kube-event-analyzer/internal/api"
	"github.com/iwanhae/kube-event-analyzer/internal/collector"
	"github.com/iwanhae/kube-event-analyzer/internal/storage"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Set up signal handling for graceful shutdown
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopCh
		log.Println("Received shutdown signal, initiating graceful shutdown...")
		cancel()
	}()

	// --- Storage ---
	storage, err := storage.New(ctx, "data/events.db")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("Shutting down storage...")
		storage.Close()
	}()

	// --- API Server ---
	apiServer := api.New(storage, "8080")
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	// --- Collector and Data Lifecycle ---
	lastResourceVersion, err := storage.GetLastResourceVersion()
	if err != nil {
		log.Fatalf("Failed to get last resource version: %v", err)
	}
	log.Printf("Starting event watch from resourceVersion: %s", lastResourceVersion)

	wg.Add(1)
	go func() {
		defer wg.Done()
		storage.ManageDataLifecycle(ctx, 30*time.Minute, 10*1024*1024*1024) // 30 min interval, 10GB limit
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runCollector(ctx, storage, lastResourceVersion)
		log.Println("Collector finished. Exiting.")
		os.Exit(0)
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// --- Graceful Shutdown ---
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during API server shutdown: %v", err)
	}

	log.Println("Waiting for all background processes to finish...")
	wg.Wait()
	log.Println("All processes finished. Exiting.")
}

func runCollector(ctx context.Context, storage *storage.Storage, initialResourceVersion string) {
	c, err := collector.ConnectK8s()
	if err != nil {
		log.Printf("Error connecting to Kubernetes: %v. Collector will not run.", err)
		return
	}

	watcher := collector.WatchEvents(ctx, c)

	log.Println("Event collector started.")
	for {
		select {
		case event, ok := <-watcher:
			if !ok {
				log.Println("Event watcher channel closed. Collector is stopping.")
				return
			}
			if err := storage.AppendEvent(&event); err != nil {
				log.Printf("Failed to append event: %v", err)
			}
		case <-ctx.Done():
			log.Println("Context cancelled. Stopping event collector.")
			return
		}
	}
}
