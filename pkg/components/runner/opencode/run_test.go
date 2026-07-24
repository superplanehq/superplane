package opencode

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunScriptRedirectsOpenCodeHomeIntoTaskDir guards against the
// "PermissionDenied: FileSystem.open (.../opencode/log/opencode.log)" failure:
// opencode must write its log/state under the always-writable task dir, not a
// read-only $HOME. We execute the embedded run.js against a stub `opencode`
// that records the XDG dirs it was launched with and writes a log where
// opencode would.
func TestRunScriptRedirectsOpenCodeHomeIntoTaskDir(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available on PATH")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available on PATH")
	}

	taskDir := t.TempDir()
	binDir := t.TempDir()
	// A read-only HOME mimics the fleet-runner sandbox that triggered the bug.
	homeDir := t.TempDir()
	require.NoError(t, os.Chmod(homeDir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(homeDir, 0o700) })

	runJS := filepath.Join(taskDir, "run.js")
	require.NoError(t, os.WriteFile(runJS, []byte(runScript), 0o644))

	promptFile := filepath.Join(taskDir, "prompt.txt")
	require.NoError(t, os.WriteFile(promptFile, []byte("do the thing"), 0o644))

	resultFile := filepath.Join(taskDir, "result.json")
	envDump := filepath.Join(taskDir, "opencode-env.txt")

	// Stub opencode: record the XDG data dir it received, then write a log file
	// there exactly like the real CLI does before emitting JSON events.
	stub := "#!/bin/sh\n" +
		"printf 'XDG_DATA_HOME=%s\\n' \"$XDG_DATA_HOME\" > \"$OPENCODE_ENV_DUMP\"\n" +
		"printf 'HOME=%s\\n' \"$HOME\" >> \"$OPENCODE_ENV_DUMP\"\n" +
		"mkdir -p \"$XDG_DATA_HOME/opencode/log\"\n" +
		"echo started > \"$XDG_DATA_HOME/opencode/log/opencode.log\"\n" +
		"printf '%s\\n' '{\"type\":\"step_start\",\"sessionID\":\"ses_test\"}'\n" +
		"printf '%s\\n' '{\"type\":\"text\",\"text\":\"hello\"}'\n" +
		"printf '%s\\n' '{\"type\":\"step_finish\",\"reason\":\"stop\"}'\n" +
		"exit 0\n"
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "opencode"), []byte(stub), 0o755))

	cmd := exec.Command(node, runJS, promptFile, "anthropic/claude-sonnet-4-5")
	cmd.Env = []string{
		"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + homeDir,
		"SUPERPLANE_TASK_DIR=" + taskDir,
		"SUPERPLANE_RESULT_FILE=" + resultFile,
		"OPENCODE_ENV_DUMP=" + envDump,
	}
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "run.js failed: %s", out)

	dump, err := os.ReadFile(envDump)
	require.NoError(t, err)
	expectedData := filepath.Join(taskDir, "opencode-home", "data")
	assert.Contains(t, string(dump), "XDG_DATA_HOME="+expectedData,
		"opencode must run with its data dir inside the task dir")
	assert.NotContains(t, string(dump), "XDG_DATA_HOME="+homeDir,
		"opencode must not fall back to the read-only sandbox home")

	// The log opencode failed to open in the sandbox now lives under the task dir.
	assert.FileExists(t, filepath.Join(expectedData, "opencode", "log", "opencode.log"))

	session, err := os.ReadFile(filepath.Join(taskDir, "session_id"))
	require.NoError(t, err)
	assert.Equal(t, "ses_test", strings.TrimSpace(string(session)))

	resultRaw, err := os.ReadFile(resultFile)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(resultRaw, &result))
	assert.Equal(t, "ses_test", result["session_id"])
	assert.Equal(t, "hello", result["result"])
}
