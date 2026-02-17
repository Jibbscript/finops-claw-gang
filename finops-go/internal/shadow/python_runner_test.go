package shadow

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPythonScript returns the path to a temporary script that echoes fixture JSON.
func mockPythonScript(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "mock_python.sh")
	content := `#!/bin/sh
echo '{"triage":{"category":"deploy_related","severity":"medium","confidence":0.7},"analysis":{"confidence":0.4},"approval":{"status":"auto_approved","details":"auto-approved; max risk=low"}}'
`
	err := os.WriteFile(script, []byte(content), 0755)
	require.NoError(t, err)
	return script
}

func TestPythonRunner_MockScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}

	script := mockPythonScript(t)

	runner := &PythonRunner{
		PythonPath:  script,
		FixturesDir: "/tmp/unused",
	}

	out, err := runner.Run(context.Background(), "EC2", 750)
	require.NoError(t, err)
	assert.Contains(t, string(out), "deploy_related")
	assert.Contains(t, string(out), "auto_approved")
}

func TestPythonRunner_BadPath(t *testing.T) {
	runner := &PythonRunner{
		PythonPath:  "/nonexistent/python",
		FixturesDir: "/tmp",
	}

	_, err := runner.Run(context.Background(), "EC2", 750)
	assert.Error(t, err)
}
