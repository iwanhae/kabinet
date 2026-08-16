//go:build !linux

package compact

// makeOOMPreferred is a no-op outside Linux.
func makeOOMPreferred() {}

// rssBytes is unavailable outside Linux; the watchdog disables itself.
func rssBytes() (int64, bool) {
	return 0, false
}
