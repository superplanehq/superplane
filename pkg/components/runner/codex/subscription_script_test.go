package codex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionScriptSavesSetupFailuresWithoutSecrets(t *testing.T) {
	const runnerToken = "subscription-runner-token-value"
	tests := []struct {
		name       string
		command    string
		status     int
		bundle     map[string]any
		brokenHome bool
		wantError  string
	}{
		{
			name:      "restore request rejected",
			command:   "restore",
			status:    http.StatusForbidden,
			wantError: "ChatGPT subscription request failed (403)",
		},
		{
			name:      "refresh credentials missing",
			command:   "refresh",
			status:    http.StatusOK,
			wantError: "ChatGPT subscription credentials are missing",
		},
		{
			name:       "restore filesystem error contains token",
			command:    "restore",
			status:     http.StatusOK,
			bundle:     map[string]any{"access_token": "access-token-value"},
			brokenHome: true,
			wantError:  "[REDACTED]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/runner/codex-subscription/credentials", r.URL.Path)
				w.WriteHeader(tt.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"bundle": tt.bundle})
			}))
			t.Cleanup(server.Close)
			taskDir := t.TempDir()
			resultFile := filepath.Join(taskDir, "result.json")
			codexHome := filepath.Join(taskDir, "codex-home")
			if tt.brokenHome {
				codexHome = filepath.Join(taskDir, runnerToken)
				require.NoError(t, os.WriteFile(codexHome, nil, 0o600))
			}
			_, callerFile, _, ok := runtime.Caller(0)
			require.True(t, ok)
			cmd := exec.Command("node", filepath.Join(filepath.Dir(callerFile), "codex_subscription.js"), tt.command)
			cmd.Env = append(os.Environ(),
				"SUPERPLANE_TASK_DIR="+taskDir,
				"SUPERPLANE_RESULT_FILE="+resultFile,
				"CODEX_HOME="+codexHome,
				"SUPERPLANE_BASE_URL="+server.URL,
				"SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN="+runnerToken,
			)
			out, err := cmd.CombinedOutput()
			require.Error(t, err)
			contents, err := os.ReadFile(resultFile)
			require.NoError(t, err, "setup failures must be saved for the task: %s", out)
			var result map[string]any
			require.NoError(t, json.Unmarshal(contents, &result))
			assert.Equal(t, "result", result["type"])
			assert.Equal(t, true, result["is_error"])
			assert.Contains(t, result["error"], "ChatGPT subscription "+tt.command+" failed")
			assert.Contains(t, result["error"], tt.wantError)
			assert.Contains(t, string(out), tt.wantError)
			assert.NotContains(t, string(out), runnerToken)
			assert.NotContains(t, string(contents), runnerToken)
		})
	}
}

func TestSubscriptionScriptReportsUsageWithoutAuthBundle(t *testing.T) {
	var body map[string]any
	var unavailable atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if unavailable.Load() {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		require.Equal(t, "/api/v1/runner/codex-subscription/checkpoint", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true, "version": 2})
	}))
	t.Cleanup(server.Close)
	taskDir := t.TempDir()
	binDir := filepath.Join(taskDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "codex"), []byte(`#!/usr/bin/env node
const readline = require("readline");
readline.createInterface({ input: process.stdin }).on("line", (line) => {
  const request = JSON.parse(line);
  if (request.method === "initialize") {
    process.stdout.write(JSON.stringify({ id: request.id, result: {} }) + "\n");
  }
  if (request.method === "account/rateLimits/read") {
    process.stdout.write(JSON.stringify({ id: request.id, result: { rateLimits: { limitId: "codex", primary: { usedPercent: 10, windowDurationMins: 300, resetsAt: 1900000000 } } } }) + "\n");
  }
  if (request.method === "model/list") {
    process.stdout.write(JSON.stringify({ id: request.id, result: { data: [{ id: "gpt-5.4", model: "gpt-5.4", displayName: "GPT-5.4", isDefault: true }] } }) + "\n");
  }
});
`), 0o755))
	_, callerFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	script := `
const { reportUsage } = require(process.env.CODEX_SUBSCRIPTION_MODULE);
const fs = require("fs");
fs.mkdirSync(process.env.CODEX_HOME, { recursive: true });
fs.writeFileSync(process.env.CODEX_HOME + "/auth.json", JSON.stringify({ access_token: "access-token-value", refresh_token: "refresh-token-value", id_token: "id-token-value" }));
reportUsage().catch((err) => { process.stderr.write(err.message + "\n"); process.exit(1); });
`
	cmd := exec.Command("node", "-e", script)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"CODEX_SUBSCRIPTION_MODULE="+filepath.Join(filepath.Dir(callerFile), "codex_subscription.js"),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"CODEX_HOME="+filepath.Join(taskDir, "codex-home"),
		"SUPERPLANE_BASE_URL="+server.URL,
		"SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN=runner-token",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	_, hasBundle := body["bundle"]
	assert.False(t, hasBundle)
	assert.NotContains(t, string(out), "refresh-token-value")
	usage, ok := body["usage"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, usage["limits"])
	unavailable.Store(true)
	cmd = exec.Command("node", filepath.Join(filepath.Dir(callerFile), "codex_subscription.js"), "usage")
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_BASE_URL="+server.URL,
		"SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN=runner-token",
	)
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "Codex usage could not be saved")
}

