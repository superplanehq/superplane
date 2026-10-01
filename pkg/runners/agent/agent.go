package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/pkg/runners/logspool"
	"github.com/superplanehq/superplane/pkg/runners/models"
	"github.com/superplanehq/superplane/pkg/runners/protocol"
)

const defaultExecutionTimeoutSeconds = 3600

type Config struct {
	BaseURL      string
	Registration protocol.Registration
	Version      string

	MaxOutputBytes      int
	MaxExecutionSeconds int
	TaskWorkDir         string
	ResetTaskHome       bool

	LogSpoolDirectory string
	LogChunkBytes     int64
	LogSpoolMaxBytes  int64

	ReconnectMin time.Duration
	ReconnectMax time.Duration
	Log          *slog.Logger
}

func DefaultConfig() Config {
	return Config{
		MaxOutputBytes:   512 * 1024,
		LogChunkBytes:    64 * 1024,
		LogSpoolMaxBytes: 10 * 1024 * 1024,
		ReconnectMin:     time.Second,
		ReconnectMax:     5 * time.Second,
	}
}

type Agent struct {
	HTTP   *http.Client
	Config Config
}

type sessionStatus struct {
	done chan struct{}
	err  error
}

type taskExecutionResult struct {
	ExitCode     int
	RunErr       error
	UserCanceled bool
	Result       json.RawMessage
}

func (r taskExecutionResult) errorMessage() string {
	if r.RunErr == nil {
		return ""
	}
	return r.RunErr.Error()
}

