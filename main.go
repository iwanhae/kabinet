package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iwanhae/kube-event-analyzer/internal/collector"
	"github.com/iwanhae/kube-event-analyzer/internal/storage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopCh
		log.Println("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	c, err := collector.ConnectK8s()
	if err != nil {
		panic(err)
	}

	db, connector, err := storage.NewDB("events.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := storage.CreateTable(db); err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	lastResourceVersion, err := storage.GetLastResourceVersion(db)
	if err != nil {
		log.Fatalf("failed to get last resource version: %v", err)
	}
	log.Printf("Starting event watch from resourceVersion: %s", lastResourceVersion)

	appender, cleanup, err := storage.NewAppender(ctx, connector)
	if err != nil {
		log.Fatalf("failed to create appender: %v", err)
	}
	defer cleanup()

	watcher, err := collector.WatchEvents(ctx, c, lastResourceVersion)
	if err != nil {
		panic(err)
	}
	enc := json.NewEncoder(os.Stdout)
	for event := range watcher.ResultChan() {
		enc.Encode(event)
		if event.Type == watch.Added || event.Type == watch.Modified {
			k8sEvent := event.Object.(*corev1.Event)

			fmt.Printf("Event: %s/%s %s (%s)\n", k8sEvent.Namespace, k8sEvent.Name, k8sEvent.Reason, event.Type)

			if err := storage.AppendEvent(appender, k8sEvent); err != nil {
				log.Printf("failed to append event: %v", err)
			}
		}
	}
	log.Println("Event watcher stopped. Program will exit.")
}
