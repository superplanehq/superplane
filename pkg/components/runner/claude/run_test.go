package claude

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedClaudeToolsAllowsPlanningSessionTools(t *testing.T) {
	tools := allowedClaudeToolsFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID": "session-1",
		"SUPERPLANE_RUN_TOKEN":           "token",
		"SUPERPLANE_BASE_URL":            "http://localhost:8000",
	})

	assert.Contains(t, tools, "Read")
	assert.Contains(t, tools, "Bash")
	assert.Contains(t, tools, "mcp__superplane")
	assert.Contains(t, tools, "mcp__superplane__propose_draft")
	assert.Contains(t, tools, "mcp__superplane__survey")
	assert.NotContains(t, tools, "Edit")
	assert.NotContains(t, tools, "Write")
	assert.NotContains(t, tools, "mcp__superplane__say")
	assert.NotContains(t, tools, "mcp__superplane__wait_for_user")
	assert.NotContains(t, tools, "mcp__superplane__ask")
}

func TestAllowedClaudeToolsAllowsFullAccessOutsidePlanning(t *testing.T) {
	tools := allowedClaudeToolsFromScript(t, map[string]string{})
	assert.Equal(t, "Bash,Read,Edit,Write", tools)
}

func TestClaudePermissionModeUsesDefaultModeWhenPlanningSessionIsAttached(t *testing.T) {
	// Planning sessions must use "default" (not "plan"): plan mode blocks the
	// planning MCP tools, breaking propose_draft/survey. Read-only is enforced
	// by allowedClaudeTools dropping Edit/Write instead.
	assert.Equal(t, "default", claudePermissionModeFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID": "session-1",
	}))
	assert.Equal(t, "acceptEdits", claudePermissionModeFromScript(t, map[string]string{
		"SUPERPLANE_RUN_TOKEN": "token",
		"SUPERPLANE_BASE_URL":  "http://localhost:8000",
	}))
	assert.Equal(t, "acceptEdits", claudePermissionModeFromScript(t, map[string]string{}))
}

func TestFormatStreamJsonLinesEmitsToolRecords(t *testing.T) {
	output := runClaudeFormatter(t, []string{
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"git status"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","content":"On branch main"}]}}`,
	})

	assert.NotContains(t, output, "-> [Bash]")
	records := liveLogRecords(t, output)
	require.GreaterOrEqual(t, len(records), 2)
	assert.Equal(t, "tool_start", records[0]["type"])
	assert.Equal(t, "bash", records[0]["kind"])
	assert.Equal(t, "git status", records[0]["text"])
	assert.NotEmpty(t, records[0]["id"])
	assert.Contains(t, output, "On branch main")
	end := records[len(records)-1]
	assert.Equal(t, "tool_end", end["type"])
	assert.Equal(t, "passed", end["status"])
}

func TestFormatStreamJsonLinesMatchesToolUseID(t *testing.T) {
	output := runClaudeFormatter(t, []string{
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_a","name":"Read","input":{"file_path":"a.go"}},{"type":"tool_use","id":"toolu_b","name":"Bash","input":{"command":"git status"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_b","is_error":true,"content":"boom"},{"type":"tool_result","tool_use_id":"toolu_a","content":"package a"}]}}`,
	})

	records := liveLogRecords(t, output)
	require.Len(t, records, 4)
	assert.Equal(t, "tool_start", records[0]["type"])
	assert.Equal(t, "bash", records[0]["kind"])
	assert.Equal(t, "tool_end", records[1]["type"])
	assert.Equal(t, "bash", records[1]["kind"])
	assert.Equal(t, "failed", records[1]["status"])
	assert.Equal(t, "read", records[2]["kind"])
	assert.Equal(t, "tool_end", records[3]["type"])
	assert.Equal(t, "read", records[3]["kind"])
	assert.Equal(t, "passed", records[3]["status"])
	assert.Regexp(t, `(?s)"kind":"bash".*boom.*"type":"tool_end".*"kind":"read".*package a`, output)
}

// Claude Code's own "result" event is the authoritative verdict for a turn.
// Headless (`-p`) mode exits 0 even when that event reports is_error: an
// invalid API key still produces a "completed" turn, just one that says it
// failed. run.js must surface that failure instead of trusting the exit code
// alone, otherwise a failed analysis looks like a pass everywhere downstream.
func TestFormatStreamJsonLinesReportsFailedResult(t *testing.T) {
	failed := formatStreamJSONLinesFailed(t, []string{
		`{"type":"system","subtype":"init","model":"claude-opus-4"}`,
		`{"type":"result","is_error":true,"num_turns":1,"total_cost_usd":0,"duration_ms":185600,"result":"Failed to authenticate. API Error: 401 API key is invalid."}`,
	})
	assert.True(t, failed)
}

func TestFormatStreamJsonLinesReportsPassedResult(t *testing.T) {
	failed := formatStreamJSONLinesFailed(t, []string{
		`{"type":"system","subtype":"init","model":"claude-opus-4"}`,
		`{"type":"result","is_error":false,"num_turns":1,"total_cost_usd":0.01,"duration_ms":1200,"result":"Done."}`,
	})
	assert.False(t, failed)
}

func TestFormatStreamJsonLinesWithNoResultEventIsNotFailed(t *testing.T) {
	failed := formatStreamJSONLinesFailed(t, []string{
		`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}]}}`,
	})
	assert.False(t, failed)
}

func claudePermissionModeFromScript(t *testing.T, env map[string]string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(env)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { claudePermissionMode } = require(process.argv[1]); process.stdout.write(claudePermissionMode(JSON.parse(process.argv[2])));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func allowedClaudeToolsFromScript(t *testing.T, env map[string]string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(env)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { allowedClaudeTools } = require(process.argv[1]); process.stdout.write(allowedClaudeTools(JSON.parse(process.argv[2])));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func runClaudeFormatter(t *testing.T, lines []string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(lines)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { formatStreamJsonLines } = require(process.argv[1]); formatStreamJsonLines(JSON.parse(process.argv[2]));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

// formatStreamJSONLinesFailed runs the formatter and reports the "failed"
// verdict it returns, without mixing it into the formatted log text. The
// script writes it to stderr, on its own, so the test does not have to parse
// it out of the log lines formatStreamJsonLines prints to stdout.
func formatStreamJSONLinesFailed(t *testing.T, lines []string) bool {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(lines)
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { formatStreamJsonLines } = require(process.argv[1]);
const result = formatStreamJsonLines(JSON.parse(process.argv[2]));
process.stderr.write(JSON.stringify(result));`,
		script,
		string(payload),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), stdout.String())

	var result struct {
		Failed bool `json:"failed"`
	}
	require.NoError(t, json.Unmarshal(stderr.Bytes(), &result), stderr.String())
	return result.Failed
}

func liveLogRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if rec["type"] == "tool_start" || rec["type"] == "tool_end" {
			records = append(records, rec)
		}
	}
	return records
}
