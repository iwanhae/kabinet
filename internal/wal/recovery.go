package wal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

// recoverOpenSegments seals any ".open" segment left behind by a crash. The
// decodable prefix (complete zstd frames, complete JSONL lines) is kept and
// re-encoded into a sealed segment; a torn tail is dropped.
func recoverOpenSegments(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read wal directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), openSuffix) {
			continue
		}
		if err := recoverSegment(dir, entry.Name()); err != nil {
			// Leave the file in place so the next start can retry; segment
			// names are timestamped, so it cannot collide with new segments.
			log.Printf("wal: failed to recover segment %s: %v", entry.Name(), err)
		}
	}
	return nil
}

func recoverSegment(dir, name string) error {
	path := filepath.Join(dir, name)

	start, ok := parseOpenName(name)
	if !ok {
		return fmt.Errorf("unrecognized open segment name: %s", name)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat segment: %w", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read segment: %w", err)
	}

	decoded := decodeValidPrefix(raw)
	if len(decoded) == 0 {
		log.Printf("wal: removing empty/corrupt open segment %s", name)
		return os.Remove(path)
	}

	enc, err := zstd.NewWriter(nil, zstd.WithEncoderConcurrency(1))
	if err != nil {
		return fmt.Errorf("failed to create zstd encoder: %w", err)
	}
	frame := enc.EncodeAll(decoded, nil)
	enc.Close()

	tmp, err := os.CreateTemp(dir, "recover-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(frame); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write recovered segment: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close recovered segment: %w", err)
	}

	// Sealed names carry the event-time range for query pruning; fall back to
	// segment-creation/mtime when no line has a parseable timestamp.
	min, max, ok := eventTimeRange(decoded)
	if !ok {
		min, max = start, info.ModTime()
	}
	sealed := filepath.Join(dir, sealedName(min, max))
	if err := os.Rename(tmp.Name(), sealed); err != nil {
		return fmt.Errorf("failed to seal recovered segment: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove open segment after recovery: %w", err)
	}

	log.Printf("wal: recovered open segment %s -> %s (%d bytes jsonl)", name, filepath.Base(sealed), len(decoded))
	return nil
}

// eventTimeRange extracts the effective event-time min/max from raw K8s Event
// JSONL, applying the same lastTimestamp fallbacks as the schema projection.
func eventTimeRange(jsonl []byte) (min, max time.Time, ok bool) {
	type eventTimes struct {
		LastTimestamp  *time.Time `json:"lastTimestamp"`
		FirstTimestamp *time.Time `json:"firstTimestamp"`
		Metadata       struct {
			CreationTimestamp *time.Time `json:"creationTimestamp"`
		} `json:"metadata"`
	}

	for line := range bytes.SplitSeq(jsonl, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var et eventTimes
		if err := json.Unmarshal(line, &et); err != nil {
			continue
		}
		ts := et.LastTimestamp
		if ts == nil {
			ts = et.FirstTimestamp
		}
		if ts == nil {
			ts = et.Metadata.CreationTimestamp
		}
		if ts == nil || ts.IsZero() {
			continue
		}
		if !ok || ts.Before(min) {
			min = *ts
		}
		if !ok || ts.After(max) {
			max = *ts
		}
		ok = true
	}
	return min, max, ok
}

// decodeValidPrefix decompresses as much of raw as possible and trims the
// result to the last complete JSONL line.
func decodeValidPrefix(raw []byte) []byte {
	dec, err := zstd.NewReader(bytes.NewReader(raw), zstd.WithDecoderConcurrency(1))
	if err != nil {
		return nil
	}
	defer dec.Close()

	// io.ReadAll stops at the first decode error, returning everything
	// decoded so far — exactly the valid prefix we want.
	decoded, _ := io.ReadAll(dec)

	if i := bytes.LastIndexByte(decoded, '\n'); i >= 0 {
		return decoded[:i+1]
	}
	return nil
}