func TestSubscriptionScriptRestoresCheckpointsAndHidesTokens(t *testing.T) {
	refreshToken := "refresh-token-value"
	var mu sync.Mutex
	var checkpoints []map[string]any
	var progress map[string]any
	checkpointAttempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/runner/codex-subscription/credentials":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"version": 3,
				"bundle": map[string]any{
					"access_token":   "access-token-value",
					"refresh_token":  refreshToken,
					"id_token":       "id-token-value",
					"OPENAI_API_KEY": "sk-live",
				},
			})
		case "/api/v1/runner/codex-subscription/checkpoint":
			mu.Lock()
			checkpointAttempts++
			unavailable := checkpointAttempts == 1
			mu.Unlock()
			if unavailable {
				http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			mu.Lock()
			checkpoints = append(checkpoints, body)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true, "version": 4})
		case "/api/v1/runner/codex-subscription/progress":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			mu.Lock()
			progress = body
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	taskDir := t.TempDir()
	binDir := filepath.Join(taskDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	fakeCodex := filepath.Join(binDir, "codex")
	require.NoError(t, os.WriteFile(fakeCodex, []byte(`#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
if (process.argv[2] === "app-server") {
  require("readline").createInterface({ input: process.stdin }).on("line", (line) => {
    const request = JSON.parse(line);
    if (request.method === "initialize") {
      process.stdout.write(JSON.stringify({ id: request.id, result: {} }) + "\n");
    }
    if (request.method === "account/rateLimits/read") {
      const reply = fs.existsSync(path.join(process.env.CODEX_HOME, "unavailable"))
        ? { error: { message: "provider unavailable access-token-value" } }
        : { result: { rateLimits: { limitId: "codex", primary: { usedPercent: 25, windowDurationMins: 300, resetsAt: 1900000000 }, secondary: null } } };
      if (reply.result && fs.existsSync(path.join(process.env.CODEX_HOME, "multi"))) {
        reply.result.rateLimitsByLimitId = {
          codex: reply.result.rateLimits,
          codex_other: { limitId: "codex_other", primary: { usedPercent: 60, windowDurationMins: 60, resetsAt: 1900000000 } },
        };
      }
      if (fs.existsSync(path.join(process.env.CODEX_HOME, "empty"))) reply.result = {};
      process.stdout.write(JSON.stringify({ id: request.id, ...reply }) + "\n");
    }
    if (request.method === "model/list") {
      const page = request.params && request.params.cursor
        ? { data: [{ id: "gpt-5.3", model: "gpt-5.3", displayName: "GPT-5.3", defaultReasoningEffort: "medium", supportedReasoningEfforts: [], isDefault: false }] }
        : { data: [{ id: "gpt-5.4", model: "gpt-5.4", displayName: "GPT-5.4", defaultReasoningEffort: "medium", supportedReasoningEfforts: [{ reasoningEffort: "low", description: "Fast" }, { reasoningEffort: "xhigh", description: "Deep" }], isDefault: true }], nextCursor: "next" };
      process.stdout.write(JSON.stringify({ id: request.id, result: page }) + "\n");
    }
  });
} else {
  fs.mkdirSync(process.env.CODEX_HOME, { recursive: true });
  fs.writeFileSync(path.join(process.env.CODEX_HOME, "auth.json"), JSON.stringify({ access_token: "access-token-value", refresh_token: "refresh-token-value", id_token: "id-token-value", OPENAI_API_KEY: "sk-live" }));
  process.stdout.write("\u001b[94mhttps://auth.openai.com/codex/device\u001b[0m\nCODE-1234\n");
}
`), 0o755))

	_, callerFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	script := `
const { subscriptionChildEnv, redactSecrets, restore, checkpoint, login } = require(process.env.CODEX_SUBSCRIPTION_MODULE);
const fs = require("fs");
const child = subscriptionChildEnv(process.env);
if (child.OPENAI_API_KEY || child.OPENAI_BASE_URL) {
  throw new Error("API credentials remained in the Codex environment");
}
(async () => {
  await restore();
  if ((fs.statSync(child.CODEX_HOME).mode & 0o777) !== 0o700 || (fs.statSync(child.CODEX_HOME + "/auth.json").mode & 0o777) !== 0o600) throw new Error("restore permissions are unsafe");
  const auth = fs.readFileSync(process.env.SUPERPLANE_TASK_DIR + "/codex-home/auth.json", "utf8");
  const redacted = redactSecrets("token " + JSON.parse(auth).refresh_token);
  if (redacted.includes(JSON.parse(auth).refresh_token)) {
    throw new Error("refresh token remained in the log");
  }
  await checkpoint({ final: true, version: 4 });
  fs.writeFileSync(child.CODEX_HOME + "/unavailable", "");
  await checkpoint();
  fs.unlinkSync(child.CODEX_HOME + "/unavailable");
  fs.writeFileSync(child.CODEX_HOME + "/empty", "");
  await checkpoint();
  fs.unlinkSync(child.CODEX_HOME + "/empty");
  fs.writeFileSync(child.CODEX_HOME + "/multi", "");
  await login();
  process.stdout.write(redacted + "\n");
})().catch((err) => {
  process.stderr.write(err.message + "\n");
  process.exit(1);
});
`
	cmd := exec.Command("node", "-e", script)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"CODEX_SUBSCRIPTION_MODULE="+filepath.Join(filepath.Dir(callerFile), "codex_subscription.js"),
		"SUPERPLANE_TASK_DIR="+taskDir,
		"SUPERPLANE_BASE_URL="+server.URL,
		"SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN=runner-token",
		"OPENAI_API_KEY=sk-secret",
		"SUPERPLANE_RESULT_FILE="+filepath.Join(taskDir, "result.json"),
		"OPENAI_BASE_URL=https://api.openai.com/v1",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	assert.NotContains(t, string(out), refreshToken)
	assert.Contains(t, string(out), "[redacted]")
	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, checkpoints)
	last := checkpoints[len(checkpoints)-1]
	bundle, _ := last["bundle"].(map[string]any)
	assert.Equal(t, refreshToken, bundle["refresh_token"])
	assert.Equal(t, true, last["final"])
	usage, ok := last["usage"].(map[string]any)
	require.True(t, ok, "checkpoint must report the account quota")
	limits := usage["limits"].([]any)
	require.Len(t, limits, 2, "all account quota buckets must be reported")
	models := usage["models"].([]any)
	require.Len(t, models, 2, "the complete account model catalog must be reported")
	assert.Equal(t, "gpt-5.4", models[0].(map[string]any)["model"])
	primary := limits[0].(map[string]any)["primary"].(map[string]any)
	assert.Equal(t, float64(25), primary["usedPercent"])
	assert.Equal(t, float64(300), primary["windowDurationMins"])
	for _, checkpoint := range checkpoints[1 : len(checkpoints)-1] {
		usage, ok := checkpoint["usage"].(map[string]any)
		require.True(t, ok, "an unavailable quota must not discard the catalog")
		assert.Empty(t, usage["limits"])
		assert.Len(t, usage["models"], 2)
	}
	assert.NotContains(t, string(out), "provider unavailable access-token-value")
	assert.Equal(t, "https://auth.openai.com/codex/device", progress["verificationUrl"])
	assert.Equal(t, "CODE-1234", progress["verificationCode"])

	_, err = os.Stat(filepath.Join(taskDir, "codex-home"))
	require.True(t, os.IsNotExist(err), "successful login must remove temporary credentials")
}

