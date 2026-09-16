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

func TestTurnTelemetryJoinsToolsAndCumulativeUsage(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const records = [];
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: (record) => records.push(record),
});
telemetry.beginTurn({ input_tokens: 10, output_tokens: 2 });
telemetry.stampToolStart({ type: "tool_start", id: "a", kind: "bash", text: "make pb.gen" });
telemetry.stampToolEnd({ type: "tool_end", id: "a", kind: "bash", status: "passed", duration_ms: 40 });
telemetry.beginTurn({ input_tokens: 8, output_tokens: 3 });
telemetry.stampToolStart({ type: "tool_start", id: "b", kind: "read", text: "pkg/protos/canvases.pb.go" });
process.stdout.write(JSON.stringify({ records, telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Records, 2)
	assert.Equal(t, "turn", payload.Records[0]["type"])
	assert.Equal(t, float64(1), payload.Records[0]["turn"])
	assert.Equal(t, "turn", payload.Records[1]["type"])
	assert.Equal(t, float64(2), payload.Records[1]["turn"])

	require.Len(t, payload.Telemetry.Turns, 2)
	assert.Equal(t, 2, payload.Telemetry.NumTurns)
	assert.Equal(t, float64(18), payload.Telemetry.Usage["input_tokens"])
	assert.Equal(t, float64(5), payload.Telemetry.Usage["output_tokens"])
	assert.Equal(t, 1, payload.Telemetry.ToolCounts["bash"])
	assert.Equal(t, 1, payload.Telemetry.ToolCounts["read"])

	first := payload.Telemetry.Turns[0]
	require.Len(t, first.Tools, 1)
	assert.Equal(t, "bash", first.Tools[0]["kind"])
	assert.Equal(t, "make pb.gen", first.Tools[0]["text"])
	assert.Equal(t, "passed", first.Tools[0]["status"])
	assert.Equal(t, float64(40), first.Tools[0]["duration_ms"])

	second := payload.Telemetry.Turns[1]
	assert.Equal(t, float64(8), second.Usage["input_tokens"])
	require.Len(t, second.Tools, 1)
	assert.Equal(t, "read", second.Tools[0]["kind"])
	assert.Equal(t, "pkg/protos/canvases.pb.go", second.Tools[0]["text"])
}

func TestTurnTelemetryTreatsLargerUsageAsCumulative(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: () => {},
});
telemetry.beginTurn({ input_tokens: 10, output_tokens: 2 });
telemetry.beginTurn({ input_tokens: 40, output_tokens: 9 });
process.stdout.write(JSON.stringify({ records: [], telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Telemetry.Turns, 2)
	assert.Equal(t, float64(10), payload.Telemetry.Turns[0].Usage["input_tokens"])
	assert.Equal(t, float64(40), payload.Telemetry.Turns[1].Usage["input_tokens"])
	assert.Equal(t, float64(9), payload.Telemetry.Turns[1].Usage["output_tokens"])
	assert.Equal(t, float64(50), payload.Telemetry.Usage["input_tokens"])
	assert.Equal(t, float64(11), payload.Telemetry.Usage["output_tokens"])
}

func TestTurnTelemetryPersistsAcrossProcesses(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: () => {},
});
telemetry.beginTurn({ input_tokens: 5, output_tokens: 1 });
telemetry.stampToolStart({ type: "tool_start", kind: "bash", text: "git status" });
process.stdout.write(JSON.stringify({ records: [], telemetry: telemetry.snapshot() }));
`)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: () => {},
});
telemetry.beginTurn({ input_tokens: 3, output_tokens: 1 });
process.stdout.write(JSON.stringify({ records: [], telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Telemetry.Turns, 1)
	assert.Equal(t, 1, payload.Telemetry.NumTurns)
	assert.Equal(t, 1, payload.Telemetry.Turns[0].Turn)
	assert.Equal(t, float64(3), payload.Telemetry.Turns[0].Usage["input_tokens"])
}

func TestTurnTelemetrySkipsDuplicateUsage(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const records = [];
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: (record) => records.push(record),
});
telemetry.beginTurn({ input_tokens: 2, output_tokens: 5, cache_read_input_tokens: 1448 });
telemetry.beginTurn({ input_tokens: 2, output_tokens: 5, cache_read_input_tokens: 1448 });
telemetry.beginTurn({ input_tokens: 2, output_tokens: 21, cache_read_input_tokens: 5394 });
process.stdout.write(JSON.stringify({ records, telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Telemetry.Turns, 2)
	assert.Equal(t, 2, payload.Telemetry.NumTurns)
	assert.Equal(t, 1, payload.Telemetry.Turns[0].Turn)
	assert.Equal(t, 2, payload.Telemetry.Turns[1].Turn)
	assert.Equal(t, float64(21), payload.Telemetry.Turns[1].Usage["output_tokens"])
	require.Len(t, payload.Records, 2)
}

func TestTurnTelemetryKeepsSameUsageWhenTurnHasTools(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const records = [];
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: (record) => records.push(record),
});
telemetry.beginTurn({ input_tokens: 2, output_tokens: 5 });
telemetry.beginTurn({ input_tokens: 2, output_tokens: 5 }, { hasTools: true });
process.stdout.write(JSON.stringify({ records, telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Telemetry.Turns, 2)
	assert.Equal(t, 2, payload.Telemetry.NumTurns)
}

func TestTurnTelemetryForceNewKeepsSameUsageTurns(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: () => undefined,
});
telemetry.beginTurn({ input_tokens: 11, output_tokens: 3 });
telemetry.beginTurn({ input_tokens: 11, output_tokens: 3 }, { forceNew: true });
process.stdout.write(JSON.stringify({ telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Telemetry.Turns, 2)
	assert.Equal(t, float64(11), payload.Telemetry.Turns[0].Usage["input_tokens"])
	assert.Equal(t, float64(11), payload.Telemetry.Turns[1].Usage["input_tokens"])
}

func TestTurnTelemetryUpdatesUsageForTheSameMessageID(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const records = [];
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: (record) => records.push(record),
});
telemetry.beginTurn({ input_tokens: 2, output_tokens: 1, cache_read_input_tokens: 100 }, { messageId: "msg_1", message: "partial" });
telemetry.beginTurn({ input_tokens: 2, output_tokens: 21, cache_read_input_tokens: 100 }, { messageId: "msg_1", message: "full answer" });
telemetry.beginTurn({ input_tokens: 4, output_tokens: 8 }, { messageId: "msg_2" });
process.stdout.write(JSON.stringify({ records, telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Telemetry.Turns, 2)
	assert.Equal(t, float64(21), payload.Telemetry.Turns[0].Usage["output_tokens"])
	assert.Equal(t, float64(2), payload.Telemetry.Turns[0].Usage["input_tokens"])
	assert.Equal(t, float64(100), payload.Telemetry.Turns[0].Usage["cache_read_input_tokens"])
	assert.Equal(t, "full answer", payload.Telemetry.Turns[0].Message)
	assert.Equal(t, float64(8), payload.Telemetry.Turns[1].Usage["output_tokens"])
	require.Len(t, payload.Records, 3)
	assert.Equal(t, float64(1), payload.Records[1]["turn"])
	updated, ok := payload.Records[1]["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(21), updated["output_tokens"])
}

func TestTurnTelemetryMergesPartialStreamUsageIntoCurrentTurn(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const records = [];
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: (record) => records.push(record),
});
telemetry.beginTurn({ input_tokens: 2, output_tokens: 1, cache_read_input_tokens: 29497 }, { messageId: "msg_25" });
telemetry.mergeCurrentUsage({ output_tokens: 47 });
process.stdout.write(JSON.stringify({ records, telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Telemetry.Turns, 1)
	assert.Equal(t, float64(47), payload.Telemetry.Turns[0].Usage["output_tokens"])
	assert.Equal(t, float64(2), payload.Telemetry.Turns[0].Usage["input_tokens"])
	assert.Equal(t, float64(29497), payload.Telemetry.Turns[0].Usage["cache_read_input_tokens"])
}

func TestTurnTelemetryReplacesUsageForAnEarlierTurn(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const records = [];
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: (record) => records.push(record),
});
telemetry.beginTurn({});
telemetry.beginTurn({});
telemetry.replaceTurnUsage(1, { input_tokens: 10, output_tokens: 2 });
process.stdout.write(JSON.stringify({ records, telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Telemetry.Turns, 2)
	assert.Equal(t, float64(10), payload.Telemetry.Turns[0].Usage["input_tokens"])
	assert.Equal(t, float64(2), payload.Telemetry.Turns[0].Usage["output_tokens"])
	lastRecord := payload.Records[len(payload.Records)-1]
	assert.Equal(t, float64(1), lastRecord["turn"])
}

func TestTurnTelemetryAppliesBilledOutputGapToLastTurn(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const records = [];
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: (record) => records.push(record),
});
telemetry.beginTurn({ input_tokens: 1, output_tokens: 2, cache_read_input_tokens: 1448, cache_creation_input_tokens: 815 });
telemetry.beginTurn({ input_tokens: 2, output_tokens: 1, cache_read_input_tokens: 20409, cache_creation_input_tokens: 2663 });
const result = telemetry.attachToResult({
  type: "result",
  usage: {
    input_tokens: 13,
    output_tokens: 2057,
    cache_read_input_tokens: 71247,
    cache_creation_input_tokens: 21624,
  },
});
process.stdout.write(JSON.stringify({ records, telemetry: result.telemetry }));
`)

	require.Len(t, payload.Telemetry.Turns, 2)
	assert.Equal(t, float64(2), payload.Telemetry.Turns[0].Usage["output_tokens"])
	assert.Equal(t, float64(1), payload.Telemetry.Turns[0].Usage["input_tokens"])
	assert.Equal(t, float64(2055), payload.Telemetry.Turns[1].Usage["output_tokens"])
	assert.Equal(t, float64(2), payload.Telemetry.Turns[1].Usage["input_tokens"])
	assert.Equal(t, float64(20409), payload.Telemetry.Turns[1].Usage["cache_read_input_tokens"])
	assert.Equal(t, float64(2663), payload.Telemetry.Turns[1].Usage["cache_creation_input_tokens"])
	assert.Equal(t, float64(2057), payload.Telemetry.Usage["output_tokens"])
	assert.Equal(t, float64(13), payload.Telemetry.Usage["input_tokens"])
	lastRecord := payload.Records[len(payload.Records)-1]
	assert.Equal(t, "turn", lastRecord["type"])
	assert.Equal(t, float64(2), lastRecord["turn"])
	usage, ok := lastRecord["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(2055), usage["output_tokens"])
}

