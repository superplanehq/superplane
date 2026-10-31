package codex

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexRunCancellationStopsLauncherAndWorker(t *testing.T) {
	for _, signal := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		t.Run(signal.String(), func(t *testing.T) {
			taskDir := t.TempDir()
			prompt := filepath.Join(taskDir, "prompt.txt")
			require.NoError(t, os.WriteFile(prompt, []byte("Inspect the change."), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(taskDir, "prompt_count"), []byte("0\n"), 0o600))
			subscription, err := os.ReadFile("codex_subscription.js")
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(taskDir, "codex_subscription.js"), subscription, 0o600))
			home := filepath.Join(taskDir, "codex-home")
			require.NoError(t, os.Mkdir(home, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{"tokens":{"access_token":"subscription-credential"}}`), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(taskDir, "codex"), []byte(`#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const worker = process.argv[2] === "worker";
for (const signal of ["SIGTERM", "SIGINT", "SIGHUP"]) process.on(signal, () => {});
setInterval(() => {}, 1000);
const pidPath = path.join(process.env.SUPERPLANE_TASK_DIR, worker ? "worker.pid" : "launcher.pid");
fs.writeFileSync(pidPath + ".next", String(process.pid));
fs.renameSync(pidPath + ".next", pidPath);
if (!worker) {
 const id="thread-cancel", dir=path.join(process.env.CODEX_HOME,"sessions");
 fs.mkdirSync(dir,{recursive:true});
 const records=[
  {type:"session_meta",payload:{id,session_id:id}},
  {type:"event_msg",payload:{type:"task_started",turn_id:"turn-cancel"}},
  {type:"token_usage_record",payload:{thread_id:id,session_id:id,turn_id:"turn-cancel",response_id:"response-cancel",usage:{input_tokens:34,cached_input_tokens:7,output_tokens:8,reasoning_output_tokens:3,total_tokens:42}}}
 ];
 fs.writeFileSync(path.join(dir,"rollout-test-"+id+".jsonl"),records.map(r=>JSON.stringify(r)).join("\n")+"\n");
 process.stdout.write(JSON.stringify({type:"thread.started",thread_id:id})+"\n");
 require("child_process").spawn(process.execPath, [__filename, "worker"], {stdio: "inherit"});
}
`), 0o700))

			script, err := filepath.Abs("run.js")
			require.NoError(t, err)
			cmd := exec.Command("node", script, prompt)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Env = append(withoutCodexAPIKeys(os.Environ()),
				"PATH="+taskDir+":"+os.Getenv("PATH"),
				"SUPERPLANE_TASK_DIR="+taskDir,
				"CODEX_HOME="+home,
				"SUPERPLANE_RESULT_FILE="+filepath.Join(taskDir, "result.json"),
				"SUPERPLANE_CODEX_SUBSCRIPTION_TOKEN=subscription-token",
			)
			require.NoError(t, cmd.Start())
			waited := false
			pids := []int{}
			t.Cleanup(func() {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				for _, pid := range pids {
					_ = syscall.Kill(-pid, syscall.SIGKILL)
					_ = syscall.Kill(pid, syscall.SIGKILL)
				}
				if !waited {
					_ = cmd.Wait()
				}
			})
			for _, file := range []string{"launcher.pid", "worker.pid"} {
				pidPath := filepath.Join(taskDir, file)
				require.Eventually(t, func() bool {
					_, err := os.Stat(pidPath)
					return err == nil
				}, 5*time.Second, 10*time.Millisecond, "%s must start before cancellation", file)
				encoded, err := os.ReadFile(pidPath)
				require.NoError(t, err)
				pid, err := strconv.Atoi(string(encoded))
				require.NoError(t, err)
				pids = append(pids, pid)
			}

			// Usage must reach the result while the launcher and worker remain
			// alive. A final-only reader cannot satisfy this boundary.
			resultPath := filepath.Join(taskDir, "result.json")
			require.Eventually(t, func() bool {
				encoded, err := os.ReadFile(resultPath)
				if err != nil {
					return false
				}
				var result struct {
					Usage struct {
						Total int `json:"total_tokens"`
					} `json:"usage"`
				}
				return json.Unmarshal(encoded, &result) == nil && result.Usage.Total == 42
			}, 5*time.Second, 10*time.Millisecond)

			// Host cancellation and timeouts signal the task group, then force
			// termination after 100 ms. Codex must stop within this boundary.
			require.NoError(t, syscall.Kill(-cmd.Process.Pid, signal))
			time.Sleep(100 * time.Millisecond)
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			err = cmd.Wait()
			waited = true
			require.Error(t, err)
			encoded, err := os.ReadFile(resultPath)
			require.NoError(t, err)
			var result struct {
				UsageState string `json:"usage_state"`
				Usage      struct {
					Total int `json:"total_tokens"`
					Input int `json:"input_tokens"`
				} `json:"usage"`
			}
			require.NoError(t, json.Unmarshal(encoded, &result))
			assert.Equal(t, "partial", result.UsageState)
			assert.Equal(t, 42, result.Usage.Total)
			assert.Equal(t, 27, result.Usage.Input)
			for _, pid := range pids {
				assert.EventuallyWithT(t, func(collect *assert.CollectT) {
					state, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
					if os.IsNotExist(err) {
						return
					}
					if assert.NoError(collect, err) {
						assert.Equal(collect, "Z", strings.Fields(string(state))[2], "Codex process %d survived cancellation", pid)
					}
				}, 5*time.Second, 10*time.Millisecond)
			}
		})
	}
}
