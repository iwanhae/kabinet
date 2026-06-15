package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	// Max JSONL file size before rotation (50MB)
	maxJSONLFileSize = 50 * 1024 * 1024
	// Max age of JSONL file before rotation (5 minutes)
	maxJSONLFileAge = 5 * time.Minute
)

// JSONLWriter manages writing Kubernetes events to JSONL files with automatic rotation
type JSONLWriter struct {
	currentFile    *os.File
	currentPath    string
	startTimestamp int64
	mu             sync.Mutex
	dataDir        string
}

// NewJSONLWriter creates a new JSONLWriter instance
func NewJSONLWriter(dataDir string) *JSONLWriter {
	return &JSONLWriter{
		dataDir: dataDir,
	}
}

// WriteEvent marshals a Kubernetes event to JSON and appends to current file.
// Rotates file when size > 50MB OR age > 5 minutes.
func (w *JSONLWriter) WriteEvent(event *corev1.Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if we need to rotate (no file, too large, or too old)
	if w.currentFile == nil || w.needsRotation() {
		if err := w.rotate(); err != nil {
			return fmt.Errorf("failed to rotate JSONL file: %w", err)
		}
	}

	// Convert event to flat map for JSON serialization
	flatMap := eventToFlatMap(event)
	jsonData, err := json.Marshal(flatMap)
	if err != nil {
		return fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	// Write JSON line with newline
	if _, err := w.currentFile.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write event to JSONL: %w", err)
	}

	return nil
}

// needsRotation checks if the current file needs rotation based on size or age
func (w *JSONLWriter) needsRotation() bool {
	if w.currentFile == nil {
		return true
	}

	// Check file size
	info, err := w.currentFile.Stat()
	if err != nil {
		log.Printf("storage: error getting file info: %v", err)
		return true
	}
	if info.Size() >= maxJSONLFileSize {
		return true
	}

	// Check file age
	age := time.Since(time.Unix(w.startTimestamp, 0))
	return age >= maxJSONLFileAge
}

// rotate closes the current file and creates a new one with timestamp-based naming
func (w *JSONLWriter) rotate() error {
	// Close existing file if open
	if w.currentFile != nil {
		if err := w.currentFile.Close(); err != nil {
			log.Printf("storage: error closing JSONL file: %v", err)
		}
		w.currentFile = nil
	}

	// Create new file with timestamp-based naming
	now := time.Now()
	w.startTimestamp = now.Unix()
	w.currentPath = filepath.Join(w.dataDir, fmt.Sprintf("events_%d.jsonl", w.startTimestamp))

	file, err := os.OpenFile(w.currentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create JSONL file: %w", err)
	}

	w.currentFile = file
	log.Printf("storage: created new JSONL file: %s", w.currentPath)
	return nil
}

// Close closes the current JSONL file
func (w *JSONLWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		if err := w.currentFile.Close(); err != nil {
			return err
		}
		w.currentFile = nil
	}
	return nil
}

// GetCurrentPath returns the current JSONL file path (for stats)
func (w *JSONLWriter) GetCurrentPath() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentPath
}

// eventToFlatMap converts a Kubernetes Event struct to a flat map matching DuckDB schema
func eventToFlatMap(event *corev1.Event) map[string]any {
	// Handle series field
	var series any
	if event.Series != nil {
		series = map[string]any{
			"count":            event.Series.Count,
			"lastObservedTime": event.Series.LastObservedTime.Time,
		}
	} else {
		series = nil
	}

	// Handle related field
	var related any
	if event.Related != nil {
		related = map[string]any{
			"kind":            event.Related.Kind,
			"namespace":       event.Related.Namespace,
			"name":            event.Related.Name,
			"uid":             string(event.Related.UID),
			"apiVersion":      event.Related.APIVersion,
			"resourceVersion": event.Related.ResourceVersion,
			"fieldPath":       event.Related.FieldPath,
		}
	} else {
		related = nil
	}

	return map[string]any{
		"kind":       event.Kind,
		"apiVersion": event.APIVersion,
		"metadata": map[string]any{
			"name":              event.ObjectMeta.Name,
			"namespace":         event.ObjectMeta.Namespace,
			"uid":               string(event.ObjectMeta.UID),
			"resourceVersion":   event.ObjectMeta.ResourceVersion,
			"creationTimestamp": event.ObjectMeta.CreationTimestamp.Time,
		},
		"involvedObject": map[string]any{
			"kind":            event.InvolvedObject.Kind,
			"namespace":       event.InvolvedObject.Namespace,
			"name":            event.InvolvedObject.Name,
			"uid":             string(event.InvolvedObject.UID),
			"apiVersion":      event.InvolvedObject.APIVersion,
			"resourceVersion": event.InvolvedObject.ResourceVersion,
			"fieldPath":       event.InvolvedObject.FieldPath,
		},
		"reason":             event.Reason,
		"message":            event.Message,
		"source": map[string]any{
			"component": event.Source.Component,
			"host":      event.Source.Host,
		},
		"firstTimestamp":     event.FirstTimestamp.Time,
		"lastTimestamp":      event.LastTimestamp.Time,
		"count":              event.Count,
		"type":               event.Type,
		"eventTime":          event.EventTime.Time,
		"series":             series,
		"action":             event.Action,
		"related":            related,
		"reportingComponent":  event.ReportingController,
		"reportingInstance":  event.ReportingInstance,
	}
}
