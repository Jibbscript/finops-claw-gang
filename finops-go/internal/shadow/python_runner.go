package shadow

import (
	"context"
	"fmt"
	"os/exec"
)

// PythonRunner invokes the Python CLI and captures JSON output.
type PythonRunner struct {
	PythonPath  string
	FixturesDir string
}

// Run executes the Python pipeline on the given fixtures and returns JSON output.
func (r *PythonRunner) Run(ctx context.Context, service string, delta float64) ([]byte, error) {
	args := []string{
		"-m", "finops_desk.cli",
		"--fixtures", r.FixturesDir,
		"--service", service,
		"--delta", fmt.Sprintf("%.0f", delta),
		"--json-output",
	}
	cmd := exec.CommandContext(ctx, r.PythonPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("python CLI failed: %s\n%s", err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("python CLI: %w", err)
	}
	return out, nil
}
