//go:build linux

package compact

import (
	"bytes"
	"os"
	"strconv"
)

// makeOOMPreferred asks the kernel to pick this process first when the OOM
// killer selects a per-process victim. Raising the score needs no privileges.
func makeOOMPreferred() {
	_ = os.WriteFile("/proc/self/oom_score_adj", []byte("1000"), 0o644)
}

// rssBytes reads the resident set size from /proc/self/statm.
func rssBytes() (int64, bool) {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, false
	}
	fields := bytes.Fields(data)
	if len(fields) < 2 {
		return 0, false
	}
	pages, err := strconv.ParseInt(string(fields[1]), 10, 64)
	if err != nil {
		return 0, false
	}
	return pages * int64(os.Getpagesize()), true
}
