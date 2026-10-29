package codex

import (
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

func TestThinkingArgsMapsReasoningEffort(t *testing.T) {
	assert.Empty(t, thinkingArgsFromScript(t, ""))
	assert.Empty(t, thinkingArgsFromScript(t, "default"))
	assert.Equal(t, []string{"-c", `model_reasoning_effort="low"`}, thinkingArgsFromScript(t, "low"))
	assert.Equal(t, []string{"-c", `model_reasoning_effort="medium"`}, thinkingArgsFromScript(t, "medium"))
	assert.Equal(t, []string{"-c", `model_reasoning_effort="high"`}, thinkingArgsFromScript(t, "HIGH"))
	assert.Equal(t, []string{"-c", `model_reasoning_effort="minimal"`}, thinkingArgsFromScript(t, "minimal"))
	assert.Equal(t, []string{"-c", `model_reasoning_effort="xhigh"`}, thinkingArgsFromScript(t, "xhigh"))
}

func TestCodexRunReadsModelFromRolloutAndMapsUsage(t *testing.T) {
	taskDir := t.TempDir()
	prompt := filepath.Join(taskDir, "prompt.txt")
	resultPath := filepath.Join(taskDir, "result.json")
	codexHome := filepath.Join(taskDir, "codex-home")
	require.NoError(t, os.WriteFile(prompt, []byte("Inspect the change."), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "prompt_count"), []byte("0\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "codex_subscription.js"), []byte(`module.exports = {
  subscriptionEnabled: () => true,
  subscriptionChildEnv: (env) => env,
  reportUsage: async () => { throw new Error("Usage upload unavailable"); },
};`), 0o600))
	fakeCodex := filepath.Join(taskDir, "codex")
	require.NoError(t, os.WriteFile(fakeCodex, []byte(`#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const id = "thread-123";
const dir = path.join(process.env.CODEX_HOME, "sessions", "2026", "10", "19");
fs.mkdirSync(dir, { recursive: true });
fs.writeFileSync(path.join(dir, "rollout-test-" + id + ".jsonl"), [
  JSON.stringify({ type: "turn_context", payload: { model: "gpt-5.4", effort: "xhigh" } }),
  "{invalid"
].join("\n"));
process.stdout.write(JSON.stringify({ type: "thread.started", thread_id: id }) + "\n");
process.stdout.write(JSON.stringify({ type: "turn.completed", usage: { input_tokens: 10, output_tokens: 4, cache_read_input_tokens: 3, cache_write_input_tokens: 2, reasoning_output_tokens: 1 } }) + "\n");
`), 0o700))

	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", script, prompt)
	cmd.Env = append(withoutCodexAPIKeys(os.Environ()),
		"OPENAI_API_KEY=sk-test-openai-key",
		"PATH="+taskDir+":"+os.Getenv("PATH"),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_RESULT_FILE="+resultPath,
		"CODEX_HOME="+codexHome,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "Usage upload unavailable")

	var result map[string]any
	content, err := os.ReadFile(resultPath)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(content, &result))
	assert.Equal(t, "gpt-5.4", result["model"])
	assert.Equal(t, "xhigh", result["thinkingLevel"])
	usage := result["usage"].(map[string]any)
	assert.Equal(t, float64(2), usage["cache_creation_input_tokens"])
	assert.Equal(t, float64(1), usage["reasoning_tokens"])
}

func thinkingArgsFromScript(t *testing.T, thinking string) []string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { thinkingArgs } = require(process.argv[1]); process.stdout.write(JSON.stringify(thinkingArgs(process.argv[2])));`,
		script,
		thinking,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var args []string
	require.NoError(t, json.Unmarshal(out, &args))
	return args
}

func TestCodexExecArgsUsesDangerousBypassOutsidePlanning(t *testing.T) {
	args := codexExecArgsFromScript(t, map[string]string{}, "gpt-5", "/task/planning_session_mcp.js")

	assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
	assert.Contains(t, args, "-m")
	assert.Contains(t, args, "gpt-5")
	assert.NotContains(t, args, "--sandbox")
	assert.NotContains(t, args, "read-only")
	joined := strings.Join(args, " ")
	assert.NotContains(t, joined, "mcp_servers")
	assert.NotContains(t, joined, "openai_base_url")
}

func TestCodexExecArgsAddsOpenAIBaseURLOverride(t *testing.T) {
	args := codexExecArgsFromScript(t, map[string]string{
		"OPENAI_BASE_URL": "https://proxy.example/v1",
	}, "gpt-5", "/task/planning_session_mcp.js")

	assert.Contains(t, strings.Join(args, " "), `openai_base_url="https://proxy.example/v1"`)
}

func TestCodexExecEnvCopiesOpenAIAPIKey(t *testing.T) {
	env := codexExecEnvFromScript(t, map[string]string{
		"OPENAI_API_KEY": "sk-from-openai",
		"OTHER":          "keep-me",
	})
	assert.Equal(t, "sk-from-openai", env["CODEX_API_KEY"])
	assert.Equal(t, "sk-from-openai", env["OPENAI_API_KEY"])
	assert.Equal(t, "keep-me", env["OTHER"])
}

func TestCodexExecEnvKeepsExplicitCodexAPIKey(t *testing.T) {
	env := codexExecEnvFromScript(t, map[string]string{
		"CODEX_API_KEY":  "sk-codex",
		"OPENAI_API_KEY": "sk-openai",
	})
	assert.Equal(t, "sk-codex", env["CODEX_API_KEY"])
	assert.Equal(t, "sk-openai", env["OPENAI_API_KEY"])
}

func TestCodexExecEnvRejectsMissingAPIKey(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `require(process.argv[1]).codexExecEnv({OPENAI_API_KEY:"  "})`, script)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	assert.Contains(t, string(out), "Codex API key is missing")
}

func TestCodexExecArgsAddsMergeConfidenceMCP(t *testing.T) {
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_BASE_URL":               "http://superplane.test",
		"SUPERPLANE_MERGE_CONFIDENCE_TOKEN": "merge-confidence-secret",
	}, "gpt-5", "/task/merge_confidence_mcp.js")

	joined := strings.Join(args, " ")
	assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
	assert.Contains(t, joined, `mcp_servers.superplane.command="node"`)
	assert.Contains(t, joined, `mcp_servers.superplane.args=["/task/merge_confidence_mcp.js"]`)
	assertForwardsMCPEnvVars(t, args, "SUPERPLANE_BASE_URL", "SUPERPLANE_MERGE_CONFIDENCE_TOKEN")
	assert.NotContains(t, joined, "merge-confidence-secret")
	assert.NotContains(t, joined, "approval_mode")
}

