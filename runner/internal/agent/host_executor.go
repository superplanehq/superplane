package agent

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/superplane/runner/shared/api"
)

// HostExecutor runs tasks directly on the host process. It is a thin
// wrapper around the existing host shell directive path (PTY+markers or
// pipe-bundle via RUNNER_SHELL_USE_PIPE) and the argv-subprocess path.
type HostExecutor struct {
	MaxOutputBytes int
}

func (h *HostExecutor) Execute(ctx context.Context, task *api.TaskPayload, live io.Writer, resultHostPath string) (int, string, error) {
	max := h.MaxOutputBytes
	if max <= 0 {
		max = 512 * 1024
	}
	env, err := processEnvironment(task.Environment)
	if err != nil {
		return 1, "", err
	}
	if len(task.Commands) > 0 {
		return runHostShellDirectives(ctx, max, task.Commands, env, live, resultHostPath)
	}
	if len(task.Command) == 0 {
		return 1, "", errors.New("empty command")
	}
	cmd := exec.CommandContext(ctx, task.Command[0], task.Command[1:]...)
	setResultEnv(cmd, resultHostPath)
	if env != nil {
		cmd.Env = env
	}
	var buf bytes.Buffer
	if live != nil {
		mw := io.MultiWriter(&buf, live)
		cmd.Stdout = mw
		cmd.Stderr = mw
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}
	writeLiveLogCommandStart(live, 0, strings.Join(task.Command, " "))
	startedAt := time.Now()
	err = cmd.Run()
	out := truncateString(buf.String(), max)
	exit := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		} else {
			exit = 1
		}
		writeLiveLogCommandEnd(live, 0, exit, time.Since(startedAt))
		return exit, out, err
	}
	writeLiveLogCommandEnd(live, 0, exit, time.Since(startedAt))
	return exit, out, nil
}