func TestTurnTelemetryStoresAgentMessage(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const records = [];
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: (record) => records.push(record),
});
telemetry.beginTurn({ input_tokens: 10, output_tokens: 2 }, { message: "I will inspect the remotes." });
process.stdout.write(JSON.stringify({ records, telemetry: telemetry.snapshot() }));
`)

	require.Len(t, payload.Records, 1)
	assert.Equal(t, "I will inspect the remotes.", payload.Records[0]["message"])
	require.Len(t, payload.Telemetry.Turns, 1)
	assert.Equal(t, "I will inspect the remotes.", payload.Telemetry.Turns[0].Message)
}

func TestTurnTelemetryAttachesJoinedSeriesToResult(t *testing.T) {
	t.Parallel()

	taskDir := t.TempDir()
	writeTurnTelemetryScript(t, taskDir)

	payload := runTurnTelemetryHarness(t, taskDir, `
const { createTurnTelemetry } = require(process.env.TURN_TELEMETRY_SCRIPT);
const telemetry = createTurnTelemetry({
  taskDir: process.env.SUPERPLANE_TASK_DIR,
  write: () => {},
});
telemetry.beginTurn({ input_tokens: 4, output_tokens: 1 });
telemetry.stampToolStart({ type: "tool_start", kind: "bash", text: "true" });
const result = telemetry.attachToResult({ type: "result", usage: { input_tokens: 4, output_tokens: 1 } });
process.stdout.write(JSON.stringify({ records: [], telemetry: result.telemetry }));
`)

	assert.Equal(t, 1, payload.Telemetry.NumTurns)
	assert.Equal(t, "true", payload.Telemetry.Turns[0].Tools[0]["text"])
}

type turnTelemetryHarnessPayload struct {
	Records   []map[string]any `json:"records"`
	Telemetry struct {
		NumTurns   int            `json:"num_turns"`
		Usage      map[string]any `json:"usage"`
		ToolCounts map[string]int `json:"tool_counts"`
		Turns      []struct {
			Turn    int              `json:"turn"`
			Usage   map[string]any   `json:"usage"`
			Tools   []map[string]any `json:"tools"`
			Message string           `json:"message"`
		} `json:"turns"`
	} `json:"telemetry"`
}

func writeTurnTelemetryScript(t *testing.T, taskDir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "turn_telemetry.js"), []byte(TurnTelemetryScript), 0o644))
}

func runTurnTelemetryHarness(t *testing.T, taskDir, body string) turnTelemetryHarnessPayload {
	t.Helper()
	script := filepath.Join(taskDir, "harness.js")
	require.NoError(t, os.WriteFile(script, []byte(body), 0o644))
	cmd := exec.Command("node", script)
	cmd.Env = append(os.Environ(),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"TURN_TELEMETRY_SCRIPT="+filepath.Join(taskDir, "turn_telemetry.js"),
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var payload turnTelemetryHarnessPayload
	require.NoError(t, json.Unmarshal(out, &payload), string(out))
	return payload
}
