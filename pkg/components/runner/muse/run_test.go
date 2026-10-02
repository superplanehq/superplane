package muse

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

func TestMuseExecArgsUsesModelSessionAndCustomBaseURL(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(map[string]any{
		"model":     "custom/muse-spark-1.3",
		"thinking":  "high",
		"workspace": "/tmp/repo",
		"sessionID": "session-1",
		"env":       map[string]string{"CUSTOM_LLM_BASE_URL": "https://api.meta.ai/v1"},
	})
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { museExecArgs } = require(process.argv[1]); process.stdout.write(JSON.stringify(museExecArgs(JSON.parse(process.argv[2]))));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var args []string
	require.NoError(t, json.Unmarshal(out, &args))
	assert.Equal(t, []string{
		"exec", "--json", "--disable-approval", "--trust-workspace", "--user-input-auto-resolve",
		"--workspace", "/tmp/repo", "--session-id", "session-1", "--model", "muse-spark-1.3",
		"--reasoning-effort", "high", "--base-url", "https://api.meta.ai/v1",
	}, args)
}

func TestUsageFromPayloadMapsMuseTokenFields(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload := `{"usage":{"promptTokens":12,"completionTokens":7,"cachedPromptTokens":3,"reasoningTokens":2}}`
	cmd := exec.Command("node", "-e", `const { usageFromPayload } = require(process.argv[1]); process.stdout.write(JSON.stringify(usageFromPayload(JSON.parse(process.argv[2]))));`, script, payload)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var usage map[string]int
	require.NoError(t, json.Unmarshal(out, &usage))
	assert.Equal(t, 12, usage["input_tokens"])
	assert.Equal(t, 7, usage["output_tokens"])
	assert.Equal(t, 3, usage["cache_read_input_tokens"])
	assert.Equal(t, 2, usage["reasoning_tokens"])
}

func TestBuildMuseSettingsMapsWorkspaceMCP(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	taskDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "workspace_mcp.json"), []byte(`{"servers":[{"name":"browser","url":"https://mcp.example/browser","headers":{"Authorization":"Bearer token"}}]}`), 0o644))
	payload, err := json.Marshal(map[string]any{"taskDir": taskDir, "env": map[string]string{}})
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { buildMuseSettings } = require(process.argv[1]); const p = JSON.parse(process.argv[2]); process.stdout.write(JSON.stringify(buildMuseSettings(p.taskDir, p.env)));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var settings map[string]any
	require.NoError(t, json.Unmarshal(out, &settings))
	mcp := settings["mcp_servers"].(map[string]any)
	browser := mcp["browser"].(map[string]any)
	assert.Equal(t, "streamable_http", browser["transport"])
	assert.Equal(t, "https://mcp.example/browser", browser["url"])
}

func TestRunPromptMapsCredentialsAndPersistsMuseSession(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("the fake Muse executable uses a POSIX shell")
	}
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	taskDir := t.TempDir()
	binDir := t.TempDir()
	promptFile := filepath.Join(taskDir, "prompt.txt")
	resultFile := filepath.Join(taskDir, "result.json")
	require.NoError(t, os.WriteFile(promptFile, []byte("Fix the tests."), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "prompt_count"), []byte("0\n"), 0o644))
	fakeMuse := `#!/bin/sh
printf '%s\n' "$@" > "$SUPERPLANE_TASK_DIR/muse_args"
printf '%s' "$META_API_KEY" > "$SUPERPLANE_TASK_DIR/meta_key"
printf '%s\n' \
'{"stream":{"kind":"session","id":"session-123"},"payload_type":"run.output.delta","payload":{"text":"Done."}}' \
'{"stream":{"kind":"session","id":"session-123"},"payload_type":"turn.completed","payload":{"modelId":"muse-spark-1.3","usage":{"promptTokens":12,"completionTokens":7}}}' \
'{"stream":{"kind":"session","id":"session-123"},"payload_type":"run.terminal.completed","payload":{"terminal":"completed","text":"Done."}}'
`
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "muse"), []byte(fakeMuse), 0o755))

	cmd := exec.Command("node", script, promptFile, "custom/muse-spark-1.3", "medium")
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_RESULT_FILE="+resultFile,
		"CUSTOM_LLM_API_KEY=meta-token",
		"CUSTOM_LLM_BASE_URL=https://api.meta.ai/v1",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "Done.")
	assert.Equal(t, "meta-token", strings.TrimSpace(string(mustReadMuseFile(t, filepath.Join(taskDir, "meta_key")))))
	assert.Equal(t, "session-123", strings.TrimSpace(string(mustReadMuseFile(t, filepath.Join(taskDir, "muse_session")))))
	args := string(mustReadMuseFile(t, filepath.Join(taskDir, "muse_args")))
	assert.Contains(t, args, "--model\nmuse-spark-1.3")
	assert.Contains(t, args, "--base-url\nhttps://api.meta.ai/v1")

	var result map[string]any
	require.NoError(t, json.Unmarshal(mustReadMuseFile(t, resultFile), &result))
	assert.Equal(t, "Done.", result["result"])
	assert.Equal(t, "session-123", result["session_id"])
	usage := result["usage"].(map[string]any)
	assert.Equal(t, float64(12), usage["input_tokens"])
	assert.Equal(t, float64(7), usage["output_tokens"])
}

func mustReadMuseFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
