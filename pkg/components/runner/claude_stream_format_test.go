package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeStreamFormatRendersReadableActivity(t *testing.T) {
	t.Parallel()

	python3, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}

	script := filepath.Join(t.TempDir(), "claude_stream_format.py")
	require.NoError(t, os.WriteFile(script, []byte(claudeStreamFormatPython), 0o600))

	input := strings.Join([]string{
		`{"type":"system","subtype":"init","model":"sonnet","cwd":"/tmp/repo"}`,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"text","text":""}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Looking around."}}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Looking around."},{"type":"tool_use","name":"Bash","input":{"command":"ls -la"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"README.md\nsrc\n"}]}}`,
		`{"type":"result","subtype":"success","is_error":false,"num_turns":2,"total_cost_usd":0.0123,"duration_ms":12345,"result":"Done."}`,
		"",
	}, "\n")

	cmd := exec.Command(python3, "-u", script)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "formatter output: %s", out)

	got := string(out)
	assert.Contains(t, got, "Claude Code started · model=sonnet · cwd=/tmp/repo")
	assert.Contains(t, got, "Claude")
	assert.Contains(t, got, "Looking around.")
	assert.Contains(t, got, "→ Bash")
	assert.Contains(t, got, "ls -la")
	assert.Contains(t, got, "← tool result")
	assert.Contains(t, got, "README.md")
	assert.Contains(t, got, "✓ done · 2 turns · $0.0123 · 12.3s")
	assert.NotContains(t, got, `"type":"assistant"`)
}

func TestBuildClaudeCodeBrokerTaskUsesReadableFormatter(t *testing.T) {
	t.Parallel()

	task := buildClaudeCodeBrokerTask(RunClaudeCodeSpec{
		Steps: []ClaudeCodeStep{
			{Name: "Do it", Type: claudeStepPrompt, Prompt: strPtr("do it")},
		},
	})
	require.Len(t, task.Commands, 2)
	assert.Equal(t, "Prepare Claude Code", task.Commands[0].Name)
	assert.Contains(t, task.Commands[0].Command, "python3 not found on PATH")
	assert.Contains(t, task.Commands[0].Command, "format.py")
	promptScript := buildClaudePromptStepScript("do it", "")
	assert.Contains(t, promptScript, "python3 -u")
	assert.Contains(t, promptScript, "tee -a")
}
