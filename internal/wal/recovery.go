package wal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iwanhae/kabinet/internal/schema"
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

	var recoveryErrors []error
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), openSuffix) {
			continue
		}
		if err := recoverSegment(dir, entry.Name()); err != nil {
			// Leave the file in place so the next start can retry; segment
			// names are timestamped, so it cannot collide with new segments.
			log.Printf("wal: failed to recover segment %s: %v", entry.Name(), err)
			recoveryErrors = append(recoveryErrors, fmt.Errorf("recover %s: %w", entry.Name(), err))
		}
	}
	return errors.Join(recoveryErrors...)
}

// RecoverOpenSegments seals crash-left open WAL segments before a data-format
// migration snapshots its immutable inputs. NewWriter performs the same
// recovery during normal startup.
func RecoverOpenSegments(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return recoverOpenSegments(dir)
}

func recoverSegment(dir, name string) error {
	path := filepath.Join(dir, name)

	_, ok := parseOpenName(name)
	if !ok {
		return fmt.Errorf("unrecognized open segment name: %s", name)
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

	// Sealed names always carry the canonical timestamp envelope. Recovery
	// fails closed instead of inventing an ingestion-time fallback.
	min, max, err := eventTimeRange(decoded)
	if err != nil {
		return err
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
// JSONL, applying the canonical timestamp fallbacks from the schema projection.
func eventTimeRange(jsonl []byte) (min, max time.Time, err error) {
	type eventTimes struct {
		LastTimestamp  *time.Time `json:"lastTimestamp"`
		FirstTimestamp *time.Time `json:"firstTimestamp"`
		Series         *struct {
			LastObservedTime *time.Time `json:"lastObservedTime"`
		} `json:"series"`
		Metadata struct {
			CreationTimestamp *time.Time `json:"creationTimestamp"`
		} `json:"metadata"`
	}

	lineNumber := 0
	for line := range bytes.SplitSeq(jsonl, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		lineNumber++
		var et eventTimes
		if err := json.Unmarshal(line, &et); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("decode WAL event line %d: %w", lineNumber, err)
		}
		var series *time.Time
		if et.Series != nil {
			series = et.Series.LastObservedTime
		}
		ts, ok := schema.ResolveTimestamp(series, et.LastTimestamp, et.FirstTimestamp, et.Metadata.CreationTimestamp)
		if !ok {
			return time.Time{}, time.Time{}, fmt.Errorf("WAL event line %d: %w", lineNumber, ErrMissingTimestamp)
		}
		if min.IsZero() || ts.Before(min) {
			min = ts
		}
		if max.IsZero() || ts.After(max) {
			max = ts
		}
	}
	if min.IsZero() {
		return time.Time{}, time.Time{}, ErrMissingTimestamp
	}
	return min, max, nil
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
