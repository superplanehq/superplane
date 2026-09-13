package codex

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

func TestCodexExecArgsUsesDangerousBypassOutsidePlanning(t *testing.T) {
	args := codexExecArgsFromScript(t, map[string]string{}, "gpt-5", "/task/planning_session_mcp.js")

	assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
	assert.Contains(t, args, "-m")
	assert.Contains(t, args, "gpt-5")
	assert.NotContains(t, args, "--sandbox")
	assert.NotContains(t, args, "read-only")
	assert.NotContains(t, strings.Join(args, " "), "mcp_servers")
}

func TestCodexExecArgsUsesReadOnlySandboxForAnalysis(t *testing.T) {
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
	}, "gpt-5", "/task/planning_session_mcp.js")

	assert.NotContains(t, args, "--dangerously-bypass-approvals-and-sandbox")
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, `sandbox_mode="read-only"`)
	assert.Contains(t, joined, `approval_policy="never"`)
	assert.Contains(t, joined, `mcp_servers.superplane.command="node"`)
	assert.Contains(t, joined, `mcp_servers.superplane.args=["/task/planning_session_mcp.js"]`)
	assert.Contains(t, joined, "developer_instructions")
}

func TestCodexExecArgsResumesExactSession(t *testing.T) {
	args := codexExecArgsFromScriptWithSession(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
	}, "gpt-5", "/task/planning_session_mcp.js", "019ce0d1-cb1e-7e60-8745-fba83baea3a7")

	assert.Equal(t, []string{"exec", "resume", "019ce0d1-cb1e-7e60-8745-fba83baea3a7"}, args[:3])
	assert.NotContains(t, args, "--last")
}

func TestCodexSessionForPromptRejectsMissingSession(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `require(process.argv[1]).codexSessionForPrompt(1, "")`, script)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "Codex session ID is missing")
}

func TestCodexSessionIDFromThreadEvent(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { codexSessionIDFromEvent } = require(process.argv[1]); process.stdout.write(codexSessionIDFromEvent({type:"thread.started", thread_id:"thread-123"}));`,
		script,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, "thread-123", string(out))
}

func TestCodexAnalysisRunRecordsAgentMessageFromFailedTurn(t *testing.T) {
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
	fakeCodex := filepath.Join(taskDir, "codex")
	require.NoError(t, os.WriteFile(
		fakeCodex,
		[]byte("#!/bin/sh\nprintf '%s\\n' '{\"type\":\"thread.started\",\"thread_id\":\"thread-1\"}' '{\"type\":\"item.completed\",\"item\":{\"id\":\"reply-1\",\"type\":\"agent_message\",\"text\":\"I found useful context before the tool failed.\"}}'\nexit 1\n"),
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

func TestCodexExecArgsUsesDeveloperInstructionsForAnalysis(t *testing.T) {
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
	}, "gpt-5", "/task/planning_session_mcp.js")

	joined := strings.Join(args, " ")
	assert.Contains(t, joined, "developer_instructions=")
	assert.Contains(t, joined, "propose_spec")
	assert.Contains(t, joined, "Use only the analysis tools")
}

func TestPlanningEnabledFromScript(t *testing.T) {
	assert.True(t, planningEnabledFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
	}))
	assert.False(t, planningEnabledFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID": "session-1",
	}))
	assert.False(t, planningEnabledFromScript(t, map[string]string{}))
}

func TestFormatCodexJsonLinesEmitsToolRecords(t *testing.T) {
	output := runCodexFormatter(t, []string{
		`{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"bash -lc git status"}}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"command_execution","command":"bash -lc git status","aggregated_output":"On branch main\n","exit_code":0,"status":"completed"}}`,
		`{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Working tree is clean."}}`,
		`{"type":"item.started","item":{"id":"item_3","type":"file_change","changes":[{"path":"pkg/foo.go","kind":"update"}]}}`,
		`{"type":"item.completed","item":{"id":"item_3","type":"file_change","changes":[{"path":"pkg/foo.go","kind":"update"}]}}`,
	})

	assert.NotContains(t, output, `"type":"item.started"`)
	assert.Contains(t, output, "On branch main")
	assert.Contains(t, output, "Working tree is clean.")
	records := liveLogRecords(t, output)
	require.Len(t, records, 4)
	assert.Equal(t, "tool_start", records[0]["type"])
	assert.Equal(t, "bash", records[0]["kind"])
	assert.Equal(t, "git status", records[0]["text"])
	assert.Equal(t, float64(1), records[0]["turn"])
	assert.Equal(t, "tool_end", records[1]["type"])
	assert.Equal(t, "passed", records[1]["status"])
	assert.Equal(t, "edit", records[2]["kind"])
	assert.Equal(t, "pkg/foo.go", records[2]["text"])
	assert.Equal(t, float64(2), records[2]["turn"])
	assert.Contains(t, output, `"type":"turn"`)
}