func assertForwardsMCPEnvVars(t *testing.T, args []string, names ...string) {
	t.Helper()
	var envVars string
	for _, arg := range args {
		if strings.HasPrefix(arg, "mcp_servers.superplane.env_vars=") {
			envVars = arg
		}
	}
	require.NotEmpty(t, envVars, "missing mcp_servers.superplane.env_vars override")
	for _, name := range names {
		assert.Contains(t, envVars, fmt.Sprintf("%q", name))
	}
}

func TestCodexSuperplaneMCPScriptUsesMergeConfidence(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { superplaneMCPScript } = require(process.argv[1]); process.stdout.write(superplaneMCPScript("/task", { SUPERPLANE_MERGE_CONFIDENCE_TOKEN: "token" }));`, script)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, "/task/merge_confidence_mcp.js", string(out))
}

func TestCodexExecArgsAddsArtifactMCPOutsidePlanning(t *testing.T) {
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_ARTIFACT_TOKEN": "artifact-token",
	}, "gpt-5", "/task/task_artifact_mcp.js")

	joined := strings.Join(args, " ")
	assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
	assert.Contains(t, joined, `mcp_servers.superplane.command="node"`)
	assert.Contains(t, joined, `mcp_servers.superplane.args=["/task/task_artifact_mcp.js"]`)
	assertForwardsMCPEnvVars(t, args, "SUPERPLANE_BASE_URL", "SUPERPLANE_TASK_DIR", "SUPERPLANE_ARTIFACT_TOKEN")
	assert.NotContains(t, joined, "artifact-token")
	assert.NotContains(t, joined, "approval_mode")
}

func TestCodexExecArgsAddsGlobalMCPWithApprovedTools(t *testing.T) {
	for name, env := range map[string]map[string]string{
		"task run": {"SUPERPLANE_RUNNER_TOKEN": "runner-token-secret"},
		"planning": {
			"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
			"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
			"SUPERPLANE_RUNNER_TOKEN":          "runner-token-secret",
		},
	} {
		t.Run(name, func(t *testing.T) {
			args := codexExecArgsFromScript(t, env, "gpt-5", "/task/planning_session_mcp.js")
			joined := strings.Join(args, " ")

			assert.Contains(t, args, `mcp_servers.superplane_global.command="node"`)
			assert.Contains(t, args, `mcp_servers.superplane_global.args=["/task/superplane_mcp/server.js"]`)
			assert.Contains(t, args, `mcp_servers.superplane_global.env_vars=["SUPERPLANE_BASE_URL", "SUPERPLANE_RUNNER_TOKEN", "SUPERPLANE_TASK_DIR"]`)
			assert.Contains(t, args, `mcp_servers.superplane_global.tools.suggest_owners.approval_mode="approve"`)
			assert.Contains(t, args, `mcp_servers.superplane_global.tools.assign_owner.approval_mode="approve"`)
			assert.NotContains(t, joined, "mcp_servers.superplane.tools.suggest_owners")
			assert.NotContains(t, joined, "runner-token-secret")
		})
	}
}

func TestCodexExecArgsUsesReadOnlySandboxForAnalysis(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "workspace_mcp.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"servers":[{"name":"docs","url":"https://mcp.example.com/mcp"}]}`), 0o644))
	for _, session := range []string{"", "thread-1"} {
		t.Run("session="+session, func(t *testing.T) {
			args := codexExecArgsFromScriptWithSession(t, map[string]string{
				"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
				"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
				"SUPERPLANE_WORKSPACE_MCP_CONFIG":  configPath,
				"SUPERPLANE_RUN_TOKEN":             "run-token-secret",
			}, "gpt-5", "/task/planning_session_mcp.js", session)

			assert.NotContains(t, args, "--dangerously-bypass-approvals-and-sandbox")
			joined := strings.Join(args, " ")
			assert.Contains(t, joined, `sandbox_mode="read-only"`)
			assert.Contains(t, joined, `approval_policy="never"`)
			assert.Contains(t, joined, `mcp_servers.superplane.command="node"`)
			assert.Contains(t, joined, `mcp_servers.superplane.args=["/task/planning_session_mcp.js"]`)
			assert.Contains(t, joined, "developer_instructions")
			assert.Contains(t, args, `mcp_servers.superplane.tools.propose_update.approval_mode="approve"`)
			assert.Contains(t, args, `mcp_servers.superplane.tools.inspect_attachment.approval_mode="approve"`)
			assert.Contains(t, args, `mcp_servers.docs.url="https://mcp.example.com/mcp"`)
			assert.NotContains(t, joined, "superplane_global")
			assertForwardsMCPEnvVars(t, args,
				"SUPERPLANE_BASE_URL",
				"SUPERPLANE_RUN_TOKEN",
				"SUPERPLANE_ARTIFACT_TOKEN",
				"SUPERPLANE_MERGE_CONFIDENCE_TOKEN",
				"SUPERPLANE_ACTIVITY_ID",
				"SUPERPLANE_TASK_DIR",
				"SUPERPLANE_ANALYSIS_SPEC_FILE",
				"SUPERPLANE_ANALYSIS_SCORE_FILE",
			)
			assert.NotContains(t, joined, "run-token-secret")
			assert.Equal(t, 2, strings.Count(joined, "approval_mode="))
			assert.NotContains(t, joined, "default_tools_approval_mode")
		})
	}
}

