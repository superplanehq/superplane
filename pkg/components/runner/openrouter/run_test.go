package openrouter

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPromptReturnsZeroWhenModelStops(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{maxTurns: 2})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.requests, 2)
	assertToolRouting(t, result.requests[0], true)
	assertToolRouting(t, result.requests[1], true)
	require.GreaterOrEqual(t, len(result.requests[1]["messages"].([]any)), 3)
	assert.Equal(t, "working", resultText(t, result.resultFile))
	assert.Regexp(t, `✓ done · \d+ turns · \$\d+\.\d{4} · \d+\.\d+s`, result.output)
}

func TestRunPromptRequiresToolCapableProviders(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{maxTurns: 1})
	assert.Equal(t, 0, result.exitCode)
	require.NotEmpty(t, result.requests)
	assertToolRouting(t, result.requests[0], true)
}

func TestRunPromptWrapsUpWithoutToolsWhenTurnLimitHits(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{alwaysTools: true, maxTurns: 2})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.requests, 3)
	assertToolRouting(t, result.requests[0], true)
	assertToolRouting(t, result.requests[1], true)
	assertToolRouting(t, result.requests[2], false)
	assert.Contains(t, result.output, "requesting a final response without tools")
	assert.Equal(t, "final summary", resultText(t, result.resultFile))
}

func TestRunPromptRunsLegacyFunctionCall(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{legacyFunctionCall: true, maxTurns: 2})
	assert.Equal(t, 0, result.exitCode)
	require.GreaterOrEqual(t, len(result.requests), 2)
	assert.Contains(t, result.output, `"type":"tool_start"`)
	assert.Contains(t, result.output, `"kind":"bash"`)
	assert.NotContains(t, result.output, "[bash]")
}

func TestRunPromptMarksFilesystemToolErrorsFailed(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{missingRead: true, maxTurns: 2})
	assert.Equal(t, 0, result.exitCode)
	assert.Contains(t, result.output, `"type":"tool_end"`)
	assert.Contains(t, result.output, `"status":"failed"`)
	assert.Contains(t, result.output, `"kind":"read"`)
}

func TestRunPromptKeepsUsageWhenLaterChatFails(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{alwaysTools: true, maxTurns: 4, failOnRequest: 2})
	assert.Equal(t, 1, result.exitCode)
	assert.Contains(t, result.output, "provider exploded")
	assert.Regexp(t, `✗ failed · \d+ turns · \$0\.0020 · \d+\.\d+s`, result.output)

	payload := resultPayload(t, result.resultFile)
	usage, ok := payload["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(11), usage["input_tokens"])
	assert.Equal(t, float64(3), usage["output_tokens"])
	assert.InDelta(t, 0.002, payload["total_cost_usd"], 1e-9)
	assert.Equal(t, "openai/gpt-4.1", payload["model"])

	sidecar := resultPayload(t, filepath.Join(result.taskDir, "llm_usage.json"))
	sidecarUsage, ok := sidecar["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(11), sidecarUsage["input_tokens"])
	assert.Equal(t, float64(3), sidecarUsage["output_tokens"])
	assert.InDelta(t, 0.002, sidecar["total_cost_usd"], 1e-9)
}

type promptHarness struct {
	alwaysTools        bool
	legacyFunctionCall bool
	missingRead        bool
	maxTurns           int
	failOnRequest      int
}

type openRouterPromptResult struct {
	exitCode   int
	output     string
	resultFile string
	taskDir    string
	requests   []map[string]any
}

func runOpenRouterPrompt(t *testing.T, harness promptHarness) openRouterPromptResult {
	t.Helper()

	dir := t.TempDir()
	resultFile := filepath.Join(dir, "result.json")
	promptFile := filepath.Join(dir, "prompt.txt")
	harnessFile := filepath.Join(dir, "harness.js")
	requestsFile := filepath.Join(dir, "requests.json")
	require.NoError(t, os.WriteFile(promptFile, []byte("do the work"), 0o644))

	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	usageScript, err := filepath.Abs("../llm_usage.js")
	require.NoError(t, err)
	usageBody, err := os.ReadFile(usageScript)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "llm_usage.js"), usageBody, 0o644))

	require.NoError(t, os.WriteFile(harnessFile, []byte(fmt.Sprintf(`
const fs = require("fs");
const { runPrompt } = require(%q);
const requests = [];
const alwaysTools = %t;
const legacyFunctionCall = %t;
const missingRead = %t;
const failOnRequest = %d;
let toolTurns = 0;
let requestCount = 0;
global.fetch = async (_url, init) => {
  const body = JSON.parse(init.body);
  requests.push(body);
  requestCount += 1;
  if (failOnRequest > 0 && requestCount === failOnRequest) {
    return {
      ok: false,
      status: 502,
      json: async () => ({ error: { message: "provider exploded" } }),
    };
  }
  const hasTools = Array.isArray(body.tools) && body.tools.length > 0;
  let message = { content: hasTools ? "working" : "final summary" };
  if (alwaysTools && hasTools) {
    message = {
      content: "working",
      tool_calls: [{
        id: "call_1",
        type: "function",
        function: { name: "bash", arguments: JSON.stringify({ command: "true" }) },
      }],
    };
  } else if (legacyFunctionCall && hasTools && toolTurns === 0) {
    toolTurns += 1;
    message = {
      content: "working",
      function_call: { name: "bash", arguments: JSON.stringify({ command: "true" }) },
    };
  } else if (missingRead && hasTools && toolTurns === 0) {
    toolTurns += 1;
    message = {
      content: "working",
      tool_calls: [{
        id: "call_read",
        type: "function",
        function: { name: "read", arguments: JSON.stringify({ path: "/definitely/missing/file.txt" }) },
      }],
    };
  }
  return {
    ok: true,
    json: async () => ({
      usage: { prompt_tokens: 11, completion_tokens: 3, cost: 0.002 },
      choices: [{ message }],
    }),
  };
};
runPrompt(%q, "openai/gpt-4.1", %d)
  .then((code) => {
    fs.writeFileSync(process.env.REQUESTS_FILE, JSON.stringify(requests));
    process.exit(code);
  })
  .catch((err) => {
    fs.writeFileSync(process.env.REQUESTS_FILE, JSON.stringify(requests));
    console.error(err && err.message ? err.message : err);
    process.exit(1);
  });
`, script, harness.alwaysTools, harness.legacyFunctionCall, harness.missingRead, harness.failOnRequest, promptFile, harness.maxTurns)), 0o644))

	cmd := exec.Command("node", harnessFile)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_RESULT_FILE="+resultFile,
		"SUPERPLANE_TASK_DIR="+dir,
		"OPENROUTER_API_KEY=test",
		"REQUESTS_FILE="+requestsFile,
	)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		require.Truef(t, ok, "node harness failed: %v\n%s", err, out)
		exitCode = exitErr.ExitCode()
	}

	raw, readErr := os.ReadFile(requestsFile)
	require.NoError(t, readErr)
	var requests []map[string]any
	require.NoError(t, json.Unmarshal(raw, &requests))

	return openRouterPromptResult{
		exitCode:   exitCode,
		output:     string(out),
		resultFile: resultFile,
		taskDir:    dir,
		requests:   requests,
	}
}