func TestFormatCodexJsonLinesPairsAnonymousItemIDs(t *testing.T) {
	output := runCodexFormatter(t, []string{
		`{"type":"item.started","item":{"type":"command_execution","command":"true"}}`,
		`{"type":"item.completed","item":{"type":"command_execution","command":"true","aggregated_output":"ok\n","exit_code":0}}`,
	})

	records := liveLogRecords(t, output)
	require.Len(t, records, 2)
	assert.Equal(t, "tool_start", records[0]["type"])
	assert.Equal(t, "bash", records[0]["kind"])
	assert.Equal(t, "tool_end", records[1]["type"])
	assert.Equal(t, "passed", records[1]["status"])
	assert.Equal(t, 1, strings.Count(output, `"type":"tool_start"`))
}

func TestFormatTurnResultWritesClaudeStyleDoneLine(t *testing.T) {
	output := runCodexTurnResult(t, map[string]any{
		"is_error":       false,
		"num_turns":      1,
		"duration_ms":    1600,
		"total_cost_usd": 0.0022,
	})
	assert.Equal(t, "✓ done · 1 turns · $0.0022 · 1.6s\n", output)
}

func TestFormatTurnResultWritesFailedLine(t *testing.T) {
	output := runCodexTurnResult(t, map[string]any{"is_error": true, "num_turns": 1, "duration_ms": 900})
	assert.Equal(t, "✗ failed · 1 turns · 0.9s\n", output)
}

func TestFormatCodexJsonLinesKeepsOverlappingOutputOnTheRightTool(t *testing.T) {
	output := runCodexFormatter(t, []string{
		`{"type":"item.started","item":{"id":"item_a","type":"command_execution","command":"echo a"}}`,
		`{"type":"item.started","item":{"id":"item_b","type":"command_execution","command":"echo b"}}`,
		`{"type":"item.completed","item":{"id":"item_b","type":"command_execution","command":"echo b","aggregated_output":"bbb\n","exit_code":0}}`,
		`{"type":"item.completed","item":{"id":"item_a","type":"command_execution","command":"echo a","aggregated_output":"aaa\n","exit_code":0}}`,
	})

	records := liveLogRecords(t, output)
	require.Len(t, records, 4)
	assert.Equal(t, "item_b", records[0]["id"])
	assert.Equal(t, "tool_end", records[1]["type"])
	assert.Equal(t, "item_b", records[1]["id"])
	assert.Equal(t, "item_a", records[2]["id"])
	assert.Equal(t, "item_a", records[3]["id"])
	assert.Regexp(t, `(?s)"id":"item_b".*bbb.*"type":"tool_end".*"id":"item_a".*aaa`, output)
}

func codexExecArgsFromScript(t *testing.T, env map[string]string, model, mcpScriptPath string) []string {
	return codexExecArgsFromScriptWithSession(t, env, model, mcpScriptPath, "")
}

func codexExecArgsFromScriptWithSession(
	t *testing.T,
	env map[string]string,
	model, mcpScriptPath, sessionID string,
) []string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	envPayload, err := json.Marshal(env)
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { codexExecArgs } = require(process.argv[1]); process.stdout.write(JSON.stringify(codexExecArgs(JSON.parse(process.argv[2]), process.argv[3], process.argv[4], process.argv[5])));`,
		script,
		string(envPayload),
		model,
		mcpScriptPath,
		sessionID,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var args []string
	require.NoError(t, json.Unmarshal(out, &args))
	return args
}

func planningEnabledFromScript(t *testing.T, env map[string]string) bool {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(env)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { planningEnabled } = require(process.argv[1]); process.stdout.write(JSON.stringify(planningEnabled(JSON.parse(process.argv[2]))));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var enabled bool
	require.NoError(t, json.Unmarshal(out, &enabled))
	return enabled
}

func runCodexTurnResult(t *testing.T, event map[string]any) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { formatTurnResult } = require(process.argv[1]); formatTurnResult(JSON.parse(process.argv[2]));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func runCodexFormatter(t *testing.T, lines []string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(lines)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { formatCodexJsonLines } = require(process.argv[1]); formatCodexJsonLines(JSON.parse(process.argv[2]));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
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
