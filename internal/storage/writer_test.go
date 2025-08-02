package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// WriterTestSuite is a test suite for the Writer component.
type WriterTestSuite struct {
	suite.Suite
	baseDir     string // Base temp dir for the suite
	parquetPath string
	dbPath      string
}

// SetupTest creates a temporary directory for each test.
func (s *WriterTestSuite) SetupTest() {
	baseDir, err := os.MkdirTemp("", "writer-test-*")
	require.NoError(s.T(), err)
	s.baseDir = baseDir
	s.parquetPath = filepath.Join(baseDir, "parquet")
	s.dbPath = filepath.Join(baseDir, "db", "writer.db")

	// NewWriter expects the directories to exist
	require.NoError(s.T(), os.MkdirAll(filepath.Dir(s.dbPath), 0755))
	require.NoError(s.T(), os.MkdirAll(s.parquetPath, 0755))
}

// TearDownTest cleans up the temporary directory.
func (s *WriterTestSuite) TearDownTest() {
	err := os.RemoveAll(s.baseDir)
	require.NoError(s.T(), err, "should be able to clean up temp dir")
}

// TestWriterSuite runs the entire Writer test suite.
func TestWriterSuite(t *testing.T) {
	suite.Run(t, new(WriterTestSuite))
}

// getEventCount directly queries the writer's DB file to get the number of events.
func (s *WriterTestSuite) getEventCount() int {
	db, err := sql.Open("duckdb", s.dbPath+"?access_mode=READ_ONLY")
	require.NoError(s.T(), err)
	defer db.Close()
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM kube_events").Scan(&count)
	require.NoError(s.T(), err)
	return count
}

// TestEventInsertionAndDeduplication verifies that events are inserted and duplicates are ignored.
func (s *WriterTestSuite) TestEventInsertionAndDeduplication() {
	s.T().Log("Goal: Verify events are collected, inserted, and duplicates are ignored.")
	writer, err := NewWriter(s.dbPath, s.parquetPath)
	require.NoError(s.T(), err)

	// 1. Insert a batch of unique events.
	eventCount := 10
	for i := 0; i < eventCount; i++ {
		evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID(fmt.Sprintf("uid-%d", i)), ResourceVersion: fmt.Sprintf("%d", i)}}
		require.NoError(s.T(), writer.AppendEvent(evt))
	}

	// 2. Insert a duplicate event (same resourceVersion).
	duplicateEvt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID("uid-duplicate"), ResourceVersion: "1"}}
	require.NoError(s.T(), writer.AppendEvent(duplicateEvt))

	// 3. Insert one more unique event.
	finalEvt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID("uid-final"), ResourceVersion: "99"}}
	require.NoError(s.T(), writer.AppendEvent(finalEvt))

	// Close the writer to ensure all events are flushed to disk.
	writer.Close()

	// Verify by connecting directly to the DB file.
	// Total should be 11 (10 initial + 1 final), with the duplicate ignored.
	require.Equal(s.T(), eventCount+1, s.getEventCount(), "should have inserted all unique events and ignored the duplicate")
}

// TestArchivingProcess verifies the database table to Parquet file archival process.
func (s *WriterTestSuite) TestArchivingProcess() {
	s.T().Log("Goal: Verify that events are archived to Parquet and the live table is cleared.")

	// --- Step 1: Create a writer, add an event, and close it to ensure data is on disk. ---
	writer1, err := NewWriter(s.dbPath, s.parquetPath)
	require.NoError(s.T(), err)

	evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID(s.T().Name()), ResourceVersion: "1"}}
	require.NoError(s.T(), writer1.AppendEvent(evt))

	// Close the writer to flush all events and release the DB file lock.
	writer1.Close()

	// Verify the event is persisted.
	require.Equal(s.T(), 1, s.getEventCount(), "should have 1 event saved on disk before archive")

	// --- Step 2: Create a new writer instance to run the archive process. ---
	writer2, err := NewWriter(s.dbPath, s.parquetPath)
	require.NoError(s.T(), err)

	// Manually trigger archive.
	err = writer2.Archive(context.Background())
	require.NoError(s.T(), err)

	// Close the second writer, which will wait for the archive goroutine to finish.
	writer2.Close()

	// --- Step 3: Verify the results. ---
	// 1. Verify the live table is now empty.
	require.Equal(s.T(), 0, s.getEventCount(), "live table should be empty after archive")

	// 2. Verify a parquet file was created.
	files, err := os.ReadDir(s.parquetPath)
	require.NoError(s.T(), err)
	foundParquet := false
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".parquet" {
			foundParquet = true
			break
		}
	}
	require.True(s.T(), foundParquet, "a .parquet file should have been created")
}

// TestRetentionPolicy verifies that old Parquet files are deleted when the size limit is exceeded.
func (s *WriterTestSuite) TestRetentionPolicy() {
	s.T().Log("Goal: Verify the retention policy correctly deletes the oldest files.")
	writer, err := NewWriter(s.dbPath, s.parquetPath)
	require.NoError(s.T(), err)
	defer writer.Close()

	// Create dummy parquet files with different timestamps.
	now := time.Now()
	// Oldest file, should be deleted.
	oldestFile := filepath.Join(s.parquetPath, fmt.Sprintf("events_%d_%d.parquet", now.Add(-3*time.Hour).Unix(), now.Add(-2*time.Hour).Unix()))
	// Newer file, should be kept.
	newerFile := filepath.Join(s.parquetPath, fmt.Sprintf("events_%d_%d.parquet", now.Add(-1*time.Hour).Unix(), now.Unix()))

	// Create files with some content to have a size.
	require.NoError(s.T(), os.WriteFile(oldestFile, make([]byte, 1024), 0644))
	require.NoError(s.T(), os.WriteFile(newerFile, make([]byte, 1024), 0644))

	// Enforce retention with a limit that allows only one file.
	err = writer.EnforceRetention(1500)
	require.NoError(s.T(), err)

	// Verify that the oldest file was deleted and the newer one remains.
	_, err = os.Stat(oldestFile)
	require.True(s.T(), os.IsNotExist(err), "oldest file should have been deleted")
	_, err = os.Stat(newerFile)
	require.NoError(s.T(), err, "newer file should still exist")
}

// TestWriterClose verifies that closing the writer flushes pending events.
func (s *WriterTestSuite) TestWriterClose() {
	s.T().Log("Goal: Verify Close() flushes events and stops accepting new ones.")
	writer, err := NewWriter(s.dbPath, s.parquetPath)
	require.NoError(s.T(), err)

	// Append an event right before closing.
	evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID(s.T().Name()), ResourceVersion: "1"}}
	require.NoError(s.T(), writer.AppendEvent(evt))

	// Close should block until the batch inserter is done.
	writer.Close()

	// Verify the event was flushed.
	require.Equal(s.T(), 1, s.getEventCount(), "event should have been flushed on close")

	// Verify that new events are rejected.
	err = writer.AppendEvent(evt)
	require.Error(s.T(), err, "should not accept events after closing")
	require.Equal(s.T(), "writer is closed", err.Error())
}
