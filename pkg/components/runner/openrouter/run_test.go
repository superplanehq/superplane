package openrouter

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

func TestOpenRouterModelIdPrefixesCatalogIds(t *testing.T) {
	assert.Equal(t, "openrouter/anthropic/claude-sonnet-4-6", jsString(t, "openRouterModelId", "anthropic/claude-sonnet-4-6"))
	assert.Equal(t, "openrouter/anthropic/claude-sonnet-4-6", jsString(t, "openRouterModelId", "openrouter/anthropic/claude-sonnet-4-6"))
}

func TestClassifyOpenRouterErrorDetectsRateLimitsAndHardFailures(t *testing.T) {
	rpm := "Rate limit exceeded: new-account-rpm/x-ai/grok-4.6-20260810. Rate limit reached: new accounts are limited to 20 requests per minute for this model. Please retry shortly."
	assert.Equal(t, "rate_limit", jsClassify(t, rpm))
	assert.Equal(t, "rate_limit", jsClassify(t, "HTTP 429 Too Many Requests"))
	assert.Equal(t, "hard", jsClassify(t, "Invalid API key"))
	assert.Equal(t, "hard", jsClassify(t, "Insufficient credits"))
	assert.Equal(t, "hard", jsClassify(t, "unknown model: nope"))
	assert.Equal(t, "retryable", jsClassify(t, "HTTP 503 Service Unavailable"))
	assert.Equal(t, "retryable", jsClassify(t, "Provider returned error"))
	assert.Equal(t, "retryable", jsClassify(t, "The model is currently overloaded"))
	assert.Equal(t, "other", jsClassify(t, "opencode crashed"))
}

func TestOpencodeRunArgsIncludesJSONAutoPureAndPrefix(t *testing.T) {
	args := jsOpencodeArgs(t, map[string]any{
		"model":   "x-ai/grok-4.6",
		"prompt":  "do the work",
		"cwd":     "/tmp/repo",
		"session": "",
	})
	assert.Equal(t, []string{
		"--pure", "run", "--format", "json", "--auto",
		"-m", "openrouter/x-ai/grok-4.6",
		"--dir", "/tmp/repo",
		"do the work",
	}, args)
	assert.NotContains(t, args, "--print-logs")
	assert.NotContains(t, args, "--agent")
}

func TestOpencodeRunArgsContinuesSession(t *testing.T) {
	args := jsOpencodeArgs(t, map[string]any{
		"model":     "anthropic/claude-sonnet-4-6",
		"prompt":    "next",
		"cwd":       "/tmp/repo",
		"sessionID": "ses_abc",
	})
	assert.Contains(t, args, "--session")
	assert.Contains(t, args, "ses_abc")
	assert.Contains(t, args, "-m")
	assert.Contains(t, args, "openrouter/anthropic/claude-sonnet-4-6")
}

func TestBuildOpenCodeConfigWritesBaseURLAndPlanningMCP(t *testing.T) {
	config := jsBuildConfig(t, "/task", map[string]string{
		"OPENROUTER_API_KEY":             "sk-or",
		"OPENROUTER_BASE_URL":            "https://proxy.example/openrouter",
		"SUPERPLANE_PLANNING_SESSION_ID": "session-1",
	})
	provider, _ := config["provider"].(map[string]any)
	openrouter, _ := provider["openrouter"].(map[string]any)
	options, _ := openrouter["options"].(map[string]any)
	assert.Equal(t, "sk-or", options["apiKey"])
	assert.Equal(t, "https://proxy.example/openrouter", options["baseURL"])
	permission, _ := config["permission"].(map[string]any)
	assert.Equal(t, "deny", permission["edit"])
	assert.Equal(t, "deny", permission["question"])
	assert.Equal(t, "allow", permission["*"])
	mcp, _ := config["mcp"].(map[string]any)
	superplane, _ := mcp["superplane"].(map[string]any)
	assert.Equal(t, "local", superplane["type"])
	command, _ := superplane["command"].([]any)
	require.Equal(t, 2, len(command))
	assert.Equal(t, "node", command[0])
	assert.Equal(t, "/task/planning_session_mcp.js", command[1])
}

func TestBuildOpenCodeConfigAllowsEditsOutsidePlanning(t *testing.T) {
	config := jsBuildConfig(t, "/task", map[string]string{})
	permission, _ := config["permission"].(map[string]any)
	assert.Equal(t, "allow", permission["*"])
	assert.Nil(t, permission["edit"])
	assert.Nil(t, config["mcp"])
}

func TestBuildOpenCodeConfigDisablesFallbacksForSelectedModel(t *testing.T) {
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(map[string]any{
		"taskDir": "/task",
		"env":     map[string]string{"OPENROUTER_API_KEY": "sk-or"},
		"models":  []string{"x-ai/grok-4.6"},
	})
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { buildOpenCodeConfig } = require(process.argv[1]); process.stdout.write(JSON.stringify(buildOpenCodeConfig(JSON.parse(process.argv[2]))));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var config map[string]any
	require.NoError(t, json.Unmarshal(out, &config))
	provider, _ := config["provider"].(map[string]any)
	openrouter, _ := provider["openrouter"].(map[string]any)
	models, _ := openrouter["models"].(map[string]any)
	require.Contains(t, models, "x-ai/grok-4.6")
	assert.Len(t, models, 1)
	grok, _ := models["x-ai/grok-4.6"].(map[string]any)
	options, _ := grok["options"].(map[string]any)
	routing, _ := options["provider"].(map[string]any)
	assert.Equal(t, false, routing["allow_fallbacks"])
	assert.Equal(t, "throughput", routing["sort"])
}

func TestFormatOpenCodeJsonLinesEmitsWorkingLineOnStepStart(t *testing.T) {
	output := runOpenCodeFormatter(t, []string{
		`{"type":"step_start","sessionID":"ses_1","part":{"type":"step-start"}}`,
	})
	assert.Contains(t, output, "OpenCode started")
}

