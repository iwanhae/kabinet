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
	tempDir string
	dbPath  string
}

// SetupTest creates a temporary directory for each test.
func (s *WriterTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "writer-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir
	s.dbPath = filepath.Join(tempDir)
}

// TearDownTest cleans up the temporary directory.
func (s *WriterTestSuite) TearDownTest() {
	err := os.RemoveAll(s.tempDir)
	require.NoError(s.T(), err, "should be able to clean up temp dir")
}

// TestWriterSuite runs the entire Writer test suite.
func TestWriterSuite(t *testing.T) {
	suite.Run(t, new(WriterTestSuite))
}

func (s *WriterTestSuite) getEventCount() int {
	db, err := sql.Open("duckdb", s.dbPath+"/events.db?access_mode=READ_ONLY")
	require.NoError(s.T(), err)
	defer db.Close()
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM kube_events").Scan(&count)
	require.NoError(s.T(), err)
	return count
}

// TestEventInsertion verifies that events are correctly inserted into the database.
func (s *WriterTestSuite) TestEventInsertion() {
	s.T().Log("Goal: Verify events are collected and inserted into the DB.")
	writer, err := NewWriter(s.dbPath, s.dbPath)
	require.NoError(s.T(), err)

	// Append a known number of events with unique UIDs.
	eventCount := 10
	for i := 0; i < eventCount; i++ {
		uid := fmt.Sprintf("%s-%d", s.T().Name(), i)
		resourceVersion := fmt.Sprintf("%d", i+1)
		evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid), ResourceVersion: resourceVersion}}
		err := writer.AppendEvent(evt)
		require.NoError(s.T(), err)
	}
	// Close the writer, which will flush the remaining events.
	writer.Close()

	// Give a moment for the batch inserter to complete flushing
	time.Sleep(10 * time.Millisecond)

	// Verify by connecting directly to the DB file.
	require.Equal(s.T(), eventCount, s.getEventCount(), "should have inserted all events")
}

// TestArchivingProcess verifies the database table to Parquet file archival process.
func (s *WriterTestSuite) TestArchivingProcess() {
	s.T().Log("Goal: Verify that events are archived to Parquet and the live table is cleared.")
	writer, err := NewWriter(s.dbPath, s.dbPath)
	require.NoError(s.T(), err)

	// Insert some events.
	evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID(s.T().Name()), ResourceVersion: "1"}}
	require.NoError(s.T(), writer.AppendEvent(evt))
	writer.Close() // Close to flush.

	// Give a moment for the batch inserter to complete flushing and database connection to close
	time.Sleep(200 * time.Millisecond)

	// Re-open the writer to perform the archive.
	writer, err = NewWriter(s.dbPath, s.dbPath)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, s.getEventCount(), "should have 1 event before archive")

	// Manually trigger archive.
	err = writer.Archive(context.Background())
	require.NoError(s.T(), err)
	time.Sleep(1 * time.Second) // allow parquet conversion to finish

	// Close writer before checking the count to avoid connection conflicts
	writer.Close()
	time.Sleep(100 * time.Millisecond) // allow connection to close

	// 1. Verify the live table is now empty.
	require.Equal(s.T(), 0, s.getEventCount(), "live table should be empty after archive")

	// 2. Verify a parquet file was created.
	files, err := os.ReadDir(s.tempDir)
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

	// Create a separate directory for retention policy testing
	retentionTestDir, err := os.MkdirTemp("", "retention-test-*")
	require.NoError(s.T(), err)
	defer os.RemoveAll(retentionTestDir)

	// Create a separate database for this test
	tempDBPath := filepath.Join(retentionTestDir, "db")
	tempDataPath := filepath.Join(retentionTestDir, "data")
	writer, err := NewWriter(tempDBPath, tempDataPath)
	require.NoError(s.T(), err)
	defer writer.Close()

	// Create dummy parquet files with different timestamps.
	now := time.Now()
	// Oldest file, should be deleted.
	oldestFile := filepath.Join(tempDataPath, fmt.Sprintf("events_%d_%d.parquet", now.Add(-3*time.Hour).Unix(), now.Add(-2*time.Hour).Unix()))
	// Newer file, should be kept.
	newerFile := filepath.Join(tempDataPath, fmt.Sprintf("events_%d_%d.parquet", now.Add(-1*time.Hour).Unix(), now.Unix()))

	// Create files with some content to have a size.
	require.NoError(s.T(), os.WriteFile(oldestFile, make([]byte, 1024), 0644))
	require.NoError(s.T(), os.WriteFile(newerFile, make([]byte, 1024), 0644))

	err = writer.EnforceRetention(1024)
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
	writer, err := NewWriter(s.dbPath, s.dbPath)
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
