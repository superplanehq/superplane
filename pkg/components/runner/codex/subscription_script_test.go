package codex

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

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
  if ((fs.statSync(child.CODEX_HOME).mode & 0o777) !== 0o700 ||
      (fs.statSync(child.CODEX_HOME + "/auth.json").mode & 0o777) !== 0o600) {
    throw new Error("restored credentials have unsafe permissions");
  }
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
	assert.True(t, os.IsNotExist(err))
}

func TestSubscriptionSetupCommandsCleanBeforeSuccess(t *testing.T) {
	for _, command := range []string{"login", "refresh"} {
		for _, failure := range []string{"", "checkpoint", "cleanup"} {
			t.Run(command+"/"+failure, func(t *testing.T) {
				taskDir := t.TempDir()
				home := filepath.Join(taskDir, "codex-home")
				resultFile := filepath.Join(taskDir, "result.json")
				bin := filepath.Join(taskDir, "bin")
				require.NoError(t, os.Mkdir(bin, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(bin, "codex"), []byte(`#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
fs.mkdirSync(process.env.CODEX_HOME, {recursive:true});
fs.writeFileSync(path.join(process.env.CODEX_HOME, "auth.json"), JSON.stringify({access_token:"final-access-token"}));
if (process.argv[2] === "app-server") {
 fs.writeFileSync(path.join(process.env.SUPERPLANE_TASK_DIR, "usage.pid"), String(process.pid));
 // Model the npm launcher, whose native child can outlive the launcher.
 const writer = require("child_process").spawn(process.execPath, ["-e", "setInterval(() => {}, 1000)"], {stdio:"ignore"});
 fs.writeFileSync(path.join(process.env.SUPERPLANE_TASK_DIR, "writer.pid"), String(writer.pid));
 require("readline").createInterface({input:process.stdin}).on("line", line => {
  const req = JSON.parse(line);
  if (req.id) process.stdout.write(JSON.stringify({id:req.id, result: req.id === 3 ? {data:[]} : {}})+"\n");
 });
}
`), 0o755))
				t.Cleanup(func() {
					contents, err := os.ReadFile(filepath.Join(taskDir, "writer.pid"))
					if err == nil {
						pid, _ := strconv.Atoi(string(contents))
						if pid > 0 {
							_ = syscall.Kill(pid, syscall.SIGKILL)
						}
					}
				})
				checkpointed := false
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasSuffix(r.URL.Path, "/credentials") {
						_ = json.NewEncoder(w).Encode(map[string]any{"bundle": map[string]any{"access_token": "restored-access-token"}})
						return
					}
					require.True(t, strings.HasSuffix(r.URL.Path, "/checkpoint"))
					pidBytes, err := os.ReadFile(filepath.Join(taskDir, "usage.pid"))
					require.NoError(t, err)
					pid, err := strconv.Atoi(string(pidBytes))
					require.NoError(t, err)
					assert.ErrorIs(t, syscall.Kill(pid, 0), syscall.ESRCH, "Codex must stop before the final checkpoint")
					writerBytes, err := os.ReadFile(filepath.Join(taskDir, "writer.pid"))
					require.NoError(t, err)
					writer, err := strconv.Atoi(string(writerBytes))
					require.NoError(t, err)
					// The native child can finish exiting after the launcher closes.
					assert.EventuallyWithT(t, func(collect *assert.CollectT) {
						state, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(writer), "stat"))
						if os.IsNotExist(err) {
							return
						}
						if assert.NoError(collect, err) {
							assert.Equal(collect, "Z", strings.Fields(string(state))[2])
						}
					}, 5*time.Second, 10*time.Millisecond, "the native Codex child must stop after its process group is killed")
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					assert.Equal(t, true, body["final"])
					assert.Equal(t, "final-access-token", body["bundle"].(map[string]any)["access_token"])
					checkpointed = true
					_ = json.NewEncoder(w).Encode(map[string]any{"accepted": failure != "checkpoint", "version": 2})
				}))
				t.Cleanup(server.Close)
				_, caller, _, ok := runtime.Caller(0)
				require.True(t, ok)
				cmd := exec.Command("node", filepath.Join(filepath.Dir(caller), "codex_subscription.js"), command)
				cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "SUPERPLANE_TASK_DIR="+taskDir, "CODEX_HOME="+home, "SUPERPLANE_RESULT_FILE="+resultFile, "SUPERPLANE_BASE_URL="+server.URL, "SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN=runner-token")
				if failure == "cleanup" {
					preload := filepath.Join(taskDir, "fail-cleanup.js")
					require.NoError(t, os.WriteFile(preload, []byte(`require("fs").rmSync = () => { throw new Error("cleanup denied"); };`), 0o600))
					cmd.Env = append(cmd.Env, "NODE_OPTIONS=--require="+preload)
				}
				stdout, err := cmd.StdoutPipe()
				require.NoError(t, err)
				require.NoError(t, cmd.Start())
				// The success line is observable before process exit: credentials must
				// already be absent when a runner sees it.
				var output string
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					output += scanner.Text()
					if failure == "" {
						_, statErr := os.Stat(home)
						assert.True(t, os.IsNotExist(statErr), "temporary credentials must be removed before reporting success")
					}
				}
				require.NoError(t, scanner.Err())
				err = cmd.Wait()
				require.True(t, checkpointed)
				if failure == "" {
					require.NoError(t, err)
					assert.NotEmpty(t, output)
					_, statErr := os.Stat(home)
					assert.True(t, os.IsNotExist(statErr))
				} else {
					require.Error(t, err)
					assert.Empty(t, output)
				}
				contents, err := os.ReadFile(resultFile)
				require.NoError(t, err)
				var result map[string]any
				require.NoError(t, json.Unmarshal(contents, &result))
				if failure == "" {
					assert.Equal(t, true, result["codexSubscriptionSetupCleaned"])
				} else {
					assert.NotEqual(t, true, result["codexSubscriptionSetupCleaned"])
					assert.Equal(t, true, result["is_error"])
				}
			})
		}
	}
}
