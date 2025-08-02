package storage

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
)

// connectionManager intelligently manages read-only database connections.
// - It creates a connection on the first request.
// - After a configured duration, it pre-warms the next connection in the background.
// - On the next request, it hot-swaps to the new connection instantly.
// - All internal state is thread-safe.
type connectionManager struct {
	mu          sync.Mutex
	dbPath      string
	warmupAfter time.Duration // Wait duration before starting to warm up a new connection.

	currentConn *sql.DB   // The connection currently used for serving requests.
	nextConn    *sql.DB   // The next connection being warmed up in the background.
	isWarmingUp bool      // A flag to prevent duplicate warm-up routines.
	createdAt   time.Time // Timestamp of when the current connection was created.

	// Fields for graceful shutdown.
	closeCh chan struct{}
	wg      sync.WaitGroup
}

// newConnectionManager creates a new connection manager.
func newConnectionManager(dbPath string, warmupAfter time.Duration) *connectionManager {
	return &connectionManager{
		dbPath:      dbPath,
		warmupAfter: warmupAfter,
		closeCh:     make(chan struct{}),
	}
}

// Get returns a ready-to-use database connection.
// This function is designed to return quickly.
func (cm *connectionManager) Get() (*sql.DB, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Case 1: First call after system startup.
	if cm.currentConn == nil {
		log.Println("conn_manager: creating initial connection...")
		conn, err := cm.createConnection() // The first connection is created synchronously.
		if err != nil {
			return nil, err
		}
		cm.currentConn = conn
		cm.createdAt = time.Now()
		cm.startWarmupRoutine() // Start the first warm-up routine.
		return cm.currentConn, nil
	}

	// Case 2: A new connection (nextConn) is ready from background warm-up.
	if cm.nextConn != nil {
		log.Println("conn_manager: hot-swapping to warmed-up connection.")
		staleConn := cm.currentConn
		cm.currentConn = cm.nextConn
		cm.createdAt = time.Now()
		cm.nextConn = nil
		cm.isWarmingUp = false

		// Close the old (stale) connection quietly in the background.
		cm.wg.Add(1)
		go func() {
			defer cm.wg.Done()
			staleConn.Close()
			log.Println("conn_manager: closed stale connection in background.")
		}()

		cm.startWarmupRoutine() // Start the next warm-up cycle.
		return cm.currentConn, nil
	}

	// Case 3: Warm-up is in progress or it's not time to warm up yet.
	// In this case, just return the current connection.
	if !cm.isWarmingUp && time.Since(cm.createdAt) > cm.warmupAfter {
		cm.startWarmupRoutine()
	}

	return cm.currentConn, nil
}

// startWarmupRoutine starts a background goroutine that acts as a warm-up timer.
func (cm *connectionManager) startWarmupRoutine() {
	cm.isWarmingUp = true
	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()
		select {
		case <-time.After(cm.warmupAfter):
			log.Println("conn_manager: warmup triggered. preparing next connection...")
			newConn, err := cm.createConnection()

			cm.mu.Lock()
			defer cm.mu.Unlock()

			if err != nil {
				log.Printf("conn_manager: failed to warm up new connection: %v", err)
				cm.isWarmingUp = false // Reset the flag for the next attempt.
				return
			}

			log.Println("conn_manager: new connection is warmed up and ready.")
			cm.nextConn = newConn

		case <-cm.closeCh: // Exit immediately if Close is called.
			return
		}
	}()
}

// createConnection opens the database file and verifies the actual connection with Ping().
func (cm *connectionManager) createConnection() (*sql.DB, error) {
	conn, err := sql.Open("duckdb", cm.dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("failed to open db for new connection: %w", err)
	}
	// Use Ping() to force a connection and check for errors early.
	if err := conn.Ping(); err != nil {
		conn.Close() // Prevent resource leaks on failure.
		return nil, fmt.Errorf("failed to ping new connection: %w", err)
	}
	return conn, nil
}

// Close safely closes all managed connections and shuts down background routines.
func (cm *connectionManager) Close() {
	log.Println("conn_manager: shutting down...")
	close(cm.closeCh) // Signal all background goroutines to terminate.
	cm.wg.Wait()      // Wait for all goroutines to finish.

	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.currentConn != nil {
		cm.currentConn.Close()
	}
	if cm.nextConn != nil {
		cm.nextConn.Close()
	}
	log.Println("conn_manager: shutdown complete.")
}