func TestFormatOpenCodeJsonLinesEmitsToolRecords(t *testing.T) {
	output := runOpenCodeFormatter(t, []string{
		`{"type":"step_start","sessionID":"ses_1","part":{"type":"step-start"}}`,
		`{"type":"text","sessionID":"ses_1","part":{"type":"text","text":"I will check git status."}}`,
		`{"type":"tool_use","sessionID":"ses_1","part":{"id":"t1","callID":"call_1","tool":"bash","state":{"status":"completed","input":{"command":"git status"},"output":"On branch main"}}}`,
		`{"type":"step_finish","sessionID":"ses_1","part":{"type":"step-finish","reason":"stop","cost":0.002,"tokens":{"input":100,"output":20,"reasoning":0,"cache":{"read":0,"write":0}}}}`,
	})
	assert.Contains(t, output, "I will check git status.")
	assert.Contains(t, output, "On branch main")
	records := liveLogRecords(t, output)
	require.GreaterOrEqual(t, len(records), 2)
	assert.Equal(t, "tool_start", records[0]["type"])
	assert.Equal(t, "bash", records[0]["kind"])
	assert.Equal(t, "git status", records[0]["text"])
	assert.Equal(t, float64(1), records[0]["turn"])
	end := records[len(records)-1]
	assert.Equal(t, "tool_end", end["type"])
	assert.Equal(t, "passed", end["status"])
	assert.Contains(t, output, `"type":"turn"`)
}

func TestFormatOpenCodeJsonLinesMarksFailedTools(t *testing.T) {
	output := runOpenCodeFormatter(t, []string{
		`{"type":"tool_use","sessionID":"ses_1","part":{"callID":"call_read","tool":"read","state":{"status":"error","input":{"path":"missing.txt"},"error":"ENOENT"}}}`,
	})
	records := liveLogRecords(t, output)
	require.GreaterOrEqual(t, len(records), 2)
	assert.Equal(t, "read", records[0]["kind"])
	assert.Equal(t, "failed", records[len(records)-1]["status"])
	assert.Contains(t, output, "ENOENT")
}

func TestFormatOpenCodeJsonLinesTreatsNonRateLimitErrorAsFailed(t *testing.T) {
	failed := formatOpenCodeJSONLinesFailed(t, []string{
		`{"type":"error","error":{"name":"APIError","data":{"message":"Invalid API key"}}}`,
	})
	assert.True(t, failed)
}

func TestFormatOpenCodeJsonLinesDoesNotFailRateLimitError(t *testing.T) {
	failed := formatOpenCodeJSONLinesFailed(t, []string{
		`{"type":"error","error":{"data":{"message":"Rate limit exceeded: new-account-rpm/x-ai/grok-4.6-20260810. Please retry shortly."}}}`,
	})
	assert.False(t, failed)
}

func TestFormatTurnResultWritesClaudeStyleDoneLine(t *testing.T) {
	output := runTurnResult(t, map[string]any{
		"is_error":       false,
		"num_turns":      1,
		"duration_ms":    1600,
		"total_cost_usd": 0.0022,
	})
	assert.Equal(t, "✓ done · 1 turns · $0.0022 · 1.6s\n", output)
}

func TestRunPromptRetriesSelectedModelAfterRateLimit(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6-20260810. Please retry shortly."),
			successSpawn("working"),
		},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 2)
	assert.Contains(t, result.spawns[0], "openrouter/x-ai/grok-4.6")
	assert.Contains(t, result.spawns[1], "openrouter/x-ai/grok-4.6")
	assert.NotContains(t, result.spawns[1], "anthropic/")
	assert.Equal(t, []float64{30000}, result.sleeps)
	requireLiveLogLine(t, result.output, "Starting OpenCode · openrouter/x-ai/grok-4.6")
	requireLiveLogLine(t, result.output, "Calling OpenCode · openrouter/x-ai/grok-4.6 (attempt 1 of 4)")
	requireLiveLogLine(t, result.output, "Rate limit on x-ai/grok-4.6. Waiting 30 seconds, then retrying (attempt 2 of 4).")
	requireLiveLogLine(t, result.output, "Calling OpenCode · openrouter/x-ai/grok-4.6 (attempt 2 of 4)")
	requireNoBareStdoutLine(t, result.output, "Starting OpenCode")
	requireNoBareStdoutLine(t, result.output, "Calling OpenCode")
	requireNoBareStdoutLine(t, result.output, "Waiting 30 seconds")
	assert.NotContains(t, result.spawns[1], "--session")
	assert.Regexp(t, `✓ done · \d+ turns`, result.output)
	payload := resultPayload(t, result.resultFile)
	assert.Equal(t, "working", payload["result"])
}

func TestRunPromptRetriesSelectedModelAfterTemporaryProviderError(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			{
				ExitCode: 1,
				Stderr:   "HTTP 503 Service Unavailable",
				Stdout: []string{
					`{"type":"error","sessionID":"ses_fail","error":{"name":"APIError","message":"Provider returned error","data":{"statusCode":503}}}`,
				},
			},
			successSpawn("recovered"),
		},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 2)
	assert.Contains(t, result.spawns[0], "openrouter/x-ai/grok-4.6")
	assert.Contains(t, result.spawns[1], "openrouter/x-ai/grok-4.6")
	assert.Contains(t, result.spawns[1], "--session")
	assert.Contains(t, result.spawns[1], "ses_fail")
	assert.Equal(t, []float64{30000}, result.sleeps)
	requireLiveLogLine(t, result.output, "Temporary error on x-ai/grok-4.6. Waiting 30 seconds, then retrying (attempt 2 of 4).")
	requireLiveLogLine(t, result.output, "Calling OpenCode · openrouter/x-ai/grok-4.6 (attempt 2 of 4)")
}

