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
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/superplane/runner/runner/internal/cloudwatchlog"
	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/cwstream"
	"github.com/superplane/runner/shared/models"
)

// Config controls runner behavior.
type Config struct {
	BaseURL  string // task-broker base URL (TASK_BROKER_URL)
	FleetID  string // fleet to claim tasks from (RUNNER_FLEET_ID)
	RunnerID string
	Token    string
	// Transport selects task-broker API: default WebSocket (GET /v1/runners/stream). Set "http", "polling", or "legacy" for POST claim/complete.
	Transport string
	PollEmpty time.Duration
	// MaxOutputBytes caps combined stdout+stderr stored and sent back.
	MaxOutputBytes int
	// MaxExecutionSeconds caps the runner's execution wall clock (0 = no cap). It does not change
	// fleet-manager claim leases, which are derived from execution_timeout_seconds on the task (or the API default).
	MaxExecutionSeconds int
	// ExitAfterEachTask stops the runner process after one successful CompleteTask.
	ExitAfterEachTask bool
	Log               *slog.Logger // optional: fleet_manager_http lines for claim / complete

	// CloudWatchLogGroup when non-empty streams task stdout/stderr to Amazon CloudWatch Logs
	// (one log stream per task; see shared/cwstream.TaskLogStream).
	CloudWatchLogGroup        string
	CloudWatchRegion          string // optional; uses default AWS credential chain region when empty
	CloudWatchLogStreamPrefix string // optional prefix for log stream name (must match TASK_CLOUDWATCH_LOG_STREAM_PREFIX on fleet-manager for callers)
	// TaskWorkDir is the initial working directory for host-mode tasks (runner user's $HOME).
	TaskWorkDir string
}

// DefaultConfig returns safe defaults.
func DefaultConfig() Config {
	return Config{
		PollEmpty:      time.Second,
		MaxOutputBytes: 512 * 1024,
	}
}

// Agent connects to task-broker, executes tasks, and reports results.
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
	base := a.fleetBase()
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
		exit, _, runErr, userCanceled, result := a.execute(ctx, base, task, nil)
		errMsg := ""
		if runErr != nil {
			errMsg = runErr.Error()
		}
		if err := a.complete(ctx, base, task.ID, exit, errMsg, userCanceled, result); err != nil {
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
		FleetID:      a.Config.FleetID,
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
		a.Config.Log.Info("task_broker_http",
			slog.String("op", op),
			slog.Int("http_status", code),
			slog.Duration("dur", dur),
			slog.String("runner_id", a.Config.RunnerID),
			slog.String("task_id", out.Task.ID),
		)
	}
	return out.Task, nil
}

func (a *Agent) complete(ctx context.Context, base, id string, exit int, errMsg string, canceled bool, result json.RawMessage) error {
	payload := api.CompleteTaskRequest{
		RunnerID: a.Config.RunnerID,
		ExitCode: exit,
		Error:    errMsg,
		Canceled: canceled,
		Result:   result,
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
		a.Config.Log.Info("task_broker_http",
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
	a.Config.Log.Warn("task_broker_http", args...)
}

const cancelPollInterval = 1500 * time.Millisecond

func (a *Agent) fleetBase() string {
	return strings.TrimRight(strings.TrimSpace(a.Config.BaseURL), "/")
}

func (a *Agent) getTaskStatus(ctx context.Context, base, id string) (*api.TaskStatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/v1/tasks/"+id, nil)
	if err != nil {
		return nil, err
	}
	a.auth(req)
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("get task: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out api.TaskStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *Agent) execute(ctx context.Context, base string, task *api.TaskPayload, wsPushCancel <-chan struct{}) (int, string, error, bool, json.RawMessage) {
	ex, err := a.executorFor(task)
	if err != nil {
		return 1, "", err, false, nil
	}

	resultPath := filepath.Join(os.TempDir(), "superplane-result-"+task.ID+".json")
	_ = os.Remove(resultPath)

	d := executionWallDuration(a.Config, task)
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, d)
	defer cancelTimeout()

	execCtx, cancelExec := context.WithCancel(timeoutCtx)
	defer cancelExec()

	var stoppedByCancel atomic.Bool
	if wsPushCancel != nil {
		go func() {
			for {
				select {
				case <-execCtx.Done():
					return
				case <-ctx.Done():
					return
				case <-wsPushCancel:
					stoppedByCancel.Store(true)
					cancelExec()
					return
				}
			}
		}()
	} else {
		go func() {
			tick := time.NewTicker(cancelPollInterval)
			defer tick.Stop()
			for {
				select {
				case <-execCtx.Done():
					return
				case <-ctx.Done():
					return
				case <-tick.C:
					qctx, qc := context.WithTimeout(ctx, 8*time.Second)
					st, err := a.getTaskStatus(qctx, base, task.ID)
					qc()
					if err != nil {
						continue
					}
					if st.CancelRequested || strings.EqualFold(st.Status, string(models.StatusCanceled)) {
						stoppedByCancel.Store(true)
						cancelExec()
						return
					}
				}
			}
		}()
	}

	// Optional live-stream of stdout/stderr to CloudWatch Logs while the
	// task runs. Task logs are read from CloudWatch (task_log on webhooks);
	// completion payloads do not include inline output.
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

	exit, out, runErr := ex.Execute(execCtx, task, live, resultPath)
	result := readTaskResultFile(resultPath, a.Config.MaxOutputBytes, a.Config.Log)

	if stoppedByCancel.Load() {
		return exit, out, runErr, true, result
	}
	if errors.Is(execCtx.Err(), context.DeadlineExceeded) || errors.Is(runErr, context.DeadlineExceeded) {
		return 124, out, errors.New("execution timed out"), false, result
	}
	return exit, out, runErr, false, result
}

func (a *Agent) executorFor(task *api.TaskPayload) (Executor, error) {
	mode := models.ExecutionMode(strings.ToLower(strings.TrimSpace(task.ExecutionMode)))
	if mode == "" {
		mode = models.ExecutionHost
	}
	switch mode {
	case models.ExecutionHost:
		return &HostExecutor{
			MaxOutputBytes: a.Config.MaxOutputBytes,
			TaskWorkDir:    a.Config.TaskWorkDir,
		}, nil
	case models.ExecutionDocker:
		return &DockerExecutor{
			MaxOutputBytes: a.Config.MaxOutputBytes,
			RunnerID:       a.Config.RunnerID,
		}, nil
	default:
		return nil, fmt.Errorf("unknown execution_mode %q", task.ExecutionMode)
	}
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

func truncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n…(truncated)"
}
