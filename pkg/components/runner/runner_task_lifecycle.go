package runner

import (
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
)

func afterRunnerTaskCreated(ctx core.ExecutionContext, taskID string) error {
	if err := ctx.ExecutionState.SetKV("task_id", taskID); err != nil {
		return fmt.Errorf("set task id in kv: %w", err)
	}
	if err := storeRunnerFleetKV(ctx, ""); err != nil {
		return fmt.Errorf("set runner fleet kv: %w", err)
	}
	if err := mergeRunnerBrokerTaskID(ctx.Metadata, taskID); err != nil {
		return fmt.Errorf("runner execution metadata: %w", err)
	}
	if err := mergeRunnerTaskBackend(ctx.Metadata, core.RunnerTaskBackendIntegrated); err != nil {
		return fmt.Errorf("runner task backend metadata: %w", err)
	}
	return scheduleTaskPoll(ctx.Requests, taskID, ctx.OrganizationID)
}

func pollTask(ctx core.ActionHookContext, finishedEventType string) error {
	taskID, ok := ctx.Parameters["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		if ctx.ExecutionState.IsFinished() {
			revokeOpenRouterChildKeyOrReschedule(ctx)
			return nil
		}
		return fmt.Errorf("task_id is missing from parameters")
	}
	organizationID, _ := ctx.Parameters["organization_id"].(string)

	if ctx.RunnerTasks == nil {
		if ctx.ExecutionState.IsFinished() {
			revokeOpenRouterChildKeyOrReschedule(ctx)
			return nil
		}
		return fmt.Errorf("runner task context is unavailable")
	}

	task, err := ctx.RunnerTasks.Find(taskID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("runner: task status check failed, will retry")
		}
		if ctx.ExecutionState.IsFinished() {
			revokeOpenRouterChildKeyOrReschedule(ctx)
		}
		return scheduleTaskPoll(ctx.Requests, taskID, organizationID)
	}

	if task.IsTerminal() {
		err := processTaskStatus(ctx.ExecutionState, task, finishedEventType, organizationID, ctx.Logger, ctx.Usage, ctx.Configuration)
		revokeOpenRouterChildKeyOrReschedule(ctx)
		return err
	}

	if ctx.ExecutionState.IsFinished() {
		revokeOpenRouterChildKeyOrReschedule(ctx)
	}
	return scheduleTaskPoll(ctx.Requests, taskID, organizationID)
}

func processTaskStatus(
	state core.ExecutionStateContext,
	task *core.RunnerTask,
	finishedEventType string,
	organizationID string,
	logger *log.Entry,
	usage core.UsageRecorder,
	configuration any,
) error {
	if !task.IsTerminal() {
		if state.IsFinished() {
			return nil
		}
		return fmt.Errorf("task is not in terminal state")
	}

	// Record final usage even if the node finished before the runner task.
	RecordRunnerLLMUsage(usage, logger, finishedEventType, configuration, task.Result)
	RecordRunnerComputeUsage(usage, logger, state, configuration, task)

	if state.IsFinished() {
		return nil
	}

	if taskCanceled(task) && isAnalysisSessionExecution(state) {
		return state.Cancel()
	}

	channel := FailedOutputChannel
	if strings.ToLower(strings.TrimSpace(task.State)) == "succeeded" && taskExitCode(task) == 0 {
		channel = PassedOutputChannel
	}

	status := task.State
	result := taskResultAsAny(task.Result)
	errorMessage := taskFailureMessage(task, finishedEventType, result)
	if status == "lost" {
		status = "failed"
		if errorMessage == "" {
			errorMessage = "runner task was lost"
		}
	}
	out := map[string]any{"status": status, "exit_code": taskExitCode(task)}
	if errorMessage != "" {
		out["error"] = errorMessage
	}
	if result != nil {
		out["result"] = result
	}
	if channel == FailedOutputChannel && !taskCanceled(task) && logger != nil {
		fields := log.Fields{
			"component":       "runner.task",
			"organization_id": organizationID,
			"task_id":         task.ID,
			"event_type":      finishedEventType,
			"status":          status,
			"exit_code":       taskExitCode(task),
		}
		if errorMessage != "" {
			fields["error"] = errorMessage
		}
		logger.WithFields(fields).Warn("runner: task failed")
	}
	return state.Emit(channel, finishedEventType, []any{out})
}

