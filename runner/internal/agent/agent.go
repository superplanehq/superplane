package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/superplane/runner/runner/internal/cloudwatchlog"
	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/cwstream"
	"github.com/superplane/runner/shared/models"
)

// Config controls runner behavior.
type Config struct {
	BaseURL  string
	RunnerID string
	Token    string
	// Transport selects fleet-manager API: default WebSocket (GET /v1/runners/stream). Set "http", "polling", or "legacy" for POST claim/complete.
	Transport string
	PollEmpty time.Duration
	// MaxOutputBytes caps combined stdout+stderr stored and sent back.
	MaxOutputBytes int
	// MaxExecutionSeconds caps the runner's execution wall clock (0 = no cap). It does not change
	// fleet-manager claim leases, which are derived from execution_timeout_seconds on the task (or the API default).
	MaxExecutionSeconds int
	// ExitAfterEachTask stops the runner process after one successful CompleteTask once fleet-manager accepts the result.
	// Fleet-manager terminates the EC2 instance when runner_id is the instance id (see cloud-init user-data). Local env: RUNNER_TERMINATE_AFTER_EACH_TASK.
	ExitAfterEachTask bool
	Log               *slog.Logger // optional: fleet_manager_http lines for claim / complete

	// CloudWatchLogGroup when non-empty streams task stdout/stderr to Amazon CloudWatch Logs
	// (one log stream per task; see shared/cwstream.TaskLogStream).
	CloudWatchLogGroup        string
	CloudWatchRegion          string // optional; uses default AWS credential chain region when empty
	CloudWatchLogStreamPrefix string // optional prefix for log stream name (must match TASK_CLOUDWATCH_LOG_STREAM_PREFIX on fleet-manager for callers)
}

// DefaultConfig returns safe defaults.
func DefaultConfig() Config {
	return Config{
		PollEmpty:      time.Second,
		MaxOutputBytes: 512 * 1024,
	}
}

// Agent polls fleet-manager, executes tasks, and reports results.
type Agent struct {
	HTTP   *http.Client
	Config Config
}

// Run blocks until ctx is cancelled, processing tasks in a loop.
func (a *Agent) Run(ctx context.Context) error {
	if a.HTTP == nil {
		a.HTTP = http.DefaultClient
	}
	if a.Config.MaxOutputBytes <= 0 {
		a.Config.MaxOutputBytes = 512 * 1024
	}
	if a.Config.PollEmpty <= 0 {
		a.Config.PollEmpty = time.Second
	}
	if transportWebSocket(a.Config) {
		return RunWebSocket(ctx, a)
	}
	base := strings.TrimRight(a.Config.BaseURL, "/")
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		task, err := a.claim(ctx, base)
		if err != nil {
			return err
		}
		if task == nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(a.Config.PollEmpty):
			}
			continue
		}
		exit, out, runErr := a.execute(ctx, task)
		errMsg := ""
		if runErr != nil {
			errMsg = runErr.Error()
		}
		if err := a.complete(ctx, base, task.ID, exit, out, errMsg); err != nil {
			return err
		}
		if a.Config.ExitAfterEachTask {
			return nil
		}
	}
}

