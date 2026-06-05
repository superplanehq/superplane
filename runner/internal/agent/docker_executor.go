package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

// DockerExecutor runs a task inside a Docker container using a
// pull -> run -d --name -> exec -> stop+rm lifecycle. Cleanup runs in a
// deferred function so cancelled / panicked tasks do not leak containers.
//
// Multi-line `commands` are bundled into a single `sh -c` invocation with
// `set -e` so env / cwd persist across directives and the script fails fast
// on the first non-zero exit code (POSIX sh chosen over bash so minimal
// images like alpine work out of the box; see dockerExecTask). Argv
// (`command`) is dispatched via `docker exec <name> <argv...>`.
//
// `docker exec` is invoked without `-t`, so the task runs in a non-TTY
// context. CLI tools that probe `isatty()` (color output, progress bars,
// interactive prompts) will see stdout/stderr as a pipe rather than a
// terminal. This is intentional and matches what callers get from `docker
// run` without `-t`; it is also more predictable for batch / CI workloads.
type DockerExecutor struct {
	MaxOutputBytes int
	RunnerID       string
}

const (
	dockerNamePrefix        = "superplane-task-"
	dockerStopGraceSeconds  = 5
	dockerResultMountTarget = "/mnt/superplane-result.json"
	// dockerIdleEntrypoint keeps the container alive while we run `docker exec`
	// against it. `sleep infinity` works on every common base image (alpine,
	// debian, ubuntu, python:*, node:*). Images without `sleep` are not supported.
	dockerIdleEntrypoint = "sleep"
	dockerIdleArg        = "infinity"
)

// dockerNameUnsafe matches any byte that is NOT in Docker's allowed
// container-name alphabet ([a-zA-Z0-9_.-]); such bytes are replaced with '_'.
var dockerNameUnsafe = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

// containerName builds the per-task container name. Including the
// runner_id keeps multiple runners on the same host from stomping on
// each other's containers (own and orphan-sweep).
func containerName(runnerID, taskID string) string {
	return dockerNamePrefix + sanitizeDockerName(runnerID) + "-" + sanitizeDockerName(taskID)
}

func sanitizeDockerName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "x"
	}
	return dockerNameUnsafe.ReplaceAllString(s, "_")
}

func (d *DockerExecutor) Execute(ctx context.Context, task *api.TaskPayload, live io.Writer, resultHostPath string) (int, string, error) {
	if strings.TrimSpace(d.RunnerID) == "" {
		return 1, "", errors.New("runner_id required for docker execution")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return 1, "", fmt.Errorf("docker not available on this runner: %w", err)
	}
	image := strings.TrimSpace(task.DockerImage)
	if image == "" {
		return 1, "", errors.New("docker_image required")
	}
	max := d.MaxOutputBytes
	if max <= 0 {
		max = 512 * 1024
	}
	name := containerName(d.RunnerID, task.ID)

	out := &capWriter{max: max}

	// Cleanup runs even when ctx is cancelled mid-task or when Phase 2 fails
	// after the container was created. Using context.Background gives the
	// cleanup commands time to complete after the task ctx has expired.
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stopAndRemoveContainer(cleanupCtx, name)
	}()

	pullOut, err := captureDocker(ctx, "pull", image)
	if err != nil {
		out.WriteString(string(pullOut))
		if live != nil {
			_, _ = live.Write(pullOut)
		}
		return 1, out.String(), fmt.Errorf("docker pull %s: %w", image, err)
	}
	if live != nil && len(pullOut) > 0 {
		_, _ = live.Write(pullOut)
	}

	var jsWorkDir string
	if api.RunModeForTask(task) == models.RunModeJavaScript {
		var prepErr error
		jsWorkDir, prepErr = prepareDockerJavaScriptWorkDir(task)
		if prepErr != nil {
			return 1, "", prepErr
		}
		defer os.RemoveAll(jsWorkDir)
	}

	runArgs := []string{"run", "-d", "--name", name}
	if jsWorkDir != "" {
		runArgs = append(runArgs, "-v", jsWorkDir+":"+dockerJavaScriptWorkMount+":ro")
	}
	if rp := strings.TrimSpace(resultHostPath); rp != "" {
		f, ferr := os.OpenFile(rp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if ferr != nil {
			return 1, "", fmt.Errorf("result file: %w", ferr)
		}
		_ = f.Close()
		runArgs = append(runArgs,
			"-e", envSuperplaneResultFile+"="+dockerResultMountTarget,
			"-v", rp+":"+dockerResultMountTarget,
		)
	}
	runArgs = append(runArgs, "--entrypoint", dockerIdleEntrypoint, image, dockerIdleArg)
	runOut, err := captureDocker(ctx, runArgs...)
	if err != nil {
		out.WriteString(string(runOut))
		if live != nil {
			_, _ = live.Write(runOut)
		}
		return 1, out.String(), fmt.Errorf("docker run -d: %w", err)
	}
	if live != nil && len(runOut) > 0 {
		_, _ = live.Write(runOut)
	}

	singleCommandText, hasSingleCommand := dockerSingleCommandText(task)
	startedAt := time.Now()
	if hasSingleCommand {
		writeLiveLogCommandStart(live, 0, singleCommandText, startedAt)
	} else if api.RunModeForTask(task) == models.RunModeJavaScript {
		writeLiveLogCommandStart(live, 0, "node "+javaScriptProgramName, startedAt)
		hasSingleCommand = true
	}
	exitCode, execOut, runErr := dockerExecTask(ctx, name, task, live)
	if hasSingleCommand {
		writeLiveLogCommandEnd(live, 0, exitCode, time.Since(startedAt))
	}
	out.WriteString(stripLiveLogControlLines(execOut))
	return exitCode, out.String(), runErr
}

