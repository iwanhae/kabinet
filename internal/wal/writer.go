// Package wal implements the ingest layer: an append-only write-ahead log of
// raw Kubernetes Event JSON, stored as zstd-compressed JSONL segment files.
// Every flush writes one complete zstd frame, so a crash costs at most the
// last unflushed batch; a torn tail is truncated at the last complete frame
// during recovery.
package wal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/klauspost/compress/zstd"
	corev1 "k8s.io/api/core/v1"
)

// Options configures a Writer.
type Options struct {
	// Dir is the WAL directory (segment files live here).
	Dir string
	// TempDir holds active-segment snapshots handed to the query layer.
	TempDir string
	// RotateInterval seals the active segment after this duration.
	RotateInterval time.Duration
	// RotateBytes seals the active segment once it reaches this size.
	RotateBytes int64
	// FlushInterval is the max time an event sits in the in-memory batch.
	FlushInterval time.Duration
	// MaxBatchEvents flushes the batch early once it holds this many events.
	MaxBatchEvents int
}

func (o *Options) withDefaults() {
	if o.RotateInterval <= 0 {
		o.RotateInterval = time.Minute
	}
	if o.RotateBytes <= 0 {
		o.RotateBytes = 8 << 20
	}
	if o.FlushInterval <= 0 {
		o.FlushInterval = 5 * time.Second
	}
	if o.MaxBatchEvents <= 0 {
		o.MaxBatchEvents = 1000
	}
}

// Writer appends Kubernetes events to the WAL.
type Writer struct {
	opts    Options
	eventCh chan *corev1.Event
	enc     *zstd.Encoder
	wg      sync.WaitGroup

	mu          sync.Mutex
	file        *os.File
	activePath  string
	activeStart time.Time
	// stableSize is the byte offset up to which the active segment contains
	// only complete zstd frames (i.e. safe to read).
	stableSize int64
	// segMin/segMax track the event-time range of the active segment, used
	// for the sealed name so query pruning works on event time (which can
	// lag ingestion time, e.g. after an informer relist).
	segMin, segMax time.Time

	snapMu  sync.Mutex
	snapKey string
}

// NewWriter recovers any crashed segments in opts.Dir and starts the
// background batch loop. The loop stops, flushes, and seals the active
// segment when ctx is cancelled; use Wait to block until that finishes.
func NewWriter(ctx context.Context, opts Options) (*Writer, error) {
	opts.withDefaults()
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create wal directory: %w", err)
	}
	if err := os.MkdirAll(opts.TempDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create wal temp directory: %w", err)
	}
	if err := recoverOpenSegments(opts.Dir); err != nil {
		return nil, fmt.Errorf("failed to recover wal segments: %w", err)
	}

	enc, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}

	w := &Writer{
		opts:    opts,
		eventCh: make(chan *corev1.Event, 2000),
		enc:     enc,
	}
	w.wg.Add(1)
	go w.run(ctx)
	return w, nil
}

// Append queues an event for the next batch flush.
func (w *Writer) Append(ctx context.Context, event *corev1.Event) error {
	select {
	case w.eventCh <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Wait blocks until the background loop has flushed and sealed after
// context cancellation.
func (w *Writer) Wait() {
	w.wg.Wait()
}

// Stats returns ingest statistics for the /stats endpoint.
func (w *Writer) Stats() map[string]any {
	w.mu.Lock()
	defer w.mu.Unlock()
	return map[string]any{
		"queue_len":          len(w.eventCh),
		"active_segment":     w.activePath,
		"active_stable_size": w.stableSize,
	}
}

func (w *Writer) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.opts.FlushInterval)
	defer ticker.Stop()

	var buf bytes.Buffer
	pending := 0
	var batchMin, batchMax time.Time

	flush := func() {
		if pending == 0 {
			return
		}
		if err := w.flush(buf.Bytes(), batchMin, batchMax); err != nil {
			log.Printf("wal: failed to flush %d events: %v", pending, err)
		} else {
			log.Printf("wal: flushed %d events (%d bytes jsonl)", pending, buf.Len())
		}
		buf.Reset()
		pending = 0
		batchMin, batchMax = time.Time{}, time.Time{}
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("wal: context cancelled, flushing and sealing...")
			flush()
			w.mu.Lock()
			w.sealLocked()
			w.mu.Unlock()
			return
		case event := <-w.eventCh:
			line, err := json.Marshal(event)
			if err != nil {
				log.Printf("wal: failed to marshal event: %v", err)
				continue
			}
			buf.Write(line)
			buf.WriteByte('\n')
			pending++
			ts := eventTime(event)
			if batchMin.IsZero() || ts.Before(batchMin) {
				batchMin = ts
			}
			if ts.After(batchMax) {
				batchMax = ts
			}
			if pending >= w.opts.MaxBatchEvents {
				flush()
				w.maybeRotate()
			}
		case <-ticker.C:
			flush()
			w.maybeRotate()
		}
	}
}

