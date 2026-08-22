package openrouter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunPromptReturnsZeroWhenModelStops(t *testing.T) {
	code := runOpenRouterPrompt(t, false)
	assert.Equal(t, 0, code)
}

func TestRunPromptReturnsNonZeroWhenTurnLimitLeavesPendingTools(t *testing.T) {
	code := runOpenRouterPrompt(t, true)
	assert.Equal(t, 1, code)
}

func runOpenRouterPrompt(t *testing.T, alwaysTools bool) int {
	t.Helper()

	dir := t.TempDir()
	resultFile := filepath.Join(dir, "result.json")
	promptFile := filepath.Join(dir, "prompt.txt")
	harnessFile := filepath.Join(dir, "harness.js")
	require.NoError(t, os.WriteFile(promptFile, []byte("do the work"), 0o644))

	script, err := filepath.Abs("run.js")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(harnessFile, []byte(fmt.Sprintf(`
const { runPrompt } = require(%q);
global.fetch = async () => ({
  ok: true,
  json: async () => ({
    usage: { prompt_tokens: 1, completion_tokens: 1 },
    choices: [{
      message: {
        content: "working",
        tool_calls: %s,
      },
    }],
  }),
});
runPrompt(%q, "openai/gpt-4.1", 2).then((code) => process.exit(code));
`, script, toolCallsJSON(alwaysTools), promptFile)), 0o644))

	cmd := exec.Command("node", harnessFile)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_RESULT_FILE="+resultFile,
		"SUPERPLANE_TASK_DIR="+dir,
		"OPENROUTER_API_KEY=test",
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return 0
	}
	exitErr, ok := err.(*exec.ExitError)
	require.Truef(t, ok, "node harness failed: %v\n%s", err, out)
	return exitErr.ExitCode()
}

func toolCallsJSON(alwaysTools bool) string {
	if !alwaysTools {
		return "undefined"
	}
	return `[{
          id: "call_1",
          type: "function",
          function: { name: "bash", arguments: JSON.stringify({ command: "true" }) },
        }]`
}
