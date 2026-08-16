// The compactor is a short-lived subprocess spawned by the server's lifecycle
// manager. It reads one JSON Job from stdin, converts/merges event data into
// Parquet with bounded memory, and writes a JSON Result to stdout.
//
//	echo '{"mode":"convert","inputs":[...],"output":"...","tempDir":"...","memoryLimitMB":512}' | compactor
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iwanhae/kabinet/internal/compact"
)

func main() {
	log.SetPrefix("compactor: ")

	var job compact.Job
	if err := json.NewDecoder(os.Stdin).Decode(&job); err != nil {
		log.Fatalf("failed to decode job from stdin: %v", err)
	}
	if job.MemoryLimitMB <= 0 {
		job.MemoryLimitMB = 512
	}

	compact.SetupGuard(job.MemoryLimitMB)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	result, err := compact.Run(ctx, job)
	if err != nil {
		log.Fatalf("job failed: %v", err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		log.Fatalf("failed to encode result: %v", err)
	}
}