func TestSubscriptionSetupCommandsConfirmCleanupBeforeSuccess(t *testing.T) {
	for _, command := range []string{"login", "refresh"} {
		for _, failure := range []string{"", "removal", "checkpoint", "result", "shutdown"} {
			t.Run(command+"/"+failure, func(t *testing.T) {
				taskDir := t.TempDir()
				home := filepath.Join(taskDir, "codex-home")
				resultFile := filepath.Join(taskDir, "result.json")
				require.NoError(t, os.WriteFile(resultFile, []byte(`{"unrelated":"preserved","codexSubscriptionSetupCleaned":true}`), 0o600))
				binDir := filepath.Join(taskDir, "bin")
				require.NoError(t, os.Mkdir(binDir, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(binDir, "codex"), []byte(`#!/usr/bin/env node
const fs = require("fs");
const bundle = { access_token: "access-secret-value", refresh_token: "refresh-secret-value", id_token: "id-secret-value" };
if (process.argv[2] === "login") {
  fs.writeFileSync(process.env.CODEX_HOME + "/auth.json", JSON.stringify(bundle));
  process.stdout.write("https://auth.openai.com/codex/device\nCODE-1234\n");
} else {
  fs.writeFileSync(process.env.SUPERPLANE_TASK_DIR + "/child.pid", String(process.pid));
  require("readline").createInterface({ input: process.stdin }).on("line", line => {
    const request = JSON.parse(line);
    if (!request.id) return;
    const result = request.method === "model/list" ? { data: [{ model: "gpt-5.4" }] } : {};
    process.stdout.write(JSON.stringify({ id: request.id, result }) + "\n");
  });
}
`), 0o700))
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/v1/runner/codex-subscription/credentials" {
						_ = json.NewEncoder(w).Encode(map[string]any{"bundle": map[string]any{"refresh_token": "refresh-secret-value"}})
						return
					}
					if r.URL.Path == "/api/v1/runner/codex-subscription/checkpoint" {
						pid, err := os.ReadFile(filepath.Join(taskDir, "child.pid"))
						require.NoError(t, err)
						_, err = os.Stat("/proc/" + string(pid))
						assert.True(t, os.IsNotExist(err), "Codex must close before the final checkpoint")
						if failure == "checkpoint" {
							http.Error(w, "rejected", http.StatusConflict)
							return
						}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true, "version": 1})
				}))
				t.Cleanup(server.Close)
				_, callerFile, _, ok := runtime.Caller(0)
				require.True(t, ok)
				// Preload a filesystem fault at the real CLI boundary. Permissions are unreliable as root.
				preload := filepath.Join(taskDir, "fault.js")
				require.NoError(t, os.WriteFile(preload, []byte(`const fs = require("fs");
const write = fs.writeFileSync;
fs.writeFileSync = function(file, contents, ...args) {
  if (file === process.env.SUPERPLANE_RESULT_FILE && JSON.parse(contents).codexSubscriptionSetupCleaned === true && fs.existsSync(process.env.CODEX_HOME)) throw new Error("credentials still present");
  if (file === process.env.SUPERPLANE_RESULT_FILE && JSON.parse(contents).codexSubscriptionSetupCleaned === true && process.env.FAIL_RESULT === "true") throw new Error("cannot save refresh-secret-value");
  return write.call(this, file, contents, ...args);
};
if (process.env.FAIL_REMOVAL === "true") fs.rmSync = () => { throw new Error("cannot remove refresh-secret-value"); };
if (process.env.FAIL_SHUTDOWN === "true") require("child_process").ChildProcess.prototype.kill = () => { throw new Error("cannot stop refresh-secret-value"); };
`), 0o600))
				cmd := exec.Command("node", "--require", preload, filepath.Join(filepath.Dir(callerFile), "codex_subscription.js"), command)
				cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"), "SUPERPLANE_TASK_DIR="+taskDir, "CODEX_HOME="+home,
					"SUPERPLANE_RESULT_FILE="+resultFile, "SUPERPLANE_BASE_URL="+server.URL, "SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN=runner-token", fmt.Sprint("FAIL_REMOVAL=", failure == "removal"), fmt.Sprint("FAIL_SHUTDOWN=", failure == "shutdown"), fmt.Sprint("FAIL_RESULT=", failure == "result"))
				out, err := cmd.CombinedOutput()
				assert.NotContains(t, string(out), "refresh-secret-value")
				contents, readErr := os.ReadFile(resultFile)
				require.NoError(t, readErr)
				var result map[string]any
				require.NoError(t, json.Unmarshal(contents, &result))
				if failure != "" {
					require.Error(t, err, string(out))
					assert.NotEqual(t, true, result["codexSubscriptionSetupCleaned"])
					assert.Equal(t, true, result["is_error"])
					assert.NotContains(t, string(contents), "refresh-secret-value")
					return
				}
				require.NoError(t, err, string(out))
				assert.Equal(t, true, result["codexSubscriptionSetupCleaned"])
				assert.Equal(t, "preserved", result["unrelated"])
				_, err = os.Stat(home)
				assert.True(t, os.IsNotExist(err))
			})
		}
	}
}
