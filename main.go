package main

import (
	"context"
	"embed"
	"flag"
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
	role := flag.String("role", "all", "service role: all, writer, or reader")
	flag.Parse()

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// Set up signal handling for graceful shutdown
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopCh
		log.Printf("main: received shutdown signal for role=%s, initiating graceful shutdown...", *role)
		cancel()
	}()

	log.Printf("main: starting service with role: %s", *role)

	switch *role {
	case "writer":
		runWriter(ctx, &wg, cfg)
	case "reader":
		runReader(ctx, &wg, cfg)
	case "all":
		runAllInOne(ctx, &wg, cfg)
	default:
		log.Fatalf("main: unknown role: %s", *role)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	log.Println("main: waiting for all background processes to finish...")
	wg.Wait()
	log.Println("main: all processes finished. exiting.")
}

func runWriter(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config) {
	writer, err := storage.NewWriter(cfg.DBPath, cfg.ParquetPath)
	if err != nil {
		log.Fatalf("writer: failed to initialize storage writer: %v", err)
	}
	defer writer.Close()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("writer: starting data lifecycle manager...")
		writer.LifecycleManager(ctx, cfg.ArchiveInterval, cfg.StorageLimitBytes)
		log.Println("writer: data lifecycle manager finished")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("writer: starting event collector...")
		runCollector(ctx, writer)
		log.Println("writer: collector finished")
	}()
}

func runReader(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config) {
	reader, err := storage.NewReader(cfg.DBPath, cfg.ParquetPath)
	if err != nil {
		log.Fatalf("reader: failed to initialize storage reader: %v", err)
	}
	defer reader.Close()

	// The API server only needs a reader in this mode.
	apiServer := api.New(reader, nil, cfg.ListenPort, distFS)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("reader: starting API server on port %s...", cfg.ListenPort)
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("reader: API server failed: %v", err)
		}
		log.Println("reader: API server closed")
	}()

	// Graceful shutdown for the API server
	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("reader: error during API server shutdown: %v", err)
	}
}

func runAllInOne(ctx context.Context, wg *sync.WaitGroup, cfg *config.Config) {
	writer, err := storage.NewWriter(cfg.DBPath, cfg.ParquetPath)
	if err != nil {
		log.Fatalf("main: failed to initialize storage writer: %v", err)
	}
	defer writer.Close()

	reader, err := storage.NewReader(cfg.DBPath, cfg.ParquetPath)
	if err != nil {
		log.Fatalf("main: failed to initialize storage reader: %v", err)
	}
	defer reader.Close()

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

	// Graceful shutdown for the API server
	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("main: error during API server shutdown: %v", err)
	}
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
