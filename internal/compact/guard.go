package compact

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

// SetupGuard makes the compactor process die before it can take the server
// (or the whole container) down with it. Three layers:
//
//  1. DuckDB's own memory_limit + disk spill (configured per job in Run) —
//     the primary defense; compaction tolerates slowness, not OOM.
//  2. An RSS watchdog that exits the process before the kernel OOM killer
//     gets involved. This is what protects the pod on Kubernetes 1.28+ with
//     cgroup v2, where an OOM kill takes out the entire container
//     (memory.oom.group=1), making oom_score_adj useless.
//  3. oom_score_adj=1000 (Linux, best-effort) so that where per-process OOM
//     kills still happen, the compactor is the preferred victim.
func SetupGuard(memoryLimitMB int) {
	// Keep the Go side lean; DuckDB allocates outside the Go heap, so the
	// watchdog below is what actually bounds the process.
	debug.SetMemoryLimit(int64(memoryLimitMB) << 20)

	makeOOMPreferred()

	hardLimit := int64(memoryLimitMB)*2<<20 + 512<<20
	go watchRSS(hardLimit)
}

func watchRSS(hardLimitBytes int64) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		rss, ok := rssBytes()
		if !ok {
			return
		}
		if rss > hardLimitBytes {
			fmt.Fprintf(os.Stderr, "compact: RSS %d bytes exceeded hard limit %d bytes, exiting before OOM\n", rss, hardLimitBytes)
			os.Exit(3)
		}
	}
}
