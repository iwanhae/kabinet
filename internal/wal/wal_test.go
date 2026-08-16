package wal

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func testEvent(uid, rv string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-event-" + rv,
			Namespace:       "default",
			UID:             types.UID("00000000-0000-0000-0000-00000000000" + uid),
			ResourceVersion: rv,
		},
		Reason:        "Testing",
		Message:       "test event",
		LastTimestamp: metav1.Now(),
	}
}

func decodeSegment(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read segment: %v", err)
	}
	dec, err := zstd.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("failed to create decoder: %v", err)
	}
	defer dec.Close()
	decoded, err := io.ReadAll(dec)
	if err != nil {
		t.Fatalf("failed to decode segment: %v", err)
	}
	lines := bytes.Split(bytes.TrimSuffix(decoded, []byte("\n")), []byte("\n"))
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = string(l)
	}
	return out
}

func TestWriterFlushAndSeal(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	w, err := NewWriter(ctx, Options{
		Dir:           dir,
		TempDir:       t.TempDir(),
		FlushInterval: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("failed to create writer: %v", err)
	}

	for i := range 3 {
		if err := w.Append(ctx, testEvent("1", string(rune('a'+i)))); err != nil {
			t.Fatalf("failed to append: %v", err)
		}
	}
	time.Sleep(100 * time.Millisecond) // let at least one flush happen

	// Active segment must be snapshotable before sealing.
	snap, ok, err := w.Snapshot()
	if err != nil || !ok {
		t.Fatalf("expected active snapshot, got ok=%v err=%v", ok, err)
	}
	if lines := decodeSegment(t, snap.Path); len(lines) != 3 {
		t.Fatalf("snapshot: expected 3 lines, got %d", len(lines))
	}

	cancel()
	w.Wait()

	sealed, err := ListSealed(dir)
	if err != nil {
		t.Fatalf("failed to list sealed: %v", err)
	}
	if len(sealed) != 1 {
		t.Fatalf("expected 1 sealed segment, got %d", len(sealed))
	}
	if lines := decodeSegment(t, sealed[0].Path); len(lines) != 3 {
		t.Fatalf("sealed: expected 3 lines, got %d", len(lines))
	}
}

func TestRecoveryTruncatedTail(t *testing.T) {
	dir := t.TempDir()

	enc, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatalf("failed to create encoder: %v", err)
	}
	defer enc.Close()

	goodFrame := enc.EncodeAll([]byte("{\"a\":1}\n{\"a\":2}\n"), nil)
	tornFrame := enc.EncodeAll([]byte("{\"a\":3}\n"), nil)
	tornFrame = tornFrame[:len(tornFrame)-4] // simulate a crash mid-write

	start := time.Now().Add(-time.Minute)
	openPath := filepath.Join(dir, openName(start))
	if err := os.WriteFile(openPath, append(append([]byte{}, goodFrame...), tornFrame...), 0o644); err != nil {
		t.Fatalf("failed to write open segment: %v", err)
	}

	if err := recoverOpenSegments(dir); err != nil {
		t.Fatalf("recovery failed: %v", err)
	}

	if _, err := os.Stat(openPath); !os.IsNotExist(err) {
		t.Fatalf("open segment should have been removed after recovery")
	}
	sealed, err := ListSealed(dir)
	if err != nil {
		t.Fatalf("failed to list sealed: %v", err)
	}
	if len(sealed) != 1 {
		t.Fatalf("expected 1 recovered segment, got %d", len(sealed))
	}
	lines := decodeSegment(t, sealed[0].Path)
	if len(lines) != 2 || lines[0] != `{"a":1}` || lines[1] != `{"a":2}` {
		t.Fatalf("expected the 2 intact lines, got %v", lines)
	}
	if !sealed[0].Start.Equal(start.Truncate(time.Millisecond)) {
		t.Fatalf("recovered segment start mismatch: %v vs %v", sealed[0].Start, start)
	}
}

func TestRecoveryEmptySegment(t *testing.T) {
	dir := t.TempDir()
	openPath := filepath.Join(dir, openName(time.Now()))
	if err := os.WriteFile(openPath, nil, 0o644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := recoverOpenSegments(dir); err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if _, err := os.Stat(openPath); !os.IsNotExist(err) {
		t.Fatalf("empty open segment should have been removed")
	}
}