func TestRunPromptContinuesSessionWhenRetryableFailureMadeProgress(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			{
				ExitCode: 1,
				Stderr:   "Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly.",
				Stdout: []string{
					`{"type":"step_start","sessionID":"ses_work","part":{"type":"step-start"}}`,
					`{"type":"tool_use","sessionID":"ses_work","part":{"callID":"call_1","tool":"bash","state":{"status":"completed","input":{"command":"git status"},"output":"On branch main"}}}`,
					`{"type":"error","sessionID":"ses_work","error":{"data":{"message":"Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."}}}`,
				},
			},
			successSpawn("ok"),
		},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 2)
	assert.NotContains(t, result.spawns[0], "--session")
	assert.Contains(t, result.spawns[1], "--session")
	assert.Contains(t, result.spawns[1], "ses_work")
}

func TestRunPromptRetriesNestedRateLimitOnSelectedModel(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			{
				ExitCode: 1,
				Stdout: []string{
					`{"type":"error","error":{"name":"APIError","message":"Provider returned error","data":{"statusCode":429,"responseBody":"{\"error\":{\"code\":429,\"message\":\"Rate limit exceeded\"}}"}}}`,
				},
			},
			successSpawn("ok"),
		},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 2)
	assert.Contains(t, result.spawns[1], "openrouter/x-ai/grok-4.6")
	assert.Equal(t, []float64{30000}, result.sleeps)
	requireLiveLogLine(t, result.output, "Rate limit on x-ai/grok-4.6. Waiting 30 seconds, then retrying (attempt 2 of 4).")
}

func TestRunPromptUsesProgressiveWaitWithoutRetryAfter(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."),
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."),
			{
				ExitCode: 1,
				Stderr:   "HTTP 503 Service Unavailable",
				Stdout: []string{
					`{"type":"error","error":{"name":"APIError","message":"Provider returned error","data":{"statusCode":503}}}`,
				},
			},
			successSpawn("ok"),
		},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 4)
	assert.Equal(t, []float64{30000, 45000, 60000}, result.sleeps)
	requireLiveLogLine(t, result.output, "Waiting 30 seconds, then retrying (attempt 2 of 4).")
	requireLiveLogLine(t, result.output, "Waiting 45 seconds, then retrying (attempt 3 of 4).")
	requireLiveLogLine(t, result.output, "Waiting 60 seconds, then retrying (attempt 4 of 4).")
	requireLiveLogLine(t, result.output, "Calling OpenCode · openrouter/x-ai/grok-4.6 (attempt 1 of 4)")
	requireLiveLogLine(t, result.output, "Calling OpenCode · openrouter/x-ai/grok-4.6 (attempt 4 of 4)")
	requireNoBareStdoutLine(t, result.output, "Waiting 45 seconds")
}

func TestRunPromptHonorsRetryAfterOverProgressiveWait(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly. Retry-After: 90"),
			successSpawn("ok"),
		},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 2)
	assert.Equal(t, []float64{90000}, result.sleeps)
	requireLiveLogLine(t, result.output, "Rate limit on x-ai/grok-4.6. Waiting 90 seconds, then retrying (attempt 2 of 4).")
	assert.NotContains(t, result.output, "Waiting 30 seconds")
}

func TestRunPromptStopsAfterFourRateLimitAttempts(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."),
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."),
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."),
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."),
		},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 4)
	assert.Equal(t, []float64{30000, 45000, 60000}, result.sleeps)
	requireLiveLogLine(t, result.output, "Calling OpenCode · openrouter/x-ai/grok-4.6 (attempt 4 of 4)")
	requireLiveLogLine(t, result.output, "Rate limit on x-ai/grok-4.6. Stopped after 4 attempts.")
	requireNoBareStdoutLine(t, result.output, "Stopped after 4 attempts")
	assert.NotContains(t, result.output, "attempt 5")
	assert.Regexp(t, `✗ failed`, result.output)
}

func TestRunPromptSucceedsWhenOpenCodeExitsNonZeroAfterReply(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "google/gemma-4-31b-it",
		env:   map[string]string{"SUPERPLANE_PLANNING_SESSION_ID": "plan-1"},
		spawns: []spawnScript{{
			ExitCode: 1,
			Stdout: []string{
				`{"type":"step_start","sessionID":"ses_hello","part":{"type":"step-start"}}`,
				`{"type":"text","sessionID":"ses_hello","part":{"type":"text","text":"Hello! How can I help you today?"}}`,
				`{"type":"step_finish","sessionID":"ses_hello","part":{"type":"step-finish","tokens":{"input":10,"output":8}}}`,
			},
		}},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 1)
	payload := resultPayload(t, result.resultFile)
	assert.Equal(t, "Hello! How can I help you today?", payload["result"])
	assert.Regexp(t, `✓ done`, result.output)
}

func TestRunPromptKeepsBilledUsageWhenRetrySucceedsWithoutStepFinish(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			{
				ExitCode: 1,
				Stderr:   "Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly.",
				Stdout: []string{
					`{"type":"step_start","sessionID":"ses_bill","part":{"type":"step-start"}}`,
					`{"type":"text","sessionID":"ses_bill","part":{"type":"text","text":"partial"}}`,
					`{"type":"step_finish","sessionID":"ses_bill","part":{"type":"step-finish","cost":0.012,"tokens":{"input":10,"output":8}}}`,
					`{"type":"error","error":{"data":{"message":"Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."}}}`,
				},
			},
			{
				ExitCode: 0,
				Stdout: []string{
					`{"type":"text","sessionID":"ses_ok","part":{"type":"text","text":"done"}}`,
				},
			},
		},
	})
	assert.Equal(t, 0, result.exitCode)
	assertResultUsage(t, resultPayload(t, result.resultFile), 10, 8, 0.012)
}

func TestRunPromptAddsUsageAcrossBilledRetries(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{
			{
				ExitCode: 1,
				Stderr:   "Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly.",
				Stdout: []string{
					`{"type":"step_finish","sessionID":"ses_1","part":{"type":"step-finish","cost":0.01,"tokens":{"input":10,"output":8}}}`,
					`{"type":"error","error":{"data":{"message":"Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."}}}`,
				},
			},
			{
				ExitCode: 0,
				Stdout: []string{
					`{"type":"text","sessionID":"ses_2","part":{"type":"text","text":"done"}}`,
					`{"type":"step_finish","sessionID":"ses_2","part":{"type":"step-finish","cost":0.005,"tokens":{"input":5,"output":3}}}`,
				},
			},
		},
	})
	assert.Equal(t, 0, result.exitCode)
	assertResultUsage(t, resultPayload(t, result.resultFile), 15, 11, 0.015)
}

