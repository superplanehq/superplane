package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowedClaudeToolsRejectsUnknownPlanningKind(t *testing.T) {
	tools := allowedClaudeToolsFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID": "session-1",
		"SUPERPLANE_RUN_TOKEN":           "token",
		"SUPERPLANE_BASE_URL":            "http://localhost:8000",
	})

	assert.Contains(t, tools, "Read")
	assert.Contains(t, tools, "Bash")
	assert.NotContains(t, tools, "mcp__superplane")
	assert.NotContains(t, tools, "mcp__superplane__propose_spec")
	assert.NotContains(t, tools, "mcp__superplane__propose_confidence")
	assert.Contains(t, tools, "Edit")
	assert.Contains(t, tools, "Write")
	assert.NotContains(t, tools, "mcp__superplane__say")
	assert.NotContains(t, tools, "mcp__superplane__wait_for_user")
	assert.NotContains(t, tools, "mcp__superplane__ask")
}

func TestAllowedClaudeToolsAllowsAnalysisPublishTools(t *testing.T) {
	tools := allowedClaudeToolsFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
		"SUPERPLANE_RUN_TOKEN":             "token",
		"SUPERPLANE_BASE_URL":              "http://localhost:8000",
	})

	assert.Contains(t, tools, "Read")
	assert.Contains(t, tools, "Bash")
	assert.Contains(t, tools, "mcp__superplane")
	assert.Contains(t, tools, "mcp__superplane__propose_spec")
	assert.Contains(t, tools, "mcp__superplane__propose_confidence")
	assert.Contains(t, tools, "mcp__superplane__survey")
	assert.NotContains(t, tools, "mcp__superplane__propose_draft")
	assert.NotContains(t, tools, "Edit")
	assert.NotContains(t, tools, "Write")
}

func TestPlanningSystemPromptUsesAnalysisCopy(t *testing.T) {
	analysis := planningSystemPromptFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
	})
	assert.Contains(t, analysis, "propose_spec")
	assert.Contains(t, analysis, "propose_confidence")
	assert.Contains(t, analysis, "how suitable the work is for an agent")
	assert.NotContains(t, analysis, "check copy")
	assert.Contains(t, analysis, "Use only the analysis tools")
	assert.Contains(t, analysis, "call survey with 2 to 4 options")

	unknown := planningSystemPromptFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID": "session-1",
	})
	assert.Empty(t, unknown)
}

func TestAllowedClaudeToolsAllowsFullAccessOutsidePlanning(t *testing.T) {
	tools := allowedClaudeToolsFromScript(t, map[string]string{})
	assert.Equal(t, "Bash,Read,Edit,Write", tools)
}

func TestClaudePermissionModeUsesDefaultModeForAnalysisSession(t *testing.T) {
	// Planning sessions must use "default" (not "plan"): plan mode blocks the
	// planning MCP tools. Read-only is enforced
	// by allowedClaudeTools dropping Edit/Write instead.
	assert.Equal(t, "default", claudePermissionModeFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
	}))
	assert.Equal(t, "acceptEdits", claudePermissionModeFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID": "session-1",
	}))
	assert.Equal(t, "acceptEdits", claudePermissionModeFromScript(t, map[string]string{
		"SUPERPLANE_RUN_TOKEN": "token",
		"SUPERPLANE_BASE_URL":  "http://localhost:8000",
	}))
	assert.Equal(t, "acceptEdits", claudePermissionModeFromScript(t, map[string]string{}))
}

func TestClaudeContinuationUsesExactSession(t *testing.T) {
	assert.Empty(t, claudeContinuationArgsFromScript(t, 0, ""))
	assert.Equal(t, []string{"--resume", "session-123"}, claudeContinuationArgsFromScript(t, 1, "session-123"))
	assert.Equal(t, []string{"--resume", "session-123"}, claudeContinuationArgsFromScript(t, 4, "session-123"))
}

func TestClaudeContinuationRejectsMissingSession(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `require(process.argv[1]).claudeContinuationArgs(1, "")`, script)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "Claude session ID is missing")
}