// dockerExecTask runs the task inside an existing container via `docker exec`.
// For `commands`, every directive is bundled into one `sh -c` script with
// `set -e` so env / cwd persist between lines and the script exits at the
// first failure. POSIX `sh` is used (rather than bash) so minimal images like
// alpine work without installing extra packages; `set -o pipefail` is omitted
// because dash (Debian / Ubuntu `/bin/sh`) does not support it. For argv
// `command`, the program is invoked directly.
//
// When live is non-nil, stdout/stderr bytes are tee'd to it in addition to
// being captured for the returned `output` string (internal only; not sent on complete).
func dockerExecTask(ctx context.Context, name string, task *api.TaskPayload, live io.Writer) (int, string, error) {
	args, err := dockerExecArgs(name, task)
	if err != nil {
		return 1, "", err
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	var buf bytes.Buffer
	if live != nil {
		mw := io.MultiWriter(&buf, live)
		cmd.Stdout = mw
		cmd.Stderr = mw
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}
	err = cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode(), buf.String(), err
		}
		return 1, buf.String(), err
	}
	return 0, buf.String(), nil
}

func dockerExecArgs(name string, task *api.TaskPayload) ([]string, error) {
	envArgs, err := dockerExecEnvironmentArgs(task.Environment)
	if err != nil {
		return nil, err
	}
	var args []string
	switch api.RunModeForTask(task) {
	case models.RunModeJavaScript:
		args := append([]string{"exec"}, envArgs...)
		return append(args, name, "node", dockerJavaScriptProgramPath()), nil
	}
	switch {
	case len(task.Commands) > 0:
		directives := normalizeDirectiveLines(task.Commands)
		if len(directives) == 0 {
			return nil, errEmptyCommands()
		}
		script := dockerCommandsScript(directives)
		args = append([]string{"exec"}, envArgs...)
		args = append(args, name, "sh", "-c", script)
	case len(task.Command) > 0:
		args = append([]string{"exec"}, envArgs...)
		args = append(args, name)
		args = append(args, task.Command...)
	default:
		return nil, errors.New("empty command")
	}
	return args, nil
}

func dockerSingleCommandText(task *api.TaskPayload) (string, bool) {
	if len(task.Commands) > 0 || len(task.Command) == 0 {
		return "", false
	}
	return strings.TrimSpace(strings.Join(task.Command, " ")), true
}

