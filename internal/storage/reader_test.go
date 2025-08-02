package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ReaderTestSuite is a test suite for the Reader component.
type ReaderTestSuite struct {
	suite.Suite
	tempDir string
	dbPath  string
}

// SetupTest creates a temporary directory for each test.
func (s *ReaderTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "reader-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir
	s.dbPath = filepath.Join(tempDir, "events.db")
}

// TearDownTest cleans up the temporary directory.
func (s *ReaderTestSuite) TearDownTest() {
	err := os.RemoveAll(s.tempDir)
	require.NoError(s.T(), err, "should be able to clean up temp dir")
}

// TestReaderSuite runs the entire Reader test suite.
func TestReaderSuite(t *testing.T) {
	suite.Run(t, new(ReaderTestSuite))
}

// helper function to create a dummy db file with some events.
func (s *ReaderTestSuite) createTestDBWithEvents(count int) {
	writer, err := NewWriter(s.dbPath, s.dbPath)
	require.NoError(s.T(), err)
	for i := 0; i < count; i++ {
		uid := fmt.Sprintf("%s-db-event-%d", s.T().Name(), i)
		resourceVersion := fmt.Sprintf("db-%d", i+1)
		evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid), ResourceVersion: resourceVersion}}
		require.NoError(s.T(), writer.AppendEvent(evt))
	}
	// Close the writer to ensure all events are flushed to disk.
	writer.Close()
	// Give a moment for the batch inserter to complete flushing
	time.Sleep(10 * time.Millisecond)
}

// helper function to create a dummy parquet file.
func (s *ReaderTestSuite) createTestParquetFile(count int, startTime, endTime time.Time) string {
	// Create a temporary, separate DB to generate the Parquet file.
	tempDBPath := filepath.Join(s.tempDir, fmt.Sprintf("temp_db_%d.db", startTime.Unix()))
	writer, err := NewWriter(tempDBPath, tempDBPath)
	require.NoError(s.T(), err)

	for i := 0; i < count; i++ {
		uid := fmt.Sprintf("%s-pq-event-%d", s.T().Name(), i)
		resourceVersion := fmt.Sprintf("pq-%d-%d", startTime.Unix(), i+1)
		evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid), ResourceVersion: resourceVersion}}
		require.NoError(s.T(), writer.AppendEvent(evt))
	}
	writer.Close()
	// Give a moment for the batch inserter to complete flushing
	time.Sleep(10 * time.Millisecond)

	// Re-open writer to archive the file.
	writer, err = NewWriter(tempDBPath, tempDBPath)
	require.NoError(s.T(), err)
	require.NoError(s.T(), writer.Archive(context.Background()))
	writer.Close()
	time.Sleep(1 * time.Second) // allow parquet conversion to finish

	// Find the created parquet file and rename it.
	files, err := os.ReadDir(s.tempDir)
	require.NoError(s.T(), err)

	var createdParquetPath string
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".parquet" && strings.HasPrefix(f.Name(), "events_") {
			createdParquetPath = filepath.Join(s.tempDir, f.Name())
			break
		}
	}
	require.NotEmpty(s.T(), createdParquetPath, "failed to find created parquet file")

	finalParquetPath := filepath.Join(s.tempDir, fmt.Sprintf("events_%d_%d.parquet", startTime.Unix(), endTime.Unix()))
	require.NoError(s.T(), os.Rename(createdParquetPath, finalParquetPath))

	// Clean up temp db files
	require.NoError(s.T(), os.Remove(tempDBPath))
	os.Remove(tempDBPath + ".wal") // .wal may not exist, so we ignore error.

	return finalParquetPath
}

// TestQueryOnlyFromDB verifies querying when only the live DB has data.
func (s *ReaderTestSuite) TestQueryOnlyFromDB() {
	s.T().Log("Goal: Verify query works correctly when data is only in the live DB.")
	s.createTestDBWithEvents(5)

	reader, err := NewReader(s.dbPath)
	require.NoError(s.T(), err)
	defer reader.Close()

	rows, result, err := reader.RangeQuery(context.Background(), "SELECT * FROM $events", time.Now().Add(-1*time.Hour), time.Now())
	require.NoError(s.T(), err)
	defer rows.Close()

	var rowCount int
	for rows.Next() {
		rowCount++
	}
	require.Equal(s.T(), 5, rowCount, "should read 5 events from the db")
	require.Empty(s.T(), result.Files, "no parquet files should be involved")
}

// TestQueryOnlyFromParquet verifies querying when only Parquet files have relevant data.
func (s *ReaderTestSuite) TestQueryOnlyFromParquet() {
	s.T().Log("Goal: Verify query works correctly when data is only in Parquet files.")
	s.createTestDBWithEvents(0) // Create an empty DB file.

	now := time.Now()
	p1Start, p1End := now.Add(-2*time.Hour), now.Add(-1*time.Hour)
	p2Start, p2End := now.Add(-4*time.Hour), now.Add(-3*time.Hour)
	p1Path := s.createTestParquetFile(10, p1Start, p1End)
	s.createTestParquetFile(20, p2Start, p2End) // This one is outside the query range.

	reader, err := NewReader(s.dbPath)
	require.NoError(s.T(), err)
	defer reader.Close()

	// Query a range that only covers the first parquet file.
	rows, result, err := reader.RangeQuery(context.Background(), "SELECT * FROM $events", p1Start.Add(-1*time.Minute), p1End.Add(1*time.Minute))
	require.NoError(s.T(), err)
	defer rows.Close()

	var rowCount int
	for rows.Next() {
		rowCount++
	}
	require.Equal(s.T(), 10, rowCount)
	require.Len(s.T(), result.Files, 1, "should only use one parquet file")
	require.Equal(s.T(), p1Path, result.Files[0].Path)
}

// TestQueryHybrid verifies querying data from both the live DB and Parquet files.
func (s *ReaderTestSuite) TestQueryHybrid() {
	s.T().Log("Goal: Verify query works correctly with data from both DB and Parquet.")
	s.createTestDBWithEvents(5) // 5 events in the live DB.

	now := time.Now()
	p1Start, p1End := now.Add(-2*time.Hour), now.Add(-1*time.Hour)
	s.createTestParquetFile(10, p1Start, p1End) // 10 events in parquet.

	reader, err := NewReader(s.dbPath)
	require.NoError(s.T(), err)
	defer reader.Close()

	// Query a range covering both.
	rows, result, err := reader.RangeQuery(context.Background(), "SELECT * FROM $events", p1Start.Add(-1*time.Minute), now)
	require.NoError(s.T(), err)
	defer rows.Close()

	var rowCount int
	for rows.Next() {
		rowCount++
	}
	require.Equal(s.T(), 15, rowCount, "should have a sum of events from db and parquet")
	require.Len(s.T(), result.Files, 1, "should have used one parquet file")
}
