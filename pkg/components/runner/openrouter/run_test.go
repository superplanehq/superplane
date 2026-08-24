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
	result := runOpenRouterPrompt(t, false, 2)
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.requests, 1)
	assert.NotEmpty(t, result.requests[0]["tools"])
	assert.Equal(t, "working", resultText(t, result.resultFile))
}

func TestRunPromptWrapsUpWithoutToolsWhenTurnLimitHits(t *testing.T) {
	result := runOpenRouterPrompt(t, true, 2)
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.requests, 3)
	assert.NotEmpty(t, result.requests[0]["tools"])
	assert.NotEmpty(t, result.requests[1]["tools"])
	_, hasTools := result.requests[2]["tools"]
	assert.False(t, hasTools)
	assert.Contains(t, result.output, "requesting a final response without tools")
	assert.Equal(t, "final summary", resultText(t, result.resultFile))
}

type openRouterPromptResult struct {
	exitCode   int
	output     string
	resultFile string
	requests   []map[string]any
}

func runOpenRouterPrompt(t *testing.T, alwaysTools bool, maxTurns int) openRouterPromptResult {
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
global.fetch = async (_url, init) => {
  const body = JSON.parse(init.body);
  requests.push(body);
  const hasTools = Array.isArray(body.tools) && body.tools.length > 0;
  return {
    ok: true,
    json: async () => ({
      usage: { prompt_tokens: 1, completion_tokens: 1 },
      choices: [{
        message: alwaysTools && hasTools
          ? {
              content: "working",
              tool_calls: [{
                id: "call_1",
                type: "function",
                function: { name: "bash", arguments: JSON.stringify({ command: "true" }) },
              }],
            }
          : { content: hasTools ? "working" : "final summary" },
      }],
    }),
  };
};
runPrompt(%q, "openai/gpt-4.1", %d).then((code) => {
  fs.writeFileSync(process.env.REQUESTS_FILE, JSON.stringify(requests));
  process.exit(code);
});
`, script, alwaysTools, promptFile, maxTurns)), 0o644))

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

func resultText(t *testing.T, resultFile string) string {
	t.Helper()
	raw, err := os.ReadFile(resultFile)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	text, _ := payload["result"].(string)
	return text
}