// eventTime returns the effective event time used for segment time ranges,
// mirroring the lastTimestamp fallbacks in the schema projection.
func eventTime(event *corev1.Event) time.Time {
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	if !event.CreationTimestamp.IsZero() {
		return event.CreationTimestamp.Time
	}
	return time.Now()
}

// flush compresses one batch of JSONL into a single zstd frame and appends it
// to the active segment.
func (w *Writer) flush(jsonl []byte, batchMin, batchMax time.Time) error {
	frame := w.enc.EncodeAll(jsonl, nil)

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.openSegmentLocked(); err != nil {
			return err
		}
	}
	if _, err := w.file.Write(frame); err != nil {
		return fmt.Errorf("failed to write frame: %w", err)
	}
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync segment: %w", err)
	}
	w.stableSize += int64(len(frame))
	if w.segMin.IsZero() || batchMin.Before(w.segMin) {
		w.segMin = batchMin
	}
	if batchMax.After(w.segMax) {
		w.segMax = batchMax
	}
	return nil
}

func (w *Writer) openSegmentLocked() error {
	start := time.Now()
	for attempt := 0; ; attempt++ {
		path := filepath.Join(w.opts.Dir, openName(start))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_EXCL, 0o644)
		if err != nil {
			if os.IsExist(err) && attempt < 10 {
				start = start.Add(time.Millisecond)
				continue
			}
			return fmt.Errorf("failed to create segment: %w", err)
		}
		w.file = f
		w.activePath = path
		w.activeStart = start
		w.stableSize = 0
		return nil
	}
}

func (w *Writer) maybeRotate() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil || w.stableSize == 0 {
		return
	}
	if time.Since(w.activeStart) < w.opts.RotateInterval && w.stableSize < w.opts.RotateBytes {
		return
	}
	w.sealLocked()
}

// sealLocked closes the active segment and renames it to its immutable
// sealed name. An empty active segment is removed instead.
func (w *Writer) sealLocked() {
	if w.file == nil {
		return
	}
	if err := w.file.Close(); err != nil {
		log.Printf("wal: failed to close active segment: %v", err)
	}
	if w.stableSize == 0 {
		if err := os.Remove(w.activePath); err != nil {
			log.Printf("wal: failed to remove empty segment %s: %v", w.activePath, err)
		}
	} else {
		sealed := filepath.Join(w.opts.Dir, sealedName(w.segMin, w.segMax))
		if err := os.Rename(w.activePath, sealed); err != nil {
			log.Printf("wal: failed to seal segment %s: %v", w.activePath, err)
		} else {
			log.Printf("wal: sealed segment %s (%d bytes)", filepath.Base(sealed), w.stableSize)
		}
	}
	w.file = nil
	w.activePath = ""
	w.stableSize = 0
	w.segMin, w.segMax = time.Time{}, time.Time{}
}

// Snapshot copies the stable prefix (complete frames only) of the active
// segment into TempDir and returns it as a queryable segment. ok is false
// when there is no active data. The copy is race-free: the writer only
// appends past stableSize, and the snapshot reads only up to it.
func (w *Writer) Snapshot() (Segment, bool, error) {
	w.mu.Lock()
	path, size := w.activePath, w.stableSize
	segMin, segMax := w.segMin, w.segMax
	w.mu.Unlock()

	if path == "" || size == 0 {
		return Segment{}, false, nil
	}

	w.snapMu.Lock()
	defer w.snapMu.Unlock()

	snapPath := filepath.Join(w.opts.TempDir, "wal-active-snapshot.jsonl.zst")
	seg := Segment{Path: snapPath, Start: segMin, End: segMax, Size: size}

	key := fmt.Sprintf("%s:%d", path, size)
	if w.snapKey == key {
		return seg, true, nil
	}

	src, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Sealed between capture and open; the data is reachable via the
			// sealed segment on the next query.
			return Segment{}, false, nil
		}
		return Segment{}, false, fmt.Errorf("failed to open active segment: %w", err)
	}
	defer src.Close()

	tmp, err := os.CreateTemp(w.opts.TempDir, "wal-snap-*")
	if err != nil {
		return Segment{}, false, fmt.Errorf("failed to create snapshot temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := io.CopyN(tmp, src, size); err != nil {
		tmp.Close()
		return Segment{}, false, fmt.Errorf("failed to copy snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return Segment{}, false, fmt.Errorf("failed to close snapshot: %w", err)
	}
	if err := os.Rename(tmp.Name(), snapPath); err != nil {
		return Segment{}, false, fmt.Errorf("failed to publish snapshot: %w", err)
	}

	w.snapKey = key
	return seg, true, nil
}
