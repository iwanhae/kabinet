package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// AppendEvent adds a single event to the storage channel
func (s *Storage) AppendEvent(ctx context.Context, k8sEvent *corev1.Event) error {
	select {
	case s.eventCh <- k8sEvent:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	}
}

// runBatchInserter runs the background batch inserter goroutine
func (s *Storage) runBatchInserter(ctx context.Context) {
	time.Sleep(time.Duration(5-time.Now().Second()%5) * time.Second) // no special reason for this, just to make logs easier to read
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	batch := make([]*corev1.Event, 0, 1000)

	for {
		select {
		case <-ctx.Done():
			log.Println("storage: context cancelled, flushing remaining events...")
			if len(batch) > 0 {
				if err := s.AppendEvents(batch); err != nil {
					log.Printf("storage: error appending remaining events: %v", err)
				}
			}
			s.Close()
			return
		case event := <-s.eventCh:
			batch = append(batch, event)
			if len(batch) >= 1000 {
				if err := s.AppendEvents(batch); err != nil {
					log.Printf("storage: error appending events: %v", err)
				}
				batch = make([]*corev1.Event, 0, 1000)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if err := s.AppendEvents(batch); err != nil {
					log.Printf("storage: error appending events on tick: %v", err)
				}
				batch = make([]*corev1.Event, 0, 1000)
			}
		}
	}
}

// AppendEvents inserts a batch of Kubernetes events into the database
func (s *Storage) AppendEvents(k8sEvents []*corev1.Event) error {
	if len(k8sEvents) == 0 {
		return nil
	}

	// BEGIN TRANSACTION
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err) // TODO: retry?
	}
	defer tx.Rollback()

	for _, k8sEvent := range k8sEvents {
		var series any
		if k8sEvent.Series != nil {
			series = map[string]any{
				"count":            k8sEvent.Series.Count,
				"lastObservedTime": k8sEvent.Series.LastObservedTime.Time,
			}
		} else {
			series = nil
		}

		var related any
		if k8sEvent.Related != nil {
			related = map[string]any{
				"kind":            k8sEvent.Related.Kind,
				"namespace":       k8sEvent.Related.Namespace,
				"name":            k8sEvent.Related.Name,
				"uid":             string(k8sEvent.Related.UID),
				"apiVersion":      k8sEvent.Related.APIVersion,
				"resourceVersion": k8sEvent.Related.ResourceVersion,
				"fieldPath":       k8sEvent.Related.FieldPath,
			}
		} else {
			related = nil
		}

		args := []any{
			k8sEvent.Kind,
			k8sEvent.APIVersion,
			map[string]any{
				"name":              k8sEvent.ObjectMeta.Name,
				"namespace":         k8sEvent.ObjectMeta.Namespace,
				"uid":               string(k8sEvent.ObjectMeta.UID),
				"resourceVersion":   k8sEvent.ObjectMeta.ResourceVersion,
				"creationTimestamp": k8sEvent.ObjectMeta.CreationTimestamp.Time,
			},
			map[string]any{
				"kind":            k8sEvent.InvolvedObject.Kind,
				"namespace":       k8sEvent.InvolvedObject.Namespace,
				"name":            k8sEvent.InvolvedObject.Name,
				"uid":             string(k8sEvent.InvolvedObject.UID),
				"apiVersion":      k8sEvent.InvolvedObject.APIVersion,
				"resourceVersion": k8sEvent.InvolvedObject.ResourceVersion,
				"fieldPath":       k8sEvent.InvolvedObject.FieldPath,
			},
			k8sEvent.Reason,
			k8sEvent.Message,
			map[string]any{
				"component": k8sEvent.Source.Component,
				"host":      k8sEvent.Source.Host,
			},
			k8sEvent.FirstTimestamp.Time,
			k8sEvent.LastTimestamp.Time,
			k8sEvent.Count,
			k8sEvent.Type,
			k8sEvent.EventTime.Time,
			series,
			k8sEvent.Action,
			related,
			k8sEvent.ReportingController,
			k8sEvent.ReportingInstance,
		}
		placeholder := "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		query := fmt.Sprintf("INSERT OR IGNORE INTO kube_events VALUES %s", placeholder)
		_, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to batch insert events: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("storage: inserted %d events into kube_events", len(k8sEvents))

	return nil
}