func taskFailureMessage(task *core.RunnerTask, finishedEventType string, result any) string {
	message := strings.TrimSpace(task.ErrorMessage)
	if finishedEventType != "runnerCodex.finished" || !strings.EqualFold(strings.TrimSpace(task.State), "failed") {
		return message
	}
	if message != "" &&
		message != fmt.Sprintf("exit code %d", taskExitCode(task)) &&
		message != fmt.Sprintf("exit status %d", taskExitCode(task)) {
		return message
	}

	payload, ok := result.(map[string]any)
	if !ok || payload["type"] != "result" {
		return message
	}
	isError, _ := payload["is_error"].(bool)
	providerError, _ := payload["error"].(string)
	if isError && strings.TrimSpace(providerError) != "" {
		return strings.TrimSpace(providerError)
	}
	return message
}

func taskCanceled(task *core.RunnerTask) bool {
	return strings.EqualFold(strings.TrimSpace(task.State), "canceled")
}

func markAnalysisSession(state core.ExecutionStateContext) {
	if state == nil {
		return
	}
	_ = state.SetKV(executionKVAnalysisSession, "true")
}

func isAnalysisSessionExecution(state core.ExecutionStateContext) bool {
	return executionKV(state, executionKVAnalysisSession) == "true"
}

// billableSeconds rounds a task duration up to the next whole second. Clock skew
// can put the finish time before the start time, so
// non-positive durations bill nothing rather than rounding up to one second.
func billableSeconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}

	seconds := int64(duration / time.Second)
	if duration%time.Second != 0 {
		seconds++
	}
	return seconds
}

func cancelTask(ctx core.ExecutionContext, finishedEventType string) error {
	if ctx.ExecutionState.IsFinished() {
		_ = revokeOpenRouterChildKey(ctx.HTTP, ctx.ExecutionState, ctx.HostedLLM, ctx.Logger)
		return nil
	}

	taskID, err := ctx.ExecutionState.GetKV("task_id")
	if err != nil {
		if errors.Is(err, core.ErrExecutionKVNotFound) {
			_ = revokeOpenRouterChildKey(ctx.HTTP, ctx.ExecutionState, ctx.HostedLLM, ctx.Logger)
			return nil
		}
		return fmt.Errorf("get task_id kv: %w", err)
	}

	if ctx.RunnerTasks == nil {
		return fmt.Errorf("runner task context is unavailable")
	}

	if err := ctx.RunnerTasks.RequestCancel(taskID); err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}

	if err := recordTerminalTaskUsage(ctx, taskID, finishedEventType); err != nil {
		_ = revokeOpenRouterChildKey(ctx.HTTP, ctx.ExecutionState, ctx.HostedLLM, ctx.Logger)
		return err
	}

	_ = revokeOpenRouterChildKey(ctx.HTTP, ctx.ExecutionState, ctx.HostedLLM, ctx.Logger)
	return nil
}

func recordTerminalTaskUsage(ctx core.ExecutionContext, taskID, finishedEventType string) error {
	task, err := ctx.RunnerTasks.Find(taskID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("runner: fetch after cancel failed")
		}
		return scheduleTaskPollAfterCancel(ctx, taskID)
	}
	if !task.IsTerminal() {
		return scheduleTaskPollAfterCancel(ctx, taskID)
	}
	return processTaskStatus(
		ctx.ExecutionState,
		task,
		finishedEventType,
		ctx.OrganizationID,
		ctx.Logger,
		ctx.Usage,
		ctx.Configuration,
	)
}

func scheduleTaskPoll(requests core.RequestContext, taskID, organizationID string) error {
	if requests == nil {
		return nil
	}
	return requests.ScheduleActionCall(hookActionPoll, map[string]any{
		"task_id":         taskID,
		"organization_id": organizationID,
	}, pollInterval)
}

func scheduleTaskPollAfterCancel(ctx core.ExecutionContext, taskID string) error {
	if err := scheduleTaskPoll(ctx.Requests, taskID, ctx.OrganizationID); err != nil && ctx.Logger != nil {
		ctx.Logger.WithError(err).Warn("runner: failed to schedule poll after cancel")
	}
	return nil
}

func taskExitCode(task *core.RunnerTask) int {
	if task.ExitCode == nil {
		return 0
	}
	return *task.ExitCode
}