func TestClaudeSessionIDFromInitEvent(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { claudeSessionIDFromEvent } = require(process.argv[1]); process.stdout.write(claudeSessionIDFromEvent({type:"system", subtype:"init", session_id:"session-123"}));`,
		script,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, "session-123", string(out))
}

func TestClaudeAnalysisRunRecordsTheAgentMessage(t *testing.T) {
	taskDir := t.TempDir()
	prompt := filepath.Join(taskDir, "prompt.txt")
	result := filepath.Join(taskDir, "result.json")
	recorded := filepath.Join(taskDir, "recorded-agent-message")
	require.NoError(t, os.WriteFile(prompt, []byte("Analyze the task."), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "prompt_count"), []byte("0\n"), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(taskDir, "planning_session_mcp.js"),
		[]byte(`const fs = require("fs"); module.exports.recordAgentMessage = async (text) => fs.writeFileSync(process.env.RECORDED_AGENT_MESSAGE, text);`),
		0o600,
	))
	fakeClaude := filepath.Join(taskDir, "claude")
	require.NoError(t, os.WriteFile(
		fakeClaude,
		[]byte("#!/bin/sh\nprintf '%s\\n' '{\"type\":\"result\",\"subtype\":\"success\",\"is_error\":false,\"result\":\"The plan is ready.\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}'\n"),
		0o700,
	))

	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", script, prompt)
	cmd.Env = append(os.Environ(),
		"PATH="+taskDir+":"+os.Getenv("PATH"),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_RESULT_FILE="+result,
		"SUPERPLANE_PLANNING_SESSION_ID=session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND=work_order_analysis",
		"RECORDED_AGENT_MESSAGE="+recorded,
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	message, err := os.ReadFile(recorded)
	require.NoError(t, err)
	assert.Equal(t, "The plan is ready.", string(message))
	_, err = os.Stat(result)
	require.NoError(t, err)
}

func TestClaudeAnalysisRunRecordsAgentMessageFromFailedTurn(t *testing.T) {
	taskDir := t.TempDir()
	prompt := filepath.Join(taskDir, "prompt.txt")
	result := filepath.Join(taskDir, "result.json")
	recorded := filepath.Join(taskDir, "recorded-agent-message")
	require.NoError(t, os.WriteFile(prompt, []byte("Analyze the task."), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "prompt_count"), []byte("0\n"), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(taskDir, "planning_session_mcp.js"),
		[]byte(`const fs = require("fs"); module.exports.recordAgentMessage = async (text) => fs.writeFileSync(process.env.RECORDED_AGENT_MESSAGE, text);`),
		0o600,
	))
	fakeClaude := filepath.Join(taskDir, "claude")
	require.NoError(t, os.WriteFile(
		fakeClaude,
		[]byte("#!/bin/sh\nprintf '%s\\n' '{\"type\":\"result\",\"subtype\":\"error\",\"is_error\":true,\"result\":\"I found useful context before the tool failed.\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}'\n"),
		0o700,
	))

	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", script, prompt)
	cmd.Env = append(os.Environ(),
		"PATH="+taskDir+":"+os.Getenv("PATH"),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_RESULT_FILE="+result,
		"SUPERPLANE_PLANNING_SESSION_ID=session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND=work_order_analysis",
		"RECORDED_AGENT_MESSAGE="+recorded,
	)
	output, err := cmd.CombinedOutput()
	require.Error(t, err, string(output))

	message, err := os.ReadFile(recorded)
	require.NoError(t, err)
	assert.Equal(t, "I found useful context before the tool failed.", string(message))
	_, err = os.Stat(result)
	require.NoError(t, err)
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
	assert.Equal(t, float64(1), records[0]["turn"])
	assert.NotEmpty(t, records[0]["id"])
	assert.Contains(t, output, "On branch main")
	assert.Contains(t, output, `"type":"turn"`)
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
	assert.Equal(t, float64(1), records[0]["turn"])
	assert.Equal(t, float64(1), records[1]["turn"])
}

func TestFormatStreamJsonLinesEmitsTurnUsageAndStampsTools(t *testing.T) {
	output := runClaudeFormatter(t, []string{
		`{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":20},"content":[{"type":"text","text":"I will generate protobufs."},{"type":"tool_use","id":"toolu_a","name":"Bash","input":{"command":"make pb.gen"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_a","content":"ok"}]}}`,
		`{"type":"assistant","message":{"usage":{"input_tokens":80,"output_tokens":10},"content":[{"type":"text","text":"done"}]}}`,
		`{"type":"result","usage":{"input_tokens":180,"output_tokens":30},"num_turns":2}`,
	})

	turns := typedLiveLogRecords(t, output, "turn")
	require.Len(t, turns, 2)
	assert.Equal(t, float64(1), turns[0]["turn"])
	usage, ok := turns[0]["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(100), usage["input_tokens"])
	assert.Equal(t, float64(20), usage["output_tokens"])
	assert.Equal(t, "I will generate protobufs.", turns[0]["message"])
	assert.Equal(t, float64(2), turns[1]["turn"])
	assert.Equal(t, "done", turns[1]["message"])

	tools := liveLogRecords(t, output)
	require.GreaterOrEqual(t, len(tools), 2)
	assert.Equal(t, float64(1), tools[0]["turn"])
	assert.Equal(t, "make pb.gen", tools[0]["text"])
}

func TestFormatStreamJsonLinesReadsOutputFromMessageDelta(t *testing.T) {
	output := runClaudeFormatter(t, []string{
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_a","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":1448,"cache_creation_input_tokens":815}}}}`,
		`{"type":"stream_event","event":{"type":"content_block_start","content_block":{"type":"text","text":""}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Checking the docs."}}}`,
		`{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":47}}}`,
		`{"type":"assistant","message":{"id":"msg_a","usage":{"input_tokens":1,"output_tokens":2,"cache_read_input_tokens":1448,"cache_creation_input_tokens":815},"content":[{"type":"text","text":"Checking the docs."},{"type":"tool_use","id":"toolu_a","name":"Bash","input":{"command":"cat AGENTS.md"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_a","content":"ok"}]}}`,
		`{"type":"stream_event","event":{"type":"message_start","message":{"id":"msg_b","usage":{"input_tokens":2,"output_tokens":1,"cache_read_input_tokens":20409,"cache_creation_input_tokens":2663}}}}`,
		`{"type":"stream_event","event":{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":80}}}`,
		`{"type":"assistant","message":{"id":"msg_b","usage":{"input_tokens":2,"output_tokens":1,"cache_read_input_tokens":20409,"cache_creation_input_tokens":2663},"content":[{"type":"text","text":"Here is the answer."}]}}`,
		`{"type":"result","usage":{"input_tokens":3,"output_tokens":127,"cache_read_input_tokens":21857,"cache_creation_input_tokens":3478},"num_turns":2}`,
	})

	turns := typedLiveLogRecords(t, output, "turn")
	first := lastTurnRecord(t, turns, 1)
	second := lastTurnRecord(t, turns, 2)
	assert.Equal(t, float64(47), first["output_tokens"])
	assert.Equal(t, float64(1), first["input_tokens"])
	assert.Equal(t, float64(1448), first["cache_read_input_tokens"])
	assert.Equal(t, float64(80), second["output_tokens"])
	assert.Equal(t, float64(2), second["input_tokens"])
	assert.Equal(t, float64(20409), second["cache_read_input_tokens"])
	assert.NotContains(t, output, `"turn":3`)
	assert.Equal(t, "Checking the docs.", latestTurnMessage(t, turns, 1))
	assert.Equal(t, "Here is the answer.", latestTurnMessage(t, turns, 2))

	tools := liveLogRecords(t, output)
	require.GreaterOrEqual(t, len(tools), 1)
	assert.Equal(t, float64(1), tools[0]["turn"])
	assert.Equal(t, "cat AGENTS.md", tools[0]["text"])
}

