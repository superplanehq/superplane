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

type promptHarness struct {
	alwaysTools        bool
	legacyFunctionCall bool
	maxTurns           int
}

type openRouterPromptResult struct {
	exitCode   int
	output     string
	resultFile string
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

	require.NoError(t, os.WriteFile(harnessFile, []byte(fmt.Sprintf(`
const fs = require("fs");
const { runPrompt } = require(%q);
const requests = [];
const alwaysTools = %t;
const legacyFunctionCall = %t;
let toolTurns = 0;
global.fetch = async (_url, init) => {
  const body = JSON.parse(init.body);
  requests.push(body);
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
      usage: { prompt_tokens: 1, completion_tokens: 1 },
      choices: [{ message }],
    }),
  };
};
runPrompt(%q, "openai/gpt-4.1", %d).then((code) => {
  fs.writeFileSync(process.env.REQUESTS_FILE, JSON.stringify(requests));
  process.exit(code);
});
`, script, harness.alwaysTools, harness.legacyFunctionCall, promptFile, harness.maxTurns)), 0o644))

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
	raw, err := os.ReadFile(resultFile)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	text, _ := payload["result"].(string)
	return text
}