func dockerCommandsScript(directives []string) string {
	var script strings.Builder
	script.WriteString("set -e\n")
	script.WriteString("sp_now_ms() {\n")
	script.WriteString("  __sp_now=\"$(date +%s%3N 2>/dev/null || true)\"\n")
	script.WriteString("  case \"$__sp_now\" in\n")
	script.WriteString("    ''|*[!0-9]*) __sp_now=\"$(date +%s)000\" ;;\n")
	script.WriteString("  esac\n")
	script.WriteString("  printf '%s\\n' \"$__sp_now\"\n")
	script.WriteString("}\n")

	for i, directive := range directives {
		textJSON, _ := json.Marshal(directive)
		script.WriteString("__sp_cmd_start=\"$(sp_now_ms)\"\n")
		script.WriteString(`printf '{"type":"cmd_start","index":`)
		script.WriteString(strconv.Itoa(i))
		script.WriteString(`,"text":`)
		script.WriteString(string(textJSON))
		script.WriteString(`,"started_at":%s}\n' "$__sp_cmd_start"`)
		script.WriteString("\n")
		script.WriteString("if {\n")
		script.WriteString(directive)
		script.WriteString("\n}; then\n")
		script.WriteString("  __sp_cmd_exit=0\n")
		script.WriteString("else\n")
		script.WriteString("  __sp_cmd_exit=$?\n")
		script.WriteString("fi\n")
		script.WriteString("__sp_cmd_end=\"$(sp_now_ms)\"\n")
		script.WriteString("__sp_cmd_duration=$((__sp_cmd_end - __sp_cmd_start))\n")
		script.WriteString("if [ \"$__sp_cmd_duration\" -lt 0 ]; then __sp_cmd_duration=0; fi\n")
		script.WriteString("if [ \"$__sp_cmd_exit\" -eq 0 ]; then __sp_cmd_status=passed; else __sp_cmd_status=failed; fi\n")
		script.WriteString(`printf '{"type":"cmd_end","index":`)
		script.WriteString(strconv.Itoa(i))
		script.WriteString(`,"status":"%s","duration_ms":%s}\n' "$__sp_cmd_status" "$__sp_cmd_duration"` + "\n")
		script.WriteString("if [ \"$__sp_cmd_exit\" -ne 0 ]; then exit \"$__sp_cmd_exit\"; fi\n")
	}

	return script.String()
}

func stripLiveLogControlLines(output string) string {
	if output == "" {
		return output
	}
	parts := strings.SplitAfter(output, "\n")
	var cleaned strings.Builder
	for _, part := range parts {
		line := strings.TrimSuffix(part, "\n")
		if isLiveLogControlLine(line) {
			continue
		}
		cleaned.WriteString(part)
	}
	return cleaned.String()
}

func isLiveLogControlLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(line), &envelope); err != nil {
		return false
	}
	switch envelope.Type {
	case "cmd_start":
		var rec liveLogCommandStartRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return false
		}
		return rec.Index >= 0
	case "cmd_end":
		var rec liveLogCommandEndRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			return false
		}
		return rec.Index >= 0 && rec.DurationMS >= 0 && (rec.Status == liveLogCommandPassed || rec.Status == liveLogCommandFailed)
	default:
		return false
	}
}

func captureDocker(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

// stopAndRemoveContainer is the cleanup path. Both calls are best-effort:
// if the container was never created `stop` returns non-zero and we proceed;
// `rm -f` removes it whether it is running or stopped.
func stopAndRemoveContainer(ctx context.Context, name string) {
	_ = exec.CommandContext(ctx, "docker",
		"stop", "--time", fmt.Sprintf("%d", dockerStopGraceSeconds), name).Run()
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
}

// SweepDockerOrphans removes containers left over from previous runs of this
// runner (e.g. after a crash or SIGKILL where the per-task defer cleanup did
// not get a chance to run). It is a no-op when docker is not on PATH or the
// daemon is unreachable.
func SweepDockerOrphans(ctx context.Context, runnerID string) (removed int, err error) {
	if strings.TrimSpace(runnerID) == "" {
		// Avoid sanitizeDockerName("") -> "x", which would make unrelated processes
		// share the same name prefix and orphan-sweep each other's containers.
		return 0, nil
	}
	if _, lookErr := exec.LookPath("docker"); lookErr != nil {
		return 0, nil
	}
	// name=^/… anchors to container names from the engine root; requires a
	// reasonably current Docker/Moby CLI (same era as docker compose v2).
	filter := "name=^/" + dockerNamePrefix + sanitizeDockerName(runnerID) + "-"
	listOut, lerr := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", filter).Output()
	if lerr != nil {
		return 0, lerr
	}
	for _, id := range strings.Fields(string(listOut)) {
		if rerr := exec.CommandContext(ctx, "docker", "rm", "-f", id).Run(); rerr == nil {
			removed++
		}
	}
	return removed, nil
}

// capWriter accumulates string output up to max bytes, then appends a
// truncation marker once and ignores further writes. Used to keep task
// output under MaxOutputBytes without scanning the buffer on every append.
type capWriter struct {
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (w *capWriter) WriteString(s string) {
	if w.max <= 0 {
		w.buf.WriteString(s)
		return
	}
	if w.truncated {
		return
	}
	room := w.max - w.buf.Len()
	if room <= 0 {
		w.truncated = true
		w.buf.WriteString("\n…(truncated)")
		return
	}
	if len(s) <= room {
		w.buf.WriteString(s)
		return
	}
	w.buf.WriteString(s[:room])
	w.buf.WriteString("\n…(truncated)")
	w.truncated = true
}

func (w *capWriter) String() string { return w.buf.String() }