func TestRunPromptUsesSessionStoreWhenJsonlMissesStepFinish(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		sessionUsage: []map[string]any{
			{
				"input_tokens":   100,
				"output_tokens":  20,
				"total_cost_usd": 0.1,
			},
			{
				"input_tokens":   140,
				"output_tokens":  32,
				"total_cost_usd": 0.12,
			},
		},
		spawns: []spawnScript{{
			ExitCode: 0,
			Stdout: []string{
				`{"type":"text","sessionID":"ses_ok","part":{"type":"text","text":"done"}}`,
			},
		}},
	})
	assert.Equal(t, 0, result.exitCode)
	assertResultUsage(t, resultPayload(t, result.resultFile), 40, 12, 0.02)
}

func TestRunPromptReadsSessionJsonWhenJsonlMissesStepFinish(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{{
			ExitCode: 0,
			Stdout: []string{
				`{"type":"text","sessionID":"ses_ok","part":{"type":"text","text":"done"}}`,
			},
			SessionParts: []string{
				`{"type":"step-finish","cost":0.02,"tokens":{"input":40,"output":12}}`,
			},
		}},
	})
	assert.Equal(t, 0, result.exitCode)
	assertResultUsage(t, resultPayload(t, result.resultFile), 40, 12, 0.02)
}

func TestReadSessionUsageSumsStepFinishJsonParts(t *testing.T) {
	dir := t.TempDir()
	partDir := filepath.Join(dir, "xdg", "data", "opencode", "storage", "part", "msg_1")
	require.NoError(t, os.MkdirAll(partDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(partDir, "prt_1.json"), []byte(`{"type":"step-finish","cost":0.01,"tokens":{"input":10,"output":8,"reasoning":1,"cache":{"read":4,"write":2}}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(partDir, "prt_text.json"), []byte(`{"type":"text","text":"hi"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(partDir, "prt_2.json"), []byte(`{"part":{"type":"step-finish","cost":0.005,"tokens":{"input":5,"output":3,"cache":{"read":1,"write":0}}}}`), 0o644))
	usage := jsReadSessionUsage(t, dir)
	assert.Equal(t, float64(15), usage["input_tokens"])
	assert.Equal(t, float64(11), usage["output_tokens"])
	assert.Equal(t, float64(5), usage["cache_read_input_tokens"])
	assert.Equal(t, float64(2), usage["cache_creation_input_tokens"])
	assert.Equal(t, float64(1), usage["reasoning_tokens"])
	assert.InDelta(t, 0.015, usage["total_cost_usd"], 1e-9)
}

func TestReadSessionUsageSumsStepFinishSqliteParts(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "xdg", "data", "opencode")
	require.NoError(t, os.MkdirAll(dbDir, 0o755))
	dbPath := filepath.Join(dbDir, "opencode.db")
	cmd := exec.Command("node", "-e", `
let DatabaseSync;
try {
  ({ DatabaseSync } = require("node:sqlite"));
} catch {
  process.exit(2);
}
const db = new DatabaseSync(process.env.OPENCODE_DB);
db.exec("CREATE TABLE part (id TEXT PRIMARY KEY, data TEXT NOT NULL)");
const insert = db.prepare("INSERT INTO part (id, data) VALUES (?, ?)");
insert.run("prt_1", JSON.stringify({
  type: "step-finish",
  cost: 0.01,
  tokens: { input: 10, output: 8, reasoning: 1, cache: { read: 4, write: 2 } },
}));
insert.run("prt_2", JSON.stringify({ type: "text", text: "hi" }));
db.close();
`)
	cmd.Env = append(os.Environ(), "OPENCODE_DB="+dbPath, "NODE_NO_WARNINGS=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok && exitErr.ExitCode() == 2 {
			t.Skip("node:sqlite is not available")
		}
		require.NoError(t, err, string(out))
	}
	usage := jsReadSessionUsage(t, dir)
	assert.Equal(t, float64(10), usage["input_tokens"])
	assert.Equal(t, float64(8), usage["output_tokens"])
	assert.Equal(t, float64(4), usage["cache_read_input_tokens"])
	assert.Equal(t, float64(2), usage["cache_creation_input_tokens"])
	assert.Equal(t, float64(1), usage["reasoning_tokens"])
	assert.InDelta(t, 0.01, usage["total_cost_usd"], 1e-9)
}

func TestRunPromptSucceedsWhenOpenCodeExitsNonZeroAfterTwoFinishedSteps(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "google/gemma-4-31b-it",
		env:   map[string]string{"SUPERPLANE_PLANNING_SESSION_ID": "plan-1"},
		spawns: []spawnScript{{
			ExitCode: 1,
			Stdout: []string{
				`{"type":"step_start","sessionID":"ses_hello","part":{"type":"step-start"}}`,
				`{"type":"text","sessionID":"ses_hello","part":{"type":"text","text":"Hello! How can I help you today?"}}`,
				`{"type":"step_finish","sessionID":"ses_hello","part":{"type":"step-finish","tokens":{"input":10,"output":8}}}`,
				`{"type":"step_start","sessionID":"ses_hello","part":{"type":"step-start"}}`,
				`{"type":"text","sessionID":"ses_hello","part":{"type":"text","text":"I opened the repository."}}`,
				`{"type":"step_finish","sessionID":"ses_hello","part":{"type":"step-finish","tokens":{"input":5,"output":3}}}`,
			},
		}},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 1)
	payload := resultPayload(t, result.resultFile)
	assert.Equal(t, "I opened the repository.", payload["result"])
	usage, ok := payload["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(15), usage["input_tokens"])
	assert.Equal(t, float64(11), usage["output_tokens"])
	assert.Regexp(t, `✓ done`, result.output)
}

