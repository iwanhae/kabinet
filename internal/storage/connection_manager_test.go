package storage

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ConnectionManagerTestSuite is a test suite for the connectionManager.
type ConnectionManagerTestSuite struct {
	suite.Suite
	tempDir string
	dbPath  string
}

// SetupTest creates a temporary directory and a valid, empty DuckDB file for each test.
func (s *ConnectionManagerTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "cm-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir
	s.dbPath = filepath.Join(tempDir, "test.db")
	// Create a valid, empty DuckDB file by initializing and immediately closing a writer.
	writer, err := NewWriter(s.dbPath, s.dbPath)
	require.NoError(s.T(), err)
	writer.Close()
}

// TearDownTest cleans up the temporary directory after each test.
func (s *ConnectionManagerTestSuite) TearDownTest() {
	err := os.RemoveAll(s.tempDir)
	require.NoError(s.T(), err, "should be able to clean up temp dir")
}

// TestConnectionManagerSuite runs the entire test suite.
func TestConnectionManagerSuite(t *testing.T) {
	suite.Run(t, new(ConnectionManagerTestSuite))
}

// TestInitialConnection verifies that the first call to Get() creates a new connection.
func (s *ConnectionManagerTestSuite) TestInitialConnection() {
	s.T().Log("Goal: Verify the first Get() call creates a connection.")

	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	cm := newConnectionManager(s.dbPath, 5*time.Second)
	defer cm.Close()

	// First call to Get
	conn1, err := cm.Get()
	require.NoError(s.T(), err)
	require.NotNil(s.T(), conn1)

	logs := logBuf.String()
	require.Contains(s.T(), logs, "conn_manager: creating initial connection...")

	// Second immediate call should return the same connection
	conn2, err := cm.Get()
	require.NoError(s.T(), err)
	require.Same(s.T(), conn1, conn2, "second call should return the same connection instance")
}

// TestWarmupAndHotSwap verifies the background warm-up and hot-swap mechanism.
func (s *ConnectionManagerTestSuite) TestWarmupAndHotSwap() {
	s.T().Log("Goal: Verify that a new connection is warmed up and hot-swapped after the TTL.")

	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	// Use a very short warmup time for testing purposes.
	warmupTime := 100 * time.Millisecond
	cm := newConnectionManager(s.dbPath, warmupTime)
	defer cm.Close()

	// 1. Initial Get
	conn1, err := cm.Get()
	require.NoError(s.T(), err)
	require.NotNil(s.T(), conn1)

	// 2. Wait for longer than the warmup time for the background routine to complete.
	time.Sleep(warmupTime * 3)

	// 3. Second Get should trigger the hot-swap.
	conn2, err := cm.Get()
	require.NoError(s.T(), err)
	require.NotNil(s.T(), conn2)

	// 4. Give additional time for the background goroutine to close the stale connection
	time.Sleep(50 * time.Millisecond)

	// 5. Verification
	require.NotSame(s.T(), conn1, conn2, "connection should have been hot-swapped")

	logs := logBuf.String()
	require.Contains(s.T(), logs, "warmup triggered. preparing next connection...")
	require.Contains(s.T(), logs, "new connection is warmed up and ready.")
	require.Contains(s.T(), logs, "hot-swapping to warmed-up connection.")
	require.Contains(s.T(), logs, "closed stale connection in background.")
}

// TestNoHotSwapWithinTTL verifies that connections are not swapped before the warmup time expires.
func (s *ConnectionManagerTestSuite) TestNoHotSwapWithinTTL() {
	s.T().Log("Goal: Verify that connections are reused if called within the warmup TTL.")

	cm := newConnectionManager(s.dbPath, 1*time.Second)
	defer cm.Close()

	conn1, err := cm.Get()
	require.NoError(s.T(), err)

	// Call Get again immediately.
	conn2, err := cm.Get()
	require.NoError(s.T(), err)
	require.Same(s.T(), conn1, conn2, "should return the same connection instance")
}

// TestGracefulShutdown verifies that the manager and its goroutines shut down cleanly.
func (s *ConnectionManagerTestSuite) TestGracefulShutdown() {
	s.T().Log("Goal: Verify the connection manager shuts down without deadlocks.")

	var wg sync.WaitGroup
	cm := newConnectionManager(s.dbPath, 50*time.Millisecond)

	// Start a goroutine that mimics a user of the connection manager.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := cm.Get()
		require.NoError(s.T(), err)
		time.Sleep(100 * time.Millisecond) // wait for warmup to potentially start
	}()

	wg.Wait() // Wait for the Get call to complete.

	// Now, close the manager. This should not block.
	closed := make(chan struct{})
	go func() {
		cm.Close()
		close(closed)
	}()

	select {
	case <-closed:
		// Success
	case <-time.After(1 * time.Second):
		s.T().Fatal("Close() timed out, indicating a possible deadlock.")
	}
}
