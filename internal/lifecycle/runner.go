package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/iwanhae/kabinet/internal/compact"
)

// runCompactor executes one job in the compactor subprocess: job JSON on
// stdin, result JSON on stdout, logs passed through on stderr.
func (m *Manager) runCompactor(ctx context.Context, job compact.Job) (*compact.Result, error) {
	path, err := compactorPath()
	if err != nil {
		return nil, err
	}

	jobJSON, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job: %w", err)
	}

	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, path)
	cmd.Stdin = bytes.NewReader(jobJSON)
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("compactor failed (%s job, %d inputs): %w", job.Mode, len(job.Inputs), err)
	}

	var result compact.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse compactor result: %w", err)
	}
	return &result, nil
}

// compactorPath locates the compactor binary: explicit env var, next to the
// server binary, PATH, then the Docker install location.
func compactorPath() (string, error) {
	if p := os.Getenv("KABINET_COMPACTOR_PATH"); p != "" {
		return p, nil
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "compactor")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	if p, err := exec.LookPath("compactor"); err == nil {
		return p, nil
	}
	if _, err := os.Stat("/usr/local/bin/compactor"); err == nil {
		return "/usr/local/bin/compactor", nil
	}
	log.Println("lifecycle: compactor binary not found (set KABINET_COMPACTOR_PATH)")
	return "", fmt.Errorf("compactor binary not found in KABINET_COMPACTOR_PATH, next to server, PATH, or /usr/local/bin")
}
