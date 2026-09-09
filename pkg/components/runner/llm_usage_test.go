package runner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccumulateLLMUsageSumsPromptSteps(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeLLMUsageScript(t, taskDir)

	runAccumulate(t, taskDir, map[string]any{
		"model":          "google/gemini-3.7-flash",
		"usage":          map[string]any{"input_tokens": 10, "output_tokens": 2, "cache_read_input_tokens": 1},
		"total_cost_usd": 0.01,
	})
	runAccumulate(t, taskDir, map[string]any{
		"model":          "google/gemini-3.7-flash",
		"usage":          map[string]any{"input_tokens": 5, "output_tokens": 3, "reasoning_tokens": 4},
		"total_cost_usd": 0.02,
	})

	sidecar := readJSONFile(t, filepath.Join(taskDir, "llm_usage.json"))
	usage := sidecar["usage"].(map[string]any)
	assert.Equal(t, "google/gemini-3.7-flash", sidecar["model"])
	assert.Equal(t, float64(15), usage["input_tokens"])
	assert.Equal(t, float64(5), usage["output_tokens"])
	assert.Equal(t, float64(1), usage["cache_read_input_tokens"])
	assert.Equal(t, float64(4), usage["reasoning_tokens"])
	assert.InDelta(t, 0.03, sidecar["total_cost_usd"], 1e-9)
}

func TestMergeLLMUsageIntoPlanResult(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeLLMUsageScript(t, taskDir)
	resultFile := filepath.Join(taskDir, "result.json")
	require.NoError(t, os.WriteFile(resultFile, []byte(`{"plan":"cGxhbg=="}`+"\n"), 0o644))
	runAccumulate(t, taskDir, map[string]any{
		"model":          "google/gemini-3.7-flash",
		"usage":          map[string]any{"input_tokens": 8, "output_tokens": 1},
		"total_cost_usd": 0.004,
	})

	runMerge(t, taskDir, resultFile)

	merged := readJSONFile(t, resultFile)
	assert.Equal(t, "cGxhbg==", merged["plan"])
	assert.Equal(t, "google/gemini-3.7-flash", merged["model"])
	usage := merged["usage"].(map[string]any)
	assert.Equal(t, float64(8), usage["input_tokens"])
	assert.Equal(t, float64(1), usage["output_tokens"])
	assert.InDelta(t, 0.004, merged["total_cost_usd"], 1e-9)
}

func TestMergeLLMUsageRestoresPromptTelemetry(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeLLMUsageScript(t, taskDir)
	resultFile := filepath.Join(taskDir, "result.json")
	require.NoError(t, os.WriteFile(resultFile, []byte(`{"branch":"feature/x","title":"feat"}`+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "turn_telemetry_series.json"), []byte(`{"series":[`+
		`{"name":"Implementation","telemetry":{"num_turns":2,"usage":{"input_tokens":10},"turns":[{"turn":1,"usage":{"input_tokens":6},"tools":[]},{"turn":2,"usage":{"input_tokens":4},"tools":[]}]}},`+
		`{"name":"Generate PR title and description","telemetry":{"num_turns":1,"usage":{"input_tokens":2},"turns":[{"turn":1,"usage":{"input_tokens":2},"tools":[]}]}}`+
		`]}`+"\n"), 0o644))
	runAccumulate(t, taskDir, map[string]any{
		"model":          "sonnet",
		"usage":          map[string]any{"input_tokens": 12, "output_tokens": 3},
		"total_cost_usd": 0.01,
	})

	runMerge(t, taskDir, resultFile)

	merged := readJSONFile(t, resultFile)
	assert.Equal(t, "feature/x", merged["branch"])
	telemetry, ok := merged["telemetry"].(map[string]any)
	require.True(t, ok)
	prompts, ok := telemetry["prompts"].([]any)
	require.True(t, ok)
	require.Len(t, prompts, 2)
	first := prompts[0].(map[string]any)
	assert.Equal(t, "Implementation", first["name"])
}

func TestMergeLLMUsageLeavesFileUnchangedWithoutSidecar(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeLLMUsageScript(t, taskDir)
	resultFile := filepath.Join(taskDir, "result.json")
	original := `{"plan":"cGxhbg=="}` + "\n"
	require.NoError(t, os.WriteFile(resultFile, []byte(original), 0o644))

	runMerge(t, taskDir, resultFile)

	body, err := os.ReadFile(resultFile)
	require.NoError(t, err)
	assert.Equal(t, original, string(body))
}

func TestWrapAgentStepCommandMergesUsageAfterFailedStep(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeLLMUsageScript(t, taskDir)
	resultFile := filepath.Join(taskDir, "result.json")
	runAccumulate(t, taskDir, map[string]any{
		"model":          "google/gemini-3.7-flash",
		"usage":          map[string]any{"input_tokens": 11, "output_tokens": 7},
		"total_cost_usd": 0.001,
	})

	wrapped := WrapAgentStepCommand(`printf '{"plan":"abc"}\n' > "$SUPERPLANE_RESULT_FILE"; false`)
	cmd := exec.Command("bash", "-c", wrapped)
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_RESULT_FILE="+resultFile,
	)
	err := cmd.Run()
	require.Error(t, err)
	if exitErr, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 1, exitErr.ExitCode())
	}

	merged := readJSONFile(t, resultFile)
	assert.Equal(t, "abc", merged["plan"])
	usage := merged["usage"].(map[string]any)
	assert.Equal(t, float64(11), usage["input_tokens"])
	assert.Equal(t, float64(7), usage["output_tokens"])
}

func writeLLMUsageScript(t *testing.T, taskDir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "llm_usage.js"), []byte(LLMUsageScript), 0o644))
}

func runAccumulate(t *testing.T, taskDir string, payload map[string]any) {
	t.Helper()
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	cmd := exec.Command("node", filepath.Join(taskDir, "llm_usage.js"), "accumulate", string(raw))
	cmd.Env = append(os.Environ(), "SUPERPLANE_TASK_DIR="+taskDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func runMerge(t *testing.T, taskDir, resultFile string) {
	t.Helper()
	cmd := exec.Command("node", filepath.Join(taskDir, "llm_usage.js"), "merge")
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_RESULT_FILE="+resultFile,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(body, &parsed))
	return parsed
}
