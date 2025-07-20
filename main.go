package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	storage, err := storage.New(ctx, "data/events.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer storage.Close()

	lastResourceVersion, err := storage.GetLastResourceVersion()
	if err != nil {
		log.Fatalf("failed to get last resource version: %v", err)
	}
	log.Printf("Starting event watch from resourceVersion: %s", lastResourceVersion)

	// Start the data lifecycle manager
	go storage.ManageDataLifecycle(ctx, 3*time.Hour, 10*1024*1024*1024) // 3 hours interval, 10GB limit

	watcher, err := collector.WatchEvents(ctx, c, lastResourceVersion)
	if err != nil {
		panic(err)
	}
	for event := range watcher.ResultChan() {
		if event.Type == watch.Added || event.Type == watch.Modified {
			k8sEvent := event.Object.(*corev1.Event)

			fmt.Printf("Event: %s/%s %s (%s)\n", k8sEvent.Namespace, k8sEvent.Name, k8sEvent.Reason, event.Type)

			if err := storage.AppendEvent(k8sEvent); err != nil {
				log.Printf("failed to append event: %v", err)
			}
		}
	}
	log.Println("Event watcher stopped. Program will exit.")
}
