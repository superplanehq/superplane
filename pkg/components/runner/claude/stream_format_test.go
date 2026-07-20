package claude

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/components/runner"
)

func TestClaudeStreamFormatRendersReadableActivity(t *testing.T) {
	t.Parallel()

	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}

	script := filepath.Join(t.TempDir(), "claude_stream_format.js")
	require.NoError(t, os.WriteFile(script, []byte(streamFormatJS), 0o600))

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

	cmd := exec.Command(node, script)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "formatter output: %s", out)

	got := string(out)
	assert.Contains(t, got, "Claude Code started · model=sonnet · cwd=/tmp/repo")
	assert.Contains(t, got, "Looking around.")
	assert.Contains(t, got, `-> [Bash] ls -la`)
	assert.Contains(t, got, "     README.md")
	assert.Contains(t, got, "     src")
	assert.NotContains(t, got, "← tool result")
	assert.NotRegexp(t, `(?m)^Claude$`, got)
	assert.Contains(t, got, "✓ done · 2 turns · $0.0123 · 12.3s")
	assert.NotContains(t, got, `"type":"assistant"`)
}

func TestClaudeStreamFormatCoalescesPartialTextDeltas(t *testing.T) {
	t.Parallel()

	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}

	script := filepath.Join(t.TempDir(), "claude_stream_format.js")
	require.NoError(t, os.WriteFile(script, []byte(streamFormatJS), 0o600))

	input := strings.Join([]string{
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"text","text":""}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Ther"}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"e's a 'superplane' directory."}}}`,
		`{"type":"stream_event","event":{"type":"content_block_stop"}}`,
		"",
	}, "\n")

	cmd := exec.Command(node, script)
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "formatter output: %s", out)

	got := string(out)
	assert.Contains(t, got, "There's a 'superplane' directory.")
	assert.NotContains(t, got, "Ther\n")
	assert.Equal(t, 1, strings.Count(got, "There's a 'superplane' directory."))
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
	assert.Contains(t, task.Commands[0].Command, "node not found on PATH")
	assert.Equal(t, streamFormatJS, requireTaskFile(t, task.Files, "format.js").Content)
	assert.Equal(t, promptStepScript, requireTaskFile(t, task.Files, "prompt_step.sh").Content)
	assert.Equal(t, "do it", requireTaskFile(t, task.Files, "prompts/01-do-it.txt").Content)
	assert.Contains(t, promptStepScript, `node "$SP/format.js"`)
	assert.Contains(t, promptStepScript, "tee -a")
	assert.Equal(t, runner.BrokerCommand{
		Name:    "Do it",
		Command: `bash "$SUPERPLANE_TASK_DIR/prompt_step.sh" "$SUPERPLANE_TASK_DIR/prompts/01-do-it.txt" ''`,
	}, task.Commands[1])
}