func TestRunPromptFailsWhenOpenCodeExitsNonZeroAfterReplyOnLineAutomation(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "google/gemma-4-31b-it",
		spawns: []spawnScript{{
			ExitCode: 1,
			Stdout: []string{
				`{"type":"step_start","sessionID":"ses_hello","part":{"type":"step-start"}}`,
				`{"type":"text","sessionID":"ses_hello","part":{"type":"text","text":"Hello! How can I help you today?"}}`,
				`{"type":"step_finish","sessionID":"ses_hello","part":{"type":"step-finish","tokens":{"input":10,"output":8}}}`,
			},
		}},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Regexp(t, `✗ failed`, result.output)
}

func TestRunPromptFailsWhenOpenCodeExitsNonZeroAfterLaterPartialStep(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "google/gemma-4-31b-it",
		env:   map[string]string{"SUPERPLANE_PLANNING_SESSION_ID": "plan-1"},
		spawns: []spawnScript{{
			ExitCode: 1,
			Stdout: []string{
				`{"type":"step_start","sessionID":"ses_hello","part":{"type":"step-start"}}`,
				`{"type":"text","sessionID":"ses_hello","part":{"type":"text","text":"Hello! How can I help you today?"}}`,
				`{"type":"step_finish","sessionID":"ses_hello","part":{"type":"step-finish","tokens":{"input":10,"output":8}}}`,
				`{"type":"step_start","sessionID":"ses_hello","part":{"type":"step-start"}}`,
				`{"type":"text","sessionID":"ses_hello","part":{"type":"text","text":"I found a few files to inspect"}}`,
			},
		}},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Regexp(t, `✗ failed`, result.output)
	payload := resultPayload(t, result.resultFile)
	usage, ok := payload["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(10), usage["input_tokens"])
	assert.Equal(t, float64(8), usage["output_tokens"])
}

func TestRunPromptFailsWhenOpenCodeExitsNonZeroWithPartialText(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "google/gemma-4-31b-it",
		env:   map[string]string{"SUPERPLANE_PLANNING_SESSION_ID": "plan-1"},
		spawns: []spawnScript{{
			ExitCode: 1,
			Stdout: []string{
				`{"type":"step_start","sessionID":"ses_hello","part":{"type":"step-start"}}`,
				`{"type":"text","sessionID":"ses_hello","part":{"type":"text","text":"Hello! How can I help"}}`,
			},
		}},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Regexp(t, `✗ failed`, result.output)
}

func TestRunPromptFailsWhenOpenCodeExitsNonZeroWithUsageOnly(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "google/gemma-4-31b-it",
		env:   map[string]string{"SUPERPLANE_PLANNING_SESSION_ID": "plan-1"},
		spawns: []spawnScript{{
			ExitCode: 1,
			Stdout: []string{
				`{"type":"step_start","sessionID":"ses_hello","part":{"type":"step-start"}}`,
				`{"type":"step_finish","sessionID":"ses_hello","part":{"type":"step-finish","tokens":{"input":10,"output":8}}}`,
			},
		}},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Regexp(t, `✗ failed`, result.output)
}

func TestRunPromptFailsWhenOpenCodeExitsNonZeroWithoutReply(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "google/gemma-4-31b-it",
		spawns: []spawnScript{{
			ExitCode: 1,
			Stderr:   "opencode crashed",
		}},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Regexp(t, `✗ failed`, result.output)
}

func TestRunPromptFailsImmediatelyOnInvalidAPIKey(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "anthropic/claude-sonnet-4-6",
		spawns: []spawnScript{
			{
				ExitCode: 1,
				Stderr:   "Invalid API key",
				Stdout:   []string{`{"type":"error","error":{"data":{"message":"Invalid API key"}}}`},
			},
		},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Empty(t, result.sleeps)
	assert.NotContains(t, result.output, "Waiting")
	assert.NotContains(t, result.output, "Retrying")
	requireLiveLogLine(t, result.output, "Invalid API key")
	assert.Regexp(t, `✗ failed`, result.output)
}

func TestRunPromptContinuesSessionOnLaterPrompt(t *testing.T) {
	dir := t.TempDir()
	writeTaskHelpers(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "prompt_count"), []byte("0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte("first"), 0o644))

	first := runPromptInDir(t, dir, "prompt.txt", "anthropic/claude-sonnet-4-6", []spawnScript{{
		ExitCode: 0,
		Stdout: []string{
			`{"type":"step_start","sessionID":"ses_keep","part":{"type":"step-start"}}`,
			`{"type":"text","sessionID":"ses_keep","part":{"type":"text","text":"first done"}}`,
			`{"type":"step_finish","sessionID":"ses_keep","part":{"type":"step-finish","tokens":{"input":1,"output":1}}}`,
		},
	}}, nil, nil, nil)
	assert.Equal(t, 0, first.exitCode)
	session, err := os.ReadFile(filepath.Join(dir, "opencode_session"))
	require.NoError(t, err)
	assert.Equal(t, "ses_keep\n", string(session))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "prompt2.txt"), []byte("second"), 0o644))
	second := runPromptInDir(t, dir, "prompt2.txt", "anthropic/claude-sonnet-4-6", []spawnScript{{
		ExitCode: 0,
		Stdout: []string{
			`{"type":"text","sessionID":"ses_keep","part":{"type":"text","text":"second done"}}`,
			`{"type":"step_finish","sessionID":"ses_keep","part":{"type":"step-finish","tokens":{"input":1,"output":1}}}`,
		},
	}}, nil, nil, nil)
	assert.Equal(t, 0, second.exitCode)
	require.NotEmpty(t, second.spawns)
	assert.Contains(t, second.spawns[0], "--session")
	assert.Contains(t, second.spawns[0], "ses_keep")
	requireLiveLogLine(t, second.output, "Continuing OpenCode session on openrouter/anthropic/claude-sonnet-4-6")
	requireLiveLogLine(t, second.output, "Calling OpenCode · openrouter/anthropic/claude-sonnet-4-6 (attempt 1 of 4)")
	requireNoBareStdoutLine(t, second.output, "Continuing OpenCode session")
}