func (a *Agent) claim(ctx context.Context, base string) (*api.TaskPayload, error) {
	body, err := json.Marshal(api.ClaimTaskRequest{
		RunnerID:     a.Config.RunnerID,
		LeaseSeconds: int((10 * time.Minute).Seconds()),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/tasks/claim", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	a.auth(req)

	start := time.Now()
	resp, err := a.HTTP.Do(req)
	dur := time.Since(start)
	op := "claim_task"
	if err != nil {
		a.logFleetHTTPWarn(op, dur, 0, "", err)
		return nil, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		err := fmt.Errorf("claim: status %d: %s", code, strings.TrimSpace(string(b)))
		a.logFleetHTTPWarn(op, dur, code, "", err)
		return nil, err
	}
	var out api.ClaimTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		a.logFleetHTTPWarn(op, dur, code, "", err)
		return nil, err
	}
	if out.Task != nil && a.Config.Log != nil {
		a.Config.Log.Info("fleet_manager_http",
			slog.String("op", op),
			slog.Int("http_status", code),
			slog.Duration("dur", dur),
			slog.String("runner_id", a.Config.RunnerID),
			slog.String("task_id", out.Task.ID),
		)
	}
	return out.Task, nil
}

func (a *Agent) complete(ctx context.Context, base, id string, exit int, output, errMsg string) error {
	payload := api.CompleteTaskRequest{
		RunnerID: a.Config.RunnerID,
		ExitCode: exit,
		Output:   output,
		Error:    errMsg,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/tasks/"+id+"/complete", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	a.auth(req)

	op := "complete_task"
	start := time.Now()
	resp, err := a.HTTP.Do(req)
	dur := time.Since(start)
	if err != nil {
		a.logFleetHTTPWarn(op, dur, 0, id, err)
		return err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		e := fmt.Errorf("complete: status %d: %s", code, strings.TrimSpace(string(b)))
		a.logFleetHTTPWarn(op, dur, code, id, e)
		return e
	}
	if a.Config.Log != nil {
		a.Config.Log.Info("fleet_manager_http",
			slog.String("op", op),
			slog.Int("http_status", code),
			slog.Duration("dur", dur),
			slog.String("runner_id", a.Config.RunnerID),
			slog.String("task_id", id),
		)
	}
	return nil
}

func (a *Agent) auth(req *http.Request) {
	if t := strings.TrimSpace(a.Config.Token); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
}

func (a *Agent) logFleetHTTPWarn(op string, dur time.Duration, status int, taskID string, err error) {
	if a.Config.Log == nil {
		return
	}
	args := []any{
		slog.String("op", op),
		slog.Duration("dur", dur),
		slog.String("runner_id", a.Config.RunnerID),
		slog.Any("err", err),
	}
	if status > 0 {
		args = append(args, slog.Int("http_status", status))
	}
	if taskID != "" {
		args = append(args, slog.String("task_id", taskID))
	}
	a.Config.Log.Warn("fleet_manager_http", args...)
}

func (a *Agent) execute(ctx context.Context, task *api.TaskPayload) (int, string, error) {
	mode := models.ExecutionMode(strings.ToLower(strings.TrimSpace(task.ExecutionMode)))
	if mode == "" {
		mode = models.ExecutionHost
	}

	d := executionWallDuration(a.Config, task)
	execCtx, cancel := context.WithTimeout(ctx, d)
	defer cancel()

	var live io.Writer
	var cwClose func()
	if g := strings.TrimSpace(a.Config.CloudWatchLogGroup); g != "" {
		stream := cwstream.TaskLogStream(strings.TrimSpace(a.Config.CloudWatchLogStreamPrefix), task.ID)
		sw, err := cloudwatchlog.NewStreamWriter(execCtx, cloudwatchlog.StreamConfig{
			LogGroup:   g,
			StreamName: stream,
			Region:     strings.TrimSpace(a.Config.CloudWatchRegion),
		})
		if err != nil {
			if a.Config.Log != nil {
				a.Config.Log.Warn("cloudwatch_log_stream", slog.String("task_id", task.ID), slog.Any("err", err))
			}
		} else {
			live = sw
			cwClose = func() { _ = sw.Close() }
		}
	}
	if cwClose != nil {
		defer cwClose()
	}

	var exit int
	var out string
	var err error
	switch mode {
	case models.ExecutionDocker:
		exit, out, err = a.runDocker(execCtx, task, live)
	case models.ExecutionHost:
		exit, out, err = a.runHost(execCtx, task, live)
	default:
		return 1, "", fmt.Errorf("unknown execution_mode %q", task.ExecutionMode)
	}
	if errors.Is(execCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return 124, out, errors.New("execution timed out")
	}
	return exit, out, err
}

func executionWallDuration(cfg Config, task *api.TaskPayload) time.Duration {
	sec := api.DefaultExecutionTimeoutSeconds
	if task.ExecutionTimeoutSeconds != nil && *task.ExecutionTimeoutSeconds > 0 {
		sec = *task.ExecutionTimeoutSeconds
	}
	if cfg.MaxExecutionSeconds > 0 && sec > cfg.MaxExecutionSeconds {
		sec = cfg.MaxExecutionSeconds
	}
	return time.Duration(sec) * time.Second
}

func (a *Agent) runHost(ctx context.Context, task *api.TaskPayload, live io.Writer) (int, string, error) {
	if len(task.Commands) > 0 {
		return a.runHostShellScripts(ctx, task.Commands, live)
	}
	if len(task.Command) == 0 {
		return 1, "", errors.New("empty command")
	}
	cmd := exec.CommandContext(ctx, task.Command[0], task.Command[1:]...)
	var buf bytes.Buffer
	if live != nil {
		mw := io.MultiWriter(&buf, live)
		cmd.Stdout = mw
		cmd.Stderr = mw
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}
	err := cmd.Run()
	out := truncateString(buf.String(), a.Config.MaxOutputBytes)
	exit := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		} else {
			exit = 1
		}
		return exit, out, err
	}
	return exit, out, nil
}

func (a *Agent) runHostShellScripts(ctx context.Context, scripts []string, live io.Writer) (int, string, error) {
	return runHostShellDirectives(ctx, a.Config.MaxOutputBytes, scripts, live)
}

func (a *Agent) runDocker(ctx context.Context, task *api.TaskPayload, live io.Writer) (int, string, error) {
	if strings.TrimSpace(task.DockerImage) == "" {
		return 1, "", errors.New("docker_image required")
	}
	if len(task.Commands) > 0 {
		return a.runDockerShellScripts(ctx, task.DockerImage, task.Commands, live)
	}
	args := []string{"run", "--rm", task.DockerImage}
	args = append(args, task.Command...)
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
	err := cmd.Run()
	out := truncateString(buf.String(), a.Config.MaxOutputBytes)
	exit := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		} else {
			exit = 1
		}
		return exit, out, err
	}
	return exit, out, nil
}

func (a *Agent) runDockerShellScripts(ctx context.Context, image string, scripts []string, live io.Writer) (int, string, error) {
	return runDockerShellDirectives(ctx, a.Config.MaxOutputBytes, image, scripts, live)
}

func truncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n…(truncated)"
}