func TestCodexExecArgsMergesWorkspaceMCP(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "workspace_mcp.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"servers":[{"name":"docs","url":"https://mcp.example.com/mcp","headers":{"Authorization":"Bearer tok"}}]}`), 0o644))
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_WORKSPACE_MCP_CONFIG": configPath,
	}, "gpt-5", "/task/planning_session_mcp.js")
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, `mcp_servers.docs.url="https://mcp.example.com/mcp"`)
	assert.Contains(t, joined, `mcp_servers.docs.http_headers.Authorization="Bearer tok"`)
}

func TestCodexExecArgsDisablesWorkspaceMCPTools(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "workspace_mcp.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"servers":[{"name":"docs","url":"https://mcp.example.com/mcp","disabledTools":["create_issue","search"]}]}`), 0o644))
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_WORKSPACE_MCP_CONFIG": configPath,
	}, "gpt-5", "/task/planning_session_mcp.js")
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, `mcp_servers.docs.disabled_tools=["create_issue", "search"]`)
}

func TestCodexExecArgsReadsWorkspaceMCPFromTaskDir(t *testing.T) {
	taskDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "workspace_mcp.json"), []byte(`{"servers":[{"name":"deepwiki","url":"https://mcp.deepwiki.com/mcp"}]}`), 0o644))
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_TASK_DIR":             taskDir,
		"SUPERPLANE_WORKSPACE_MCP_CONFIG": "/task/workspace_mcp.json",
	}, "gpt-5", "/task/planning_session_mcp.js")
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, `mcp_servers.deepwiki.url="https://mcp.deepwiki.com/mcp"`)
}