func TestRunPromptWritesOpenRouterBaseURLIntoConfig(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "anthropic/claude-sonnet-4-6",
		env: map[string]string{
			"OPENROUTER_BASE_URL": "https://hosted.example/openrouter",
		},
		spawns: []spawnScript{{
			ExitCode: 0,
			Stdout: []string{
				`{"type":"text","sessionID":"ses_cfg","part":{"type":"text","text":"ok"}}`,
				`{"type":"step_finish","sessionID":"ses_cfg","part":{"type":"step-finish","tokens":{"input":1,"output":1}}}`,
			},
		}},
	})
	assert.Equal(t, 0, result.exitCode)
	raw, err := os.ReadFile(filepath.Join(result.taskDir, "opencode.json"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "https://hosted.example/openrouter")
	assert.Contains(t, result.spawns[0], "--format")
	assert.Contains(t, result.spawns[0], "json")
	assert.Contains(t, result.spawns[0], "--auto")
	assert.Equal(t, "--pure", result.spawns[0][0])
	assert.Equal(t, "run", result.spawns[0][1])
	requireLiveLogLine(t, result.output, "Starting OpenCode · openrouter/anthropic/claude-sonnet-4-6")
	requireLiveLogLine(t, result.output, "Calling OpenCode · openrouter/anthropic/claude-sonnet-4-6 (attempt 1 of 4)")
	requireNoBareStdoutLine(t, result.output, "Starting OpenCode")
}

func TestRunPromptDoesNotRetryWhenSuccessfulSpawnLogs429(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model: "x-ai/grok-4.6",
		spawns: []spawnScript{{
			ExitCode: 0,
			Stderr:   "HTTP 429 Too Many Requests",
			Stdout: []string{
				`{"type":"error","sessionID":"ses_ok","error":{"data":{"message":"Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."}}}`,
				`{"type":"text","sessionID":"ses_ok","part":{"type":"text","text":"recovered"}}`,
				`{"type":"step_finish","sessionID":"ses_ok","part":{"type":"step-finish","tokens":{"input":1,"output":1}}}`,
			},
		}},
	})
	assert.Equal(t, 0, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Empty(t, result.sleeps)
	assert.NotContains(t, result.output, "Waiting")
	assert.NotContains(t, result.output, "Retrying")
	payload := resultPayload(t, result.resultFile)
	assert.Equal(t, "recovered", payload["result"])
}

func TestRunPromptStopsWaitingWhenExecutionTimeoutExpires(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model:     "x-ai/grok-4.6",
		nowValues: []int64{0, 2000},
		env: map[string]string{
			"SUPERPLANE_EXECUTION_TIMEOUT_SECONDS": "1",
		},
		spawns: []spawnScript{
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."),
		},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Empty(t, result.sleeps)
	requireLiveLogLine(t, result.output, "Rate limit wait exceeded the execution timeout")
	requireNoBareStdoutLine(t, result.output, "wait exceeded the execution timeout")
}

func TestRunPromptTimeoutLineUsesTemporaryErrorLabel(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model:     "x-ai/grok-4.6",
		nowValues: []int64{0, 2000},
		env: map[string]string{
			"SUPERPLANE_EXECUTION_TIMEOUT_SECONDS": "1",
		},
		spawns: []spawnScript{{
			ExitCode: 1,
			Stderr:   "HTTP 503 Service Unavailable",
			Stdout: []string{
				`{"type":"error","error":{"name":"APIError","message":"Provider returned error","data":{"statusCode":503}}}`,
			},
		}},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Empty(t, result.sleeps)
	requireLiveLogLine(t, result.output, "Temporary error wait exceeded the execution timeout")
	assert.NotContains(t, result.output, "Rate limit wait exceeded the execution timeout")
}

func TestRunPromptDoesNotWaitWhenRetryExceedsRemainingTimeout(t *testing.T) {
	result := runOpenRouterPrompt(t, promptHarness{
		model:     "x-ai/grok-4.6",
		nowValues: []int64{0, 2000},
		env: map[string]string{
			"SUPERPLANE_EXECUTION_TIMEOUT_SECONDS": "10",
		},
		spawns: []spawnScript{
			rateLimitSpawn("Rate limit exceeded: new-account-rpm/x-ai/grok-4.6. Please retry shortly."),
		},
	})
	assert.Equal(t, 1, result.exitCode)
	require.Len(t, result.spawns, 1)
	assert.Empty(t, result.sleeps)
	requireLiveLogLine(t, result.output, "Rate limit wait exceeded the execution timeout")
}

func TestWaitDeadlineMsUsesExecutionTimeoutSeconds(t *testing.T) {
	deadline := jsWaitDeadline(t, map[string]string{"SUPERPLANE_EXECUTION_TIMEOUT_SECONDS": "30"}, 1000)
	assert.Equal(t, float64(31000), deadline)
}

func TestOpenCodeProcessEnvSetsPureAndIsolatesHomes(t *testing.T) {
	env := jsOpenCodeProcessEnv(t, "/task")
	assert.Equal(t, "1", env["OPENCODE_PURE"])
	assert.Equal(t, "1", env["OPENCODE_DISABLE_AUTOUPDATE"])
	assert.Equal(t, "1", env["OPENCODE_DISABLE_MODELS_FETCH"])
	assert.Equal(t, "1", env["OPENCODE_DISABLE_LSP_DOWNLOAD"])
	assert.Equal(t, "1", env["OPENCODE_DISABLE_CLAUDE_CODE"])
	assert.Equal(t, "/task/opencode.json", env["OPENCODE_CONFIG"])
	assert.Equal(t, "/task/xdg/data", env["XDG_DATA_HOME"])
}