func TestRunPromptPlanningAdvertisesPlanningToolsNotWriteEdit(t *testing.T) {
	result := runOpenRouterPlanningPrompt(t, "none")
	assert.Equal(t, 0, result.exitCode)
	require.NotEmpty(t, result.chatRequests)
	names := toolNames(t, result.chatRequests[0])
	assert.Contains(t, names, "propose_draft")
	assert.Contains(t, names, "survey")
	assert.Contains(t, names, "bash")
	assert.Contains(t, names, "read")
	assert.NotContains(t, names, "write")
	assert.NotContains(t, names, "edit")
	assert.Empty(t, result.planningRequests)
}

func TestRunPromptPlanningProposeDraftPostsToDraftsEndpoint(t *testing.T) {
	result := runOpenRouterPlanningPrompt(t, "propose_draft")
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.planningRequests, 1)
	req := result.planningRequests[0]
	assert.Contains(t, req["url"], "/api/v1/runner/planning-sessions/drafts")
	assert.Equal(t, "POST", req["method"])
	headers, ok := req["headers"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Bearer test-run-token", headers["Authorization"])
	body, ok := req["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Add dark mode", body["title"])
	assert.Equal(t, "Add a dark color scheme", body["description"])
}

func TestRunPromptPlanningSurveyPostsToSurveysEndpoint(t *testing.T) {
	result := runOpenRouterPlanningPrompt(t, "survey")
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.planningRequests, 1)
	req := result.planningRequests[0]
	assert.Contains(t, req["url"], "/api/v1/runner/planning-sessions/surveys")
	body, ok := req["body"].(map[string]any)
	require.True(t, ok)
	questions, ok := body["questions"].([]any)
	require.True(t, ok)
	require.Len(t, questions, 1)
	question := questions[0].(map[string]any)
	assert.Equal(t, "Which theme?", question["prompt"])
}

type openRouterPlanningResult struct {
	exitCode         int
	output           string
	resultFile       string
	chatRequests     []map[string]any
	planningRequests []map[string]any
}