func TestCodexExecArgsQuotesUnsafeWorkspaceMCPKeys(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "workspace_mcp.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"servers":[{"name":"linear-docs","url":"https://mcp.example.com/mcp","headers":{"X-API-Key":"secret","X.Custom":"dotted"}}]}`), 0o644))
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_WORKSPACE_MCP_CONFIG": configPath,
	}, "gpt-5", "/task/planning_session_mcp.js")
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, `mcp_servers."linear-docs".url="https://mcp.example.com/mcp"`)
	assert.Contains(t, joined, `mcp_servers."linear-docs".http_headers."X-API-Key"="secret"`)
	assert.Contains(t, joined, `mcp_servers."linear-docs".http_headers."X.Custom"="dotted"`)
}

func TestCodexExecArgsResumesExactSession(t *testing.T) {
	args := codexExecArgsFromScriptWithSession(t, map[string]string{}, "gpt-5", "/task/planning_session_mcp.js", "019ce0d1-cb1e-7e60-8745-fba83baea3a7")

	assert.Equal(t, []string{"exec", "resume", "019ce0d1-cb1e-7e60-8745-fba83baea3a7"}, args[:3])
	assert.NotContains(t, args, "--last")
}

func TestCodexPlanningFollowUpDoesNotResume(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { codexSessionForPrompt } = require(process.argv[1]); process.stdout.write(JSON.stringify(codexSessionForPrompt(4, "019ce0d1-cb1e-7e60-8745-fba83baea3a7", {SUPERPLANE_PLANNING_SESSION_KIND:"work_order_analysis",SUPERPLANE_ANALYSIS_REWIND:"yes"})));`,
		script,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, `""`, strings.TrimSpace(string(out)))
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
		"OPENAI_API_KEY=sk-test-openai-key",
	)
	output, err := cmd.CombinedOutput()
	require.Error(t, err, string(output))

	message, err := os.ReadFile(recorded)
	require.NoError(t, err)
	assert.Equal(t, "I found useful context before the tool failed.", string(message))
	_, err = os.Stat(result)
	require.NoError(t, err)
}

func TestCodexRunSelectsAuthenticationForCodexChild(t *testing.T) {
	for _, test := range []struct {
		name         string
		subscription bool
		suppliedKeys bool
	}{
		{name: "API key", suppliedKeys: true},
		{name: "subscription without API keys", subscription: true},
		{name: "subscription removes API keys", subscription: true, suppliedKeys: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			taskDir := t.TempDir()
			prompt := filepath.Join(taskDir, "prompt.txt")
			result := filepath.Join(taskDir, "result.json")
			received := filepath.Join(taskDir, "received-environment.json")
			require.NoError(t, os.WriteFile(prompt, []byte("hello"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(taskDir, "prompt_count"), []byte("0\n"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(taskDir, "codex"), []byte(`#!/usr/bin/env node
const fs = require("fs");
fs.writeFileSync(process.env.RECEIVED_ENVIRONMENT, JSON.stringify(process.env));
process.stdout.write('{"type":"thread.started","thread_id":"thread-1"}\n{"type":"item.completed","item":{"id":"reply-1","type":"agent_message","text":"ok"}}\n');
`), 0o700))
			script, err := filepath.Abs("run.js")
			require.NoError(t, err)
			cmd := exec.Command("node", script, prompt)
			cmd.Env = append(withoutCodexAPIKeys(os.Environ()),
				"PATH="+taskDir+":"+os.Getenv("PATH"),
				"SUPERPLANE_TASK_DIR="+taskDir,
				"SUPERPLANE_RESULT_FILE="+result,
				"RECEIVED_ENVIRONMENT="+received,
			)
			if test.suppliedKeys {
				cmd.Env = append(cmd.Env, "OPENAI_API_KEY=sk-from-openai", "OPENAI_BASE_URL=https://api.example.com")
				if test.subscription {
					cmd.Env = append(cmd.Env, "CODEX_API_KEY=sk-codex")
				}
			}
			if test.subscription {
				require.NoError(t, os.WriteFile(filepath.Join(taskDir, "codex_subscription.js"), []byte(subscriptionScript), 0o600))
				cmd.Env = append(cmd.Env, "SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN=subscription-token")
			}
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, string(output))
			encoded, err := os.ReadFile(received)
			require.NoError(t, err)
			var got map[string]string
			require.NoError(t, json.Unmarshal(encoded, &got))
			if test.subscription {
				assert.Empty(t, got["CODEX_API_KEY"])
				assert.Empty(t, got["OPENAI_API_KEY"])
				assert.Empty(t, got["OPENAI_BASE_URL"])
				assert.Equal(t, filepath.Join(taskDir, "codex-home"), got["CODEX_HOME"])
				return
			}
			assert.Equal(t, "sk-from-openai", got["CODEX_API_KEY"])
			assert.Equal(t, "sk-from-openai", got["OPENAI_API_KEY"])
		})
	}
}

func TestCodexRunRejectsMissingAPIKeyBeforeSpawn(t *testing.T) {
	taskDir := t.TempDir()
	prompt := filepath.Join(taskDir, "prompt.txt")
	result := filepath.Join(taskDir, "result.json")
	spawned := filepath.Join(taskDir, "spawned")
	require.NoError(t, os.WriteFile(prompt, []byte("hello"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "prompt_count"), []byte("0\n"), 0o600))
	require.NoError(t, os.WriteFile(
		filepath.Join(taskDir, "codex"),
		[]byte("#!/bin/sh\ntouch \"$SPAWNED\"\n"),
		0o700,
	))

	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", script, prompt)
	cmd.Env = append(withoutCodexAPIKeys(os.Environ()),
		"PATH="+taskDir+":"+os.Getenv("PATH"),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_RESULT_FILE="+result,
		"SPAWNED="+spawned,
	)
	output, err := cmd.CombinedOutput()
	require.Error(t, err, string(output))
	assert.Contains(t, string(output), "Codex API key is missing")
	_, statErr := os.Stat(spawned)
	assert.Error(t, statErr)
}

func TestCodexExecArgsUsesDeveloperInstructionsForAnalysis(t *testing.T) {
	args := codexExecArgsFromScript(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
	}, "gpt-5", "/task/planning_session_mcp.js")

	joined := strings.Join(args, " ")
	assert.Contains(t, joined, "developer_instructions=")
	assert.Contains(t, joined, "propose_update")
	assert.Contains(t, joined, "Do not call propose_spec, propose_clarity, propose_confidence, or survey")
	assert.NotContains(t, joined, "Call propose_spec")
	assert.NotContains(t, joined, "Call propose_clarity")
	assert.NotContains(t, joined, "Call propose_confidence")
	assert.NotContains(t, joined, "propose_plan")
	assert.Contains(t, joined, "Use only the analysis tools")
}

func TestCodexExecArgsKeepsProtocolAtDeveloperPriority(t *testing.T) {
	protocol, err := os.ReadFile(filepath.Join("..", "analysis_protocol_review.md"))
	require.NoError(t, err)
	args := codexExecArgsFromScriptWithPrompt(t, map[string]string{
		"SUPERPLANE_PLANNING_SESSION_ID":   "session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND": "work_order_analysis",
	}, "gpt-5", "/task/planning_session_mcp.js", "", string(protocol)+"\n\nTask:\nFix retries.")

	assert.Contains(t, strings.Join(args, " "), "developer_instructions")
	assert.Contains(t, strings.Join(args, " "), strings.Split(string(protocol), "\n")[0])
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
		`{"type":"item.started","item":{"id":"item_3","type":"file_change","changes":[{"path":"pkg/foo.go","kind":"update"},{"path":"pkg/bar.go","kind":"update"}]}}`,
		`{"type":"item.completed","item":{"id":"item_3","type":"file_change","changes":[{"path":"pkg/foo.go","kind":"update"},{"path":"pkg/bar.go","kind":"update"}]}}`,
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
	assert.Equal(t, "pkg/foo.go\npkg/bar.go", records[2]["text"])
	assert.Equal(t, float64(2), records[2]["turn"])
	assert.Contains(t, output, `"type":"turn"`)
}

func TestFormatCodexJsonLinesRedactsSecretsAndSummarizesImages(t *testing.T) {
	token := "github-token-for-codex-redaction"
	t.Setenv("GITHUB_TOKEN", token)
	image := strings.Repeat("b", 2048)
	imageResult := fmt.Sprintf(`{"content":[{"type":"image","data":%q,"mimeType":"image/png"}]}`, image)
	output := runCodexFormatter(t, []string{
		fmt.Sprintf(`{"type":"item.completed","item":{"id":"reasoning-1","type":"reasoning","text":%q}}`, token),
		fmt.Sprintf(`{"type":"item.started","item":{"id":"item-1","type":"command_execution","command":%q}}`, "git clone https://x-access-token:"+token+"@github.com/acme/app.git"),
		fmt.Sprintf(`{"type":"item.completed","item":{"id":"item-1","type":"command_execution","command":"true","aggregated_output":%q,"exit_code":0}}`, imageResult),
		fmt.Sprintf(`{"type":"item.completed","item":{"id":"publish-1","type":"mcp_tool_call","server":"superplane","tool":"propose_update","status":"failed","error":{"message":%q}}}`, token),
		fmt.Sprintf(`{"type":"item.completed","item":{"id":"answer-1","type":"agent_message","text":%q}}`, token),
	})

	assert.NotContains(t, output, token)
	assert.NotContains(t, output, image)
	assert.NotContains(t, output, "x-access-token:")
	assert.Contains(t, output, "[REDACTED]")
	assert.Contains(t, output, "image content omitted from logs")
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

func TestCodexFormatterReportsPlanningToolFailures(t *testing.T) {
	for _, status := range []string{"failed", "cancelled", "declined", "completed"} {
		for _, withActivity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/activity=%t", status, withActivity), func(t *testing.T) {
				item := map[string]any{
					"id": "publish-1", "type": "mcp_tool_call", "server": "superplane",
					"tool": "propose_update", "status": status,
				}
				failed := status != "completed"
				if failed {
					item["error"] = map[string]any{"message": "MCP tool approval was cancelled"}
				}
				event, err := json.Marshal(map[string]any{"type": "item.completed", "item": item})
				require.NoError(t, err)
				format := runCodexFormatter
				if withActivity {
					format = runCodexFormatterWithActivity
				}
				output := format(t, []string{string(event)})
				records := liveLogRecords(t, output)
				require.Len(t, records, 2)
				assert.Contains(t, records[0]["text"], "propose_update")
				expectedStatus := "passed"
				if failed {
					expectedStatus = "failed"
					assert.Contains(t, output, "MCP tool approval was cancelled")
				}
				assert.Equal(t, expectedStatus, records[1]["status"])
				if !withActivity {
					return
				}
				var ended bool
				var errorOutput bool
				for _, record := range activityRecords(t, output) {
					if record["type"] == "activity_tool_end" {
						ended = true
						activityStatus := expectedStatus
						if status == "cancelled" {
							activityStatus = "cancelled"
						}
						assert.Equal(t, activityStatus, record["status"])
					}
					if record["channel"] == "tool_output" && record["output_stream"] == "stderr" {
						errorOutput = true
					}
				}
				assert.True(t, ended)
				assert.Equal(t, failed, errorOutput)
			})
		}
	}
}

func TestCodexFormatterShowsFailedMCPToolResultText(t *testing.T) {
	event := `{"type":"item.completed","item":{"id":"publish-1","type":"mcp_tool_call","server":"superplane","tool":"propose_update","status":"failed","error":null,"result":{"content":[{"type":"text","text":"SUPERPLANE_BASE_URL is required"}],"structured_content":null}}}`
	output := runCodexFormatterWithActivity(t, []string{event})

	var errorOutput string
	for _, record := range activityRecords(t, output) {
		if record["channel"] == "tool_output" && record["output_stream"] == "stderr" {
			errorOutput, _ = record["text"].(string)
		}
	}
	assert.Equal(t, "SUPERPLANE_BASE_URL is required", errorOutput)
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
	assert.Equal(t, "item_a", records[0]["id"])
	assert.Equal(t, "tool_start", records[0]["type"])
	assert.Equal(t, "item_b", records[1]["id"])
	assert.Equal(t, "tool_start", records[1]["type"])
	assert.Equal(t, "item_b", records[2]["id"])
	assert.Equal(t, "tool_end", records[2]["type"])
	assert.Equal(t, "item_a", records[3]["id"])
	assert.Equal(t, "tool_end", records[3]["type"])
	assert.Regexp(t, `(?s)"id":"item_a".*"id":"item_b".*bbb.*"type":"tool_end".*aaa`, output)
}

func TestFormatCodexJsonLinesEmitsReasoningAndOrderedActivity(t *testing.T) {
	output := runCodexFormatterWithActivity(t, []string{
		`{"type":"item.completed","item":{"id":"reasoning-1","type":"reasoning","text":"Inspect the retry path."}}`,
		`{"type":"item.started","item":{"id":"tool-a","type":"command_execution","command":"echo a"}}`,
		`{"type":"item.started","item":{"id":"tool-b","type":"command_execution","command":"echo b"}}`,
		`{"type":"item.completed","item":{"id":"tool-b","type":"command_execution","command":"echo b","aggregated_output":"b","exit_code":0}}`,
		`{"type":"item.completed","item":{"id":"tool-a","type":"command_execution","command":"echo a","aggregated_output":"a","exit_code":0}}`,
	})

	records := activityRecords(t, output)
	require.NotEmpty(t, records)
	assert.Equal(t, "activity_start", records[0]["type"])
	assert.Equal(t, "content_start", records[1]["type"])
	assert.Equal(t, "reasoning", records[1]["channel"])
	starts := []string{}
	for _, record := range records {
		if record["type"] == "activity_tool_start" {
			starts = append(starts, record["id"].(string))
		}
	}
	assert.Equal(t, []string{"tool-a", "tool-b"}, starts)
}

func codexExecArgsFromScript(t *testing.T, env map[string]string, model, mcpScriptPath string) []string {
	return codexExecArgsFromScriptWithSession(t, env, model, mcpScriptPath, "")
}

func codexExecArgsFromScriptWithSession(
	t *testing.T,
	env map[string]string,
	model, mcpScriptPath, sessionID string,
) []string {
	return codexExecArgsFromScriptWithPrompt(t, env, model, mcpScriptPath, sessionID, "")
}

func codexExecArgsFromScriptWithPrompt(
	t *testing.T,
	env map[string]string,
	model, mcpScriptPath, sessionID, prompt string,
) []string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	envPayload, err := json.Marshal(env)
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { codexExecArgs } = require(process.argv[1]); process.stdout.write(JSON.stringify(codexExecArgs(JSON.parse(process.argv[2]), process.argv[3], process.argv[4], process.argv[5], process.argv[6])));`,
		script,
		string(envPayload),
		model,
		mcpScriptPath,
		sessionID,
		prompt,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var args []string
	require.NoError(t, json.Unmarshal(out, &args))
	return args
}