type spawnScript struct {
	ExitCode     int      `json:"exitCode"`
	Stdout       []string `json:"stdout"`
	Stderr       string   `json:"stderr"`
	SessionParts []string `json:"sessionParts"`
}

type promptHarness struct {
	model        string
	spawns       []spawnScript
	env          map[string]string
	nowValues    []int64
	sessionUsage []map[string]any
}

type openRouterPromptResult struct {
	exitCode   int
	output     string
	resultFile string
	taskDir    string
	spawns     [][]string
	sleeps     []float64
}

func runOpenRouterPrompt(t *testing.T, harness promptHarness) openRouterPromptResult {
	t.Helper()
	dir := t.TempDir()
	writeTaskHelpers(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "prompt_count"), []byte("0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte("do the work"), 0o644))
	return runPromptInDir(t, dir, "prompt.txt", harness.model, harness.spawns, harness.env, harness.nowValues, harness.sessionUsage)
}

func rateLimitSpawn(message string) spawnScript {
	return spawnScript{
		ExitCode: 1,
		Stderr:   message,
		Stdout:   []string{fmt.Sprintf(`{"type":"error","error":{"data":{"message":%q}}}`, message)},
	}
}

func successSpawn(text string) spawnScript {
	return spawnScript{
		ExitCode: 0,
		Stdout: []string{
			fmt.Sprintf(`{"type":"text","sessionID":"ses_ok","part":{"type":"text","text":%q}}`, text),
			`{"type":"step_finish","sessionID":"ses_ok","part":{"type":"step-finish","tokens":{"input":1,"output":1}}}`,
		},
	}
}

func runPromptInDir(t *testing.T, dir, promptName, model string, spawns []spawnScript, extraEnv map[string]string, nowValues []int64, sessionUsage []map[string]any) openRouterPromptResult {
	t.Helper()
	resultFile := filepath.Join(dir, "result.json")
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	spawnsJSON, err := json.Marshal(spawns)
	require.NoError(t, err)
	if nowValues == nil {
		nowValues = []int64{}
	}
	nowJSON, err := json.Marshal(nowValues)
	require.NoError(t, err)
	sessionJSON, err := json.Marshal(sessionUsage)
	require.NoError(t, err)
	harnessFile := filepath.Join(dir, "harness.js")
	require.NoError(t, os.WriteFile(harnessFile, []byte(fmt.Sprintf(`
const fs = require("fs");
const { PassThrough } = require("stream");
const { runPrompt } = require(%q);
const spawns = %s;
const nowValues = %s;
const sessionReads = %s;
const calls = [];
const sleeps = [];
let index = 0;
let nowIndex = 0;
let sessionReadIndex = 0;
function mockChild(spec) {
  const stdout = new PassThrough();
  const stderr = new PassThrough();
  const listeners = {};
  const child = {
    stdout,
    stderr,
    on(event, cb) {
      listeners[event] = cb;
      return child;
    },
  };
  setImmediate(() => {
    for (const line of spec.stdout || []) {
      stdout.write(line + "\n");
    }
    stdout.end();
    if (spec.stderr) {
      stderr.write(spec.stderr);
    }
    stderr.end();
    if (listeners.close) {
      listeners.close(spec.exitCode == null ? 0 : spec.exitCode);
    }
  });
  return child;
}
const helpers = {
  spawnOpenCode(args) {
    const spec = spawns[index] || { exitCode: 1, stderr: "unexpected extra spawn", stdout: [] };
    index += 1;
    calls.push(args);
    if (Array.isArray(spec.sessionParts) && spec.sessionParts.length) {
      const partDir = require("path").join(process.env.SUPERPLANE_TASK_DIR, "xdg", "data", "opencode", "storage", "part", "msg_live");
      fs.mkdirSync(partDir, { recursive: true });
      spec.sessionParts.forEach((body, i) => {
        fs.writeFileSync(require("path").join(partDir, "prt_" + i + ".json"), body);
      });
    }
    return mockChild(spec);
  },
  sleep(ms) {
    sleeps.push(ms);
    return Promise.resolve();
  },
  now: nowValues.length
    ? () => nowValues[Math.min(nowIndex++, nowValues.length - 1)]
    : undefined,
  cwd: %q,
};
if (Array.isArray(sessionReads)) {
  helpers.readSessionUsage = () => {
    const row = sessionReadIndex < sessionReads.length ? sessionReads[sessionReadIndex] : {};
    sessionReadIndex += 1;
    return row;
  };
}
runPrompt(%q, %q, helpers)
  .then((code) => {
    fs.writeFileSync(process.env.SPAWNS_FILE, JSON.stringify({ calls, sleeps }));
    process.exit(code);
  })
  .catch((err) => {
    fs.writeFileSync(process.env.SPAWNS_FILE, JSON.stringify({ calls, sleeps }));
    console.error(err && err.message ? err.message : err);
    process.exit(1);
  });
`, script, spawnsJSON, nowJSON, sessionJSON, dir, filepath.Join(dir, promptName), model)), 0o644))

	spawnsFile := filepath.Join(dir, "spawns.json")
	cmd := exec.Command("node", harnessFile)
	cmd.Dir = dir
	env := append(os.Environ(),
		"SUPERPLANE_RESULT_FILE="+resultFile,
		"SUPERPLANE_TASK_DIR="+dir,
		"OPENROUTER_API_KEY=test",
		"SPAWNS_FILE="+spawnsFile,
	)
	if extraEnv != nil {
		for key, value := range extraEnv {
			env = append(env, key+"="+value)
		}
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		require.Truef(t, ok, "node harness failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		exitCode = exitErr.ExitCode()
	}
	recorded := struct {
		Calls  [][]string `json:"calls"`
		Sleeps []float64  `json:"sleeps"`
	}{}
	raw, readErr := os.ReadFile(spawnsFile)
	require.NoError(t, readErr)
	require.NoError(t, json.Unmarshal(raw, &recorded))
	return openRouterPromptResult{
		exitCode:   exitCode,
		output:     stdout.String(),
		resultFile: resultFile,
		taskDir:    dir,
		spawns:     recorded.Calls,
		sleeps:     recorded.Sleeps,
	}
}

func writeTaskHelpers(t *testing.T, dir string) {
	t.Helper()
	copyScript(t, dir, "../llm_usage.js", "llm_usage.js")
	copyScript(t, dir, "../turn_telemetry.js", "turn_telemetry.js")
}

func copyScript(t *testing.T, dir, src, dest string) {
	t.Helper()
	path, err := filepath.Abs(src)
	require.NoError(t, err)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, dest), body, 0o644))
}