func TestFormatStreamJsonLinesUpdatesUsageForTheSameMessageID(t *testing.T) {
	output := runClaudeFormatter(t, []string{
		`{"type":"assistant","message":{"id":"msg_1","usage":{"input_tokens":2,"output_tokens":1,"cache_read_input_tokens":20409},"content":[{"type":"text","text":"partial"}]}}`,
		`{"type":"assistant","message":{"id":"msg_1","usage":{"input_tokens":2,"output_tokens":2000,"cache_read_input_tokens":20409},"content":[{"type":"text","text":"full answer"}]}}`,
	})

	turns := typedLiveLogRecords(t, output, "turn")
	require.Len(t, turns, 2)
	assert.Equal(t, float64(1), turns[0]["turn"])
	assert.Equal(t, float64(1), turns[1]["turn"])
	usage, ok := turns[1]["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(2000), usage["output_tokens"])
	assert.Equal(t, float64(2), usage["input_tokens"])
	assert.Equal(t, "full answer", turns[1]["message"])
	assert.NotContains(t, output, `"turn":2`)
}

func TestFormatStreamJsonLinesAppliesBilledOutputGapToLastTurn(t *testing.T) {
	output := runClaudeFormatter(t, []string{
		`{"type":"assistant","message":{"usage":{"input_tokens":1,"output_tokens":2,"cache_read_input_tokens":1448},"content":[{"type":"tool_use","id":"toolu_a","name":"Bash","input":{"command":"git status"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_a","content":"ok"}]}}`,
		`{"type":"assistant","message":{"usage":{"input_tokens":2,"output_tokens":1,"cache_read_input_tokens":20409},"content":[{"type":"text","text":"done"}]}}`,
		`{"type":"result","usage":{"input_tokens":13,"output_tokens":2057,"cache_read_input_tokens":71247},"num_turns":2}`,
	})

	turns := typedLiveLogRecords(t, output, "turn")
	require.GreaterOrEqual(t, len(turns), 3)
	last := turns[len(turns)-1]
	assert.Equal(t, float64(2), last["turn"])
	usage, ok := last["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(2055), usage["output_tokens"])
	assert.Equal(t, float64(2), usage["input_tokens"])
}

func TestFormatStreamJsonLinesSkipsDuplicateAssistantUsage(t *testing.T) {
	output := runClaudeFormatter(t, []string{
		`{"type":"assistant","message":{"usage":{"input_tokens":2,"output_tokens":5,"cache_read_input_tokens":1448},"content":[{"type":"text","text":"partial"}]}}`,
		`{"type":"assistant","message":{"usage":{"input_tokens":2,"output_tokens":5,"cache_read_input_tokens":1448},"content":[{"type":"text","text":"partial"}]}}`,
		`{"type":"assistant","message":{"usage":{"input_tokens":2,"output_tokens":21,"cache_read_input_tokens":5394},"content":[{"type":"tool_use","id":"toolu_a","name":"Bash","input":{"command":"git status"}}]}}`,
		`{"type":"result","usage":{"input_tokens":100,"output_tokens":14190,"cache_read_input_tokens":1505585},"total_cost_usd":0.88,"num_turns":2}`,
	})

	turns := typedLiveLogRecords(t, output, "turn")
	require.Len(t, turns, 3)
	assert.Equal(t, float64(1), turns[0]["turn"])
	assert.Equal(t, float64(2), turns[1]["turn"])
	assert.Equal(t, float64(2), turns[2]["turn"])
	usage, ok := turns[1]["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(21), usage["output_tokens"])
	billed, ok := turns[2]["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(14185), billed["output_tokens"])
	assert.NotContains(t, output, `"turn":3`)
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

func claudeContinuationArgsFromScript(t *testing.T, promptCount int, sessionID string) []string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { claudeContinuationArgs } = require(process.argv[1]); process.stdout.write(JSON.stringify(claudeContinuationArgs(Number(process.argv[2]), process.argv[3])));`,
		script,
		fmt.Sprint(promptCount),
		sessionID,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var args []string
	require.NoError(t, json.Unmarshal(out, &args))
	return args
}

func planningSystemPromptFromScript(t *testing.T, env map[string]string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(env)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { planningSystemPrompt } = require(process.argv[1]); process.stdout.write(planningSystemPrompt(JSON.parse(process.argv[2])));`, script, string(payload))
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

func typedLiveLogRecords(t *testing.T, output string, recordType string) []map[string]any {
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
		if rec["type"] == recordType {
			records = append(records, rec)
		}
	}
	return records
}

func lastTurnRecord(t *testing.T, turns []map[string]any, turn float64) map[string]any {
	t.Helper()
	var usage map[string]any
	for _, rec := range turns {
		if rec["turn"] != turn {
			continue
		}
		next, ok := rec["usage"].(map[string]any)
		require.True(t, ok)
		usage = next
	}
	require.NotNil(t, usage, "missing usage for turn %v", turn)
	return usage
}

func latestTurnMessage(t *testing.T, turns []map[string]any, turn float64) string {
	t.Helper()
	message := ""
	found := false
	for _, rec := range turns {
		if rec["turn"] != turn {
			continue
		}
		found = true
		if text, ok := rec["message"].(string); ok && text != "" {
			message = text
		}
	}
	require.True(t, found, "missing turn %v", turn)
	return message
}
