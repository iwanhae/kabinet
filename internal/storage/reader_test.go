package storage

import (
	"context"
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

// ReaderTestSuite is a test suite for the Reader component.
type ReaderTestSuite struct {
	suite.Suite
	baseDir     string
	parquetPath string
	dbPath      string
}

// SetupTest creates a temporary directory structure for each test.
func (s *ReaderTestSuite) SetupTest() {
	baseDir, err := os.MkdirTemp("", "reader-test-*")
	require.NoError(s.T(), err)
	s.baseDir = baseDir
	s.parquetPath = filepath.Join(baseDir, "parquet")
	dbDir := filepath.Join(baseDir, "db")
	s.dbPath = filepath.Join(dbDir, "writer.db")
	require.NoError(s.T(), os.MkdirAll(s.parquetPath, 0755))
	require.NoError(s.T(), os.MkdirAll(dbDir, 0755))
}

// TearDownTest cleans up the temporary directory.
func (s *ReaderTestSuite) TearDownTest() {
	err := os.RemoveAll(s.baseDir)
	require.NoError(s.T(), err, "should be able to clean up temp dir")
}

// TestReaderSuite runs the entire Reader test suite.
func TestReaderSuite(t *testing.T) {
	suite.Run(t, new(ReaderTestSuite))
}

// createTestDBWithEvents is a helper to create a writer.db file with specified number of events.
// It creates a real Writer, adds events, and closes it to simulate the file state.
func (s *ReaderTestSuite) createTestDBWithEvents(count int) {
	writer, err := NewWriter(s.dbPath, s.parquetPath)
	require.NoError(s.T(), err)
	for i := 0; i < count; i++ {
		uid := fmt.Sprintf("db-event-%d", i)
		resourceVersion := fmt.Sprintf("db-%d", i+1)
		evt := &corev1.Event{ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid), ResourceVersion: resourceVersion}}
		require.NoError(s.T(), writer.AppendEvent(evt))
	}
	writer.Close()
}

// createTestParquetFile creates a dummy parquet file with specified number of events.
func (s *ReaderTestSuite) createTestParquetFile(count int, startTime, endTime time.Time) string {
	// To create a parquet file, we use a temporary writer instance.
	tempDBPath := filepath.Join(s.baseDir, fmt.Sprintf("temp_db_%d.db", startTime.Unix()))
	writer, err := NewWriter(tempDBPath, s.parquetPath)
	require.NoError(s.T(), err)

	for i := 0; i < count; i++ {
		uid := fmt.Sprintf("pq-event-%d", i)
		resourceVersion := fmt.Sprintf("pq-%d-%d", startTime.Unix(), i+1)
		evt := &corev1.Event{
			ObjectMeta: metav1.ObjectMeta{UID: types.UID(uid), ResourceVersion: resourceVersion},
			LastTimestamp: metav1.NewTime(startTime.Add(time.Duration(i) * time.Second)),
		}
		require.NoError(s.T(), writer.AppendEvent(evt))
	}

	// Wait for events to be flushed
	time.Sleep(1500 * time.Millisecond)

	// Archive and close
	require.NoError(s.T(), writer.Archive(context.Background()))
	writer.Close()

	// Find the created parquet file.
	files, err := os.ReadDir(s.parquetPath)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), files, "a parquet file should have been created")

	// Rename the file to have a predictable name for testing.
	createdParquetPath := filepath.Join(s.parquetPath, files[0].Name())
	finalParquetPath := filepath.Join(s.parquetPath, fmt.Sprintf("events_%d_%d.parquet", startTime.Unix(), endTime.Unix()))
	require.NoError(s.T(), os.Rename(createdParquetPath, finalParquetPath))

	require.NoError(s.T(), os.Remove(tempDBPath))
	os.Remove(tempDBPath + ".wal")

	return finalParquetPath
}

func (s *ReaderTestSuite) TestQueryOnlyFromDB() {
	s.T().Log("Goal: Verify query works correctly when data is only in the live DB.")
	s.createTestDBWithEvents(5)

	reader, err := NewReader(s.dbPath, s.parquetPath)
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

func (s *ReaderTestSuite) TestQueryOnlyFromParquet() {
	s.T().Log("Goal: Verify query works correctly when data is only in Parquet files.")
	// Create an empty DB file.
	s.createTestDBWithEvents(0)

	now := time.Now()
	p1Start, p1End := now.Add(-2*time.Hour), now.Add(-1*time.Hour)
	p1Path := s.createTestParquetFile(10, p1Start, p1End)

	reader, err := NewReader(s.dbPath, s.parquetPath)
	require.NoError(s.T(), err)
	defer reader.Close()

	// Query a range that only covers the parquet file.
	rows, result, err := reader.RangeQuery(context.Background(), "SELECT * FROM $events", p1Start.Add(-1*time.Minute), p1End.Add(1*time.Minute))
	require.NoError(s.T(), err)
	defer rows.Close()

	var rowCount int
	for rows.Next() {
		rowCount++
	}
	require.Equal(s.T(), 10, rowCount)
	require.Len(s.T(), result.Files, 1, "should only use one parquet file")
	require.Equal(s.T(), p1Path, result.Files[0])
}

func (s *ReaderTestSuite) TestQueryHybrid() {
	s.T().Log("Goal: Verify query works correctly with data from both DB and Parquet.")
	s.createTestDBWithEvents(5) // 5 events in the live DB.

	now := time.Now()
	p1Start, p1End := now.Add(-2*time.Hour), now.Add(-1*time.Hour)
	s.createTestParquetFile(10, p1Start, p1End) // 10 events in parquet.

	reader, err := NewReader(s.dbPath, s.parquetPath)
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

func (s *ReaderTestSuite) TestQueryNoData() {
	s.T().Log("Goal: Verify query returns no results when no data sources exist.")
	// Don't create any db or parquet files.

	reader, err := NewReader(s.dbPath, s.parquetPath)
	require.NoError(s.T(), err)
	defer reader.Close()

	_, _, err = reader.RangeQuery(context.Background(), "SELECT * FROM $events", time.Now().Add(-1*time.Hour), time.Now())
	// This should fail because ATTACH will fail and there are no parquet files.
	require.Error(s.T(), err)
}
