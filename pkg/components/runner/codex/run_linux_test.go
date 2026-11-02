package codex

import (
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
if (!worker) require("child_process").spawn(process.execPath, [__filename, "worker"], {stdio: "inherit"});
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

			// Host cancellation and timeouts signal the task group, then force
			// termination after 100 ms. Codex must stop within this boundary.
			require.NoError(t, syscall.Kill(-cmd.Process.Pid, signal))
			time.Sleep(100 * time.Millisecond)
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			err = cmd.Wait()
			waited = true
			require.Error(t, err)
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