// Run executes one task for an ephemeral runner or continues accepting tasks
// for a reusable runner. A task completes only after all retained logs and its
// terminal result have been acknowledged.
func (a *Agent) Run(ctx context.Context) error {
	if a.HTTP == nil {
		a.HTTP = http.DefaultClient
	}
	if a.Config.MaxOutputBytes <= 0 {
		a.Config.MaxOutputBytes = 512 * 1024
	}

	session, err := protocol.NewSession(protocol.SessionConfig{
		BaseURL:      a.Config.BaseURL,
		Registration: a.Config.Registration,
		Version:      a.Config.Version,
		ReconnectMin: a.Config.ReconnectMin,
		ReconnectMax: a.Config.ReconnectMax,
	})
	if err != nil {
		return err
	}
	sessionContext, stopSession := context.WithCancel(ctx)
	defer stopSession()
	sessionStatus := &sessionStatus{done: make(chan struct{})}
	go func() {
		sessionStatus.err = session.Run(sessionContext)
		close(sessionStatus.done)
	}()

	receivedTask := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sessionStatus.done:
			err := sessionStatus.err
			if err == nil && !receivedTask {
				return errors.New("runner stopped before receiving a task")
			}
			return err
		case rawTask := <-session.Tasks():
			receivedTask = true
			if err := a.runTask(ctx, session, sessionStatus, rawTask); err != nil {
				return err
			}
			if !a.Config.Registration.Ephemeral {
				continue
			}
			a.logger().Info(
				"ephemeral runner finished",
				slog.String("runner_id", a.Config.Registration.RunnerID),
			)
			stopSession()
			select {
			case <-sessionStatus.done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}

func (a *Agent) runTask(
	ctx context.Context,
	session *protocol.Session,
	status *sessionStatus,
	rawTask json.RawMessage,
) error {
	var task api.TaskPayload
	if err := json.Unmarshal(rawTask, &task); err != nil {
		return fmt.Errorf("decode task payload: %w", err)
	}
	if strings.TrimSpace(task.ID) == "" {
		return errors.New("task payload does not contain an ID")
	}
	task.Environment = useRunnerBaseURL(
		task.Environment,
		a.Config.BaseURL,
	)
	log := a.logger()
	executionMode := strings.ToLower(strings.TrimSpace(task.ExecutionMode))
	if executionMode == "" {
		executionMode = string(models.ExecutionHost)
	}
	log.Info(
		"task received",
		slog.String("task_id", task.ID),
		slog.String("run_mode", string(api.RunModeForTask(&task))),
		slog.String("execution_mode", executionMode),
	)

	taskContext, cancelTask := context.WithCancel(ctx)
	defer cancelTask()
	sessionStopped := make(chan error, 1)
	go func() {
		select {
		case <-status.done:
			err := status.err
			if err == nil {
				err = errors.New("runner session stopped")
			}
			sessionStopped <- err
			cancelTask()
		case <-taskContext.Done():
		}
	}()

	chunkBytes := a.Config.LogChunkBytes
	flushMinimum := time.Duration(0)
	flushMaximum := time.Duration(0)
	if task.LogUploadPolicy != nil {
		policy := api.NormalizeLogUploadPolicy(task.LogUploadPolicy)
		chunkBytes = policy.TargetChunkBytes
		flushMinimum = time.Duration(policy.PartialFlushMinimumMS) * time.Millisecond
		flushMaximum = time.Duration(policy.PartialFlushMaximumMS) * time.Millisecond
	}
	spool, err := logspool.New(logspool.Config{
		Directory:        a.Config.LogSpoolDirectory,
		TaskID:           task.ID,
		BaseURL:          a.Config.BaseURL,
		AccessToken:      a.Config.Registration.AccessToken,
		ChunkBytes:       chunkBytes,
		MaxBytes:         a.Config.LogSpoolMaxBytes,
		FlushIntervalMin: flushMinimum,
		FlushIntervalMax: flushMaximum,
		HTTPClient:       a.HTTP,
		Log:              log,
	})
	if err != nil {
		return fmt.Errorf("create task log spool: %w", err)
	}

	startedAt := time.Now()
	log.Info("task execution started", slog.String("task_id", task.ID))
	execution := a.execute(
		taskContext,
		&task,
		spool,
		session.Cancellations(),
	)
	log.Info(
		"task execution finished",
		slog.String("task_id", task.ID),
		slog.Int("exit_code", execution.ExitCode),
		slog.Bool("canceled", execution.UserCanceled),
		slog.Duration("duration", time.Since(startedAt)),
	)
	if err := spool.Close(); err != nil {
		return fmt.Errorf("close task log spool: %w", err)
	}
	if err := spool.Wait(taskContext); err != nil {
		select {
		case sessionErr := <-sessionStopped:
			return sessionErr
		default:
		}
		return fmt.Errorf("wait for task log acknowledgements: %w", err)
	}
	log.Info("task logs acknowledged", slog.String("task_id", task.ID))

	completion := protocol.CompleteMessage{
		RequestID: uuid.NewString(),
		TaskID:    task.ID,
		ExitCode:  int32(execution.ExitCode),
		Error:     execution.errorMessage(),
		Result:    execution.Result,
		Canceled:  execution.UserCanceled,
	}
	if err := session.Complete(taskContext, completion); err != nil {
		select {
		case sessionErr := <-sessionStopped:
			return sessionErr
		default:
		}
		return fmt.Errorf("complete task: %w", err)
	}
	log.Info(
		"task completion acknowledged",
		slog.String("task_id", task.ID),
	)
	return nil
}

func (a *Agent) execute(
	ctx context.Context,
	task *api.TaskPayload,
	live *logspool.Spool,
	cancellations <-chan string,
) taskExecutionResult {
	executor, err := a.executorFor(task)
	if err != nil {
		return taskExecutionResult{ExitCode: 1, RunErr: err}
	}

	resultPath := filepath.Join(os.TempDir(), "superplane-result-"+task.ID+".json")
	_ = os.Remove(resultPath)

	timeoutContext, cancelTimeout := context.WithTimeout(
		ctx,
		executionWallDuration(a.Config, task),
	)
	defer cancelTimeout()
	executionContext, cancelExecution := context.WithCancel(timeoutContext)
	defer cancelExecution()

	var canceled atomic.Bool
	go func() {
		for {
			select {
			case <-executionContext.Done():
				return
			case taskID := <-cancellations:
				if taskID == task.ID {
					a.logger().Info(
						"task cancellation received",
						slog.String("task_id", task.ID),
					)
					canceled.Store(true)
					cancelExecution()
					return
				}
			}
		}
	}()

	exitCode, _, runErr := executor.Execute(
		executionContext,
		task,
		live,
		resultPath,
	)
	resultLimit := a.Config.MaxOutputBytes
	if task.WebhookPayloadSizeLimit > 0 {
		webhookResultLimit := task.WebhookPayloadSizeLimit - 1024
		if resultLimit <= 0 || webhookResultLimit < resultLimit {
			resultLimit = webhookResultLimit
		}
	}
	result, resultTooLarge := readTaskResultFile(resultPath, resultLimit, a.Config.Log)
	if resultTooLarge {
		return taskExecutionResult{
			ExitCode: 1,
			RunErr: fmt.Errorf(
				"result payload too large: exceeds %d byte limit",
				resultLimit,
			),
		}
	}
	if canceled.Load() {
		return taskExecutionResult{
			ExitCode:     exitCode,
			RunErr:       runErr,
			UserCanceled: true,
			Result:       result,
		}
	}
	if errors.Is(executionContext.Err(), context.DeadlineExceeded) ||
		errors.Is(runErr, context.DeadlineExceeded) {
		return taskExecutionResult{
			ExitCode: 124,
			RunErr:   errors.New("execution timed out"),
			Result:   result,
		}
	}
	return taskExecutionResult{
		ExitCode: exitCode,
		RunErr:   runErr,
		Result:   result,
	}
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
			ResetTaskHome:  a.Config.ResetTaskHome,
		}, nil
	case models.ExecutionDocker:
		return &DockerExecutor{
			MaxOutputBytes: a.Config.MaxOutputBytes,
			RunnerID:       a.Config.Registration.RunnerID,
		}, nil
	default:
		return nil, fmt.Errorf("unknown execution_mode %q", task.ExecutionMode)
	}
}

func (a *Agent) logger() *slog.Logger {
	if a.Config.Log != nil {
		return a.Config.Log
	}
	return slog.Default()
}

func executionWallDuration(config Config, task *api.TaskPayload) time.Duration {
	seconds := defaultExecutionTimeoutSeconds
	if task.ExecutionTimeoutSeconds != nil && *task.ExecutionTimeoutSeconds > 0 {
		seconds = *task.ExecutionTimeoutSeconds
	}
	if config.MaxExecutionSeconds > 0 && seconds > config.MaxExecutionSeconds {
		seconds = config.MaxExecutionSeconds
	}
	return time.Duration(seconds) * time.Second
}

func truncateString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "\n…(truncated)"
}