// toolMode selects which tool call (if any) the fake model returns on its
// first tool-enabled turn: "propose_draft", "survey", or "none".
func runOpenRouterPlanningPrompt(t *testing.T, toolMode string) openRouterPlanningResult {
	t.Helper()

	dir := t.TempDir()
	resultFile := filepath.Join(dir, "result.json")
	promptFile := filepath.Join(dir, "prompt.txt")
	harnessFile := filepath.Join(dir, "harness.js")
	requestsFile := filepath.Join(dir, "requests.json")
	require.NoError(t, os.WriteFile(promptFile, []byte("what should we build next?"), 0o644))

	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	planningScript, err := filepath.Abs("../planning_session_mcp.js")
	require.NoError(t, err)
	planningBody, err := os.ReadFile(planningScript)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "planning_session_mcp.js"), planningBody, 0o644))

	require.NoError(t, os.WriteFile(harnessFile, []byte(fmt.Sprintf(`
const fs = require("fs");
const { runPrompt } = require(%q);
const chatRequests = [];
const planningRequests = [];
const toolMode = %q;
let toolTurn = 0;
global.fetch = async (url, init) => {
  const target = String(url);
  if (target.endsWith("/chat/completions")) {
    const body = JSON.parse(init.body);
    chatRequests.push(body);
    const hasTools = Array.isArray(body.tools) && body.tools.length > 0;
    let message = { content: hasTools ? "" : "Thanks, noted." };
    if (hasTools && toolTurn === 0 && toolMode === "propose_draft") {
      toolTurn += 1;
      message = {
        content: "",
        tool_calls: [{
          id: "call_1",
          type: "function",
          function: {
            name: "propose_draft",
            arguments: JSON.stringify({ title: "Add dark mode", description: "Add a dark color scheme" }),
          },
        }],
      };
    } else if (hasTools && toolTurn === 0 && toolMode === "survey") {
      toolTurn += 1;
      message = {
        content: "",
        tool_calls: [{
          id: "call_1",
          type: "function",
          function: {
            name: "survey",
            arguments: JSON.stringify({ questions: [{ prompt: "Which theme?", options: ["Dark", "Light"] }] }),
          },
        }],
      };
    }
    return { ok: true, json: async () => ({ usage: { prompt_tokens: 5, completion_tokens: 2 }, choices: [{ message }] }) };
  }
  const body = init.body ? JSON.parse(init.body) : undefined;
  planningRequests.push({ url: target, method: init.method, headers: init.headers, body });
  if (target.endsWith("/drafts")) {
    return { ok: true, text: async () => JSON.stringify({ status: "created", work_order_key: "WO-1" }) };
  }
  if (target.endsWith("/surveys")) {
    return { ok: true, text: async () => JSON.stringify({ status: "ok" }) };
  }
  return { ok: false, status: 404, text: async () => "not found" };
};
runPrompt(%q, "openai/gpt-4.1", 4)
  .then((code) => {
    fs.writeFileSync(process.env.REQUESTS_FILE, JSON.stringify({ chatRequests, planningRequests }));
    process.exit(code);
  })
  .catch((err) => {
    fs.writeFileSync(process.env.REQUESTS_FILE, JSON.stringify({ chatRequests, planningRequests }));
    console.error(err && err.message ? err.message : err);
    process.exit(1);
  });
`, script, toolMode, promptFile)), 0o644))

	cmd := exec.Command("node", harnessFile)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_RESULT_FILE="+resultFile,
		"SUPERPLANE_TASK_DIR="+dir,
		"SUPERPLANE_PLANNING_SESSION_ID=session-1",
		"SUPERPLANE_BASE_URL=https://superplane.example",
		"SUPERPLANE_RUN_TOKEN=test-run-token",
		"OPENROUTER_API_KEY=test",
		"REQUESTS_FILE="+requestsFile,
	)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		require.Truef(t, ok, "node harness failed: %v\n%s", err, out)
		exitCode = exitErr.ExitCode()
	}

	raw, readErr := os.ReadFile(requestsFile)
	require.NoError(t, readErr)
	var parsed struct {
		ChatRequests     []map[string]any `json:"chatRequests"`
		PlanningRequests []map[string]any `json:"planningRequests"`
	}
	require.NoError(t, json.Unmarshal(raw, &parsed))

	return openRouterPlanningResult{
		exitCode:         exitCode,
		output:           string(out),
		resultFile:       resultFile,
		chatRequests:     parsed.ChatRequests,
		planningRequests: parsed.PlanningRequests,
	}
}

func toolNames(t *testing.T, request map[string]any) []string {
	t.Helper()
	tools, ok := request["tools"].([]any)
	require.True(t, ok)
	names := make([]string, 0, len(tools))
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		require.True(t, ok)
		fn, ok := tool["function"].(map[string]any)
		require.True(t, ok)
		name, _ := fn["name"].(string)
		names = append(names, name)
	}
	return names
}

func assertToolRouting(t *testing.T, request map[string]any, wantTools bool) {
	t.Helper()
	_, hasTools := request["tools"]
	if !wantTools {
		assert.False(t, hasTools)
		_, hasChoice := request["tool_choice"]
		assert.False(t, hasChoice)
		_, hasProvider := request["provider"]
		assert.False(t, hasProvider)
		return
	}
	assert.NotEmpty(t, request["tools"])
	assert.Equal(t, "auto", request["tool_choice"])
	provider, ok := request["provider"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, provider["require_parameters"])
}

func resultText(t *testing.T, resultFile string) string {
	t.Helper()
	text, _ := resultPayload(t, resultFile)["result"].(string)
	return text
}

func resultPayload(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}