func jsWaitDeadline(t *testing.T, env map[string]string, nowMs int64) float64 {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(map[string]any{"env": env, "now": nowMs})
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { waitDeadlineMs } = require(process.argv[1]); const input = JSON.parse(process.argv[2]); process.stdout.write(String(waitDeadlineMs(input.env, () => input.now)));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var deadline float64
	_, err = fmt.Sscan(string(out), &deadline)
	require.NoError(t, err)
	return deadline
}

func jsOpenCodeProcessEnv(t *testing.T, taskDir string) map[string]string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { openCodeProcessEnv } = require(process.argv[1]); process.stdout.write(JSON.stringify(openCodeProcessEnv(process.argv[2], {})));`, script, taskDir)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var env map[string]string
	require.NoError(t, json.Unmarshal(out, &env))
	return env
}

func jsString(t *testing.T, fn, arg string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const m = require(process.argv[1]); process.stdout.write(m[process.argv[2]](process.argv[3]));`, script, fn, arg)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func jsClassify(t *testing.T, text string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { classifyOpenRouterError } = require(process.argv[1]); process.stdout.write(classifyOpenRouterError(process.argv[2]));`, script, text)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func jsOpencodeArgs(t *testing.T, input map[string]any) []string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(input)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { opencodeRunArgs } = require(process.argv[1]); process.stdout.write(JSON.stringify(opencodeRunArgs(JSON.parse(process.argv[2]))));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var args []string
	require.NoError(t, json.Unmarshal(out, &args))
	return args
}

func jsBuildConfig(t *testing.T, taskDir string, env map[string]string) map[string]any {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(map[string]any{"taskDir": taskDir, "env": env, "planning": env["SUPERPLANE_PLANNING_SESSION_ID"] != ""})
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { buildOpenCodeConfig } = require(process.argv[1]); process.stdout.write(JSON.stringify(buildOpenCodeConfig(JSON.parse(process.argv[2]))));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var config map[string]any
	require.NoError(t, json.Unmarshal(out, &config))
	return config
}

func runOpenCodeFormatter(t *testing.T, lines []string) string {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(lines)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { formatOpenCodeJsonLines } = require(process.argv[1]); formatOpenCodeJsonLines(JSON.parse(process.argv[2]));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}

func formatOpenCodeJSONLinesFailed(t *testing.T, lines []string) bool {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	payload, err := json.Marshal(lines)
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { formatOpenCodeJsonLines } = require(process.argv[1]); process.stdout.write(JSON.stringify(formatOpenCodeJsonLines(JSON.parse(process.argv[2]))));`, script, string(payload))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	var result struct {
		Failed bool `json:"failed"`
	}
	require.NoError(t, json.Unmarshal(out, &result), string(out))
	return result.Failed
}

func runTurnResult(t *testing.T, event map[string]any) string {
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

func liveLogRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, rec := range typedLiveLogRecords(t, output) {
		if rec["type"] == "tool_start" || rec["type"] == "tool_end" {
			records = append(records, rec)
		}
	}
	return records
}

func typedLiveLogRecords(t *testing.T, output string) []map[string]any {
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
		if strings.TrimSpace(fmt.Sprint(rec["type"])) == "" {
			continue
		}
		records = append(records, rec)
	}
	return records
}

func liveLogLineTexts(t *testing.T, output string) []string {
	t.Helper()
	var texts []string
	for _, rec := range typedLiveLogRecords(t, output) {
		if rec["type"] != "line" {
			continue
		}
		text, ok := rec["text"].(string)
		if !ok || text == "" {
			continue
		}
		texts = append(texts, text)
	}
	return texts
}

func requireLiveLogLine(t *testing.T, output, needle string) {
	t.Helper()
	for _, text := range liveLogLineTexts(t, output) {
		if strings.Contains(text, needle) {
			return
		}
	}
	require.Failf(t, "missing typed live-log line", "wanted %q in type=line records; got %q", needle, liveLogLineTexts(t, output))
}

func requireNoBareStdoutLine(t *testing.T, output, needle string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "{") {
			continue
		}
		if strings.Contains(trimmed, needle) {
			require.Failf(t, "found bare stdout line", "wanted %q only as a type=line live-log record; found %q", needle, trimmed)
		}
	}
}

func resultPayload(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}

func assertResultUsage(t *testing.T, payload map[string]any, input, output, cost float64) {
	t.Helper()
	usage, ok := payload["usage"].(map[string]any)
	require.True(t, ok, "result payload must include usage")
	assert.Equal(t, input, usage["input_tokens"])
	assert.Equal(t, output, usage["output_tokens"])
	require.Contains(t, payload, "total_cost_usd")
	assert.InDelta(t, cost, payload["total_cost_usd"], 1e-9)
}

func jsReadSessionUsage(t *testing.T, taskDir string) map[string]any {
	t.Helper()
	script, err := filepath.Abs("run.js")
	require.NoError(t, err)
	cmd := exec.Command("node", "-e", `const { readSessionUsage } = require(process.argv[1]); process.stdout.write(JSON.stringify(readSessionUsage(process.argv[2])));`, script, taskDir)
	cmd.Env = append(os.Environ(), "NODE_NO_WARNINGS=1")
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		require.NoError(t, err, stderr)
	}
	var usage map[string]any
	require.NoError(t, json.Unmarshal(out, &usage), string(out))
	return usage
}
