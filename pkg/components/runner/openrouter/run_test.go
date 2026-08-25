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
	assert.Contains(t, result.output, "[bash]")
}

func TestRunPromptKeepsUsageWhenLaterChatFails(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{alwaysTools: true, maxTurns: 4, failOnRequest: 2})
	assert.Equal(t, 1, result.exitCode)
	assert.Contains(t, result.output, "provider exploded")

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
`, script, harness.alwaysTools, harness.legacyFunctionCall, harness.failOnRequest, promptFile, harness.maxTurns)), 0o644))

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