func codexExecEnvFromScript(t *testing.T, env map[string]string) map[string]string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(env)
	require.NoError(t, err)
	cmd := exec.Command(
		"node",
		"-e",
		`const { codexExecEnv } = require(process.argv[1]); process.stdout.write(JSON.stringify(codexExecEnv(JSON.parse(process.argv[2]))));`,
		script,
		string(payload),
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var childEnv map[string]string
	require.NoError(t, json.Unmarshal(out, &childEnv))
	return childEnv
}

func withoutCodexAPIKeys(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "OPENAI_API_KEY=") || strings.HasPrefix(entry, "CODEX_API_KEY=") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
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

func runCodexFormatterWithActivity(t *testing.T, lines []string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(lines)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { formatCodexJsonLines } = require(process.argv[1]); formatCodexJsonLines(JSON.parse(process.argv[2]));`, script, string(payload))
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_PLANNING_SESSION_ID=session-1",
		"SUPERPLANE_PLANNING_SESSION_KIND=work_order_analysis",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func activityRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(output, "\n") {
		var record map[string]any
		if json.Unmarshal([]byte(line), &record) == nil && record["schema_version"] == float64(2) {
			records = append(records, record)
		}
	}
	return records
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

func TestFormatCodexJsonLinesEmitsCompletedUsage(t *testing.T) {
	output := runCodexFormatter(t, []string{
		`{"type":"item.completed","item":{"id":"answer","type":"agent_message","text":"done"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":100,"cached_input_tokens":30,"output_tokens":20},"total_cost_usd":0.12}`,
	})
	var last map[string]any
	for _, line := range strings.Split(output, "\n") {
		var record map[string]any
		if json.Unmarshal([]byte(line), &record) == nil && record["type"] == "turn" {
			last = record
		}
	}
	require.NotNil(t, last)
	usage := last["usage"].(map[string]any)
	assert.Equal(t, float64(100), usage["input_tokens"])
	assert.Equal(t, float64(30), usage["cache_read_input_tokens"])
	assert.Equal(t, float64(20), usage["output_tokens"])
	require.Contains(t, usage, "total_cost_usd")
	assert.InDelta(t, 0.12, usage["total_cost_usd"], 1e-9)
}
