package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// TaskOutcome maps a finished broker task onto a component's output channels and
// finished event type. It lets components other than Runner reuse the broker
// task lifecycle (create → poll/webhook → emit) while keeping their own channel
// names and event types.
type TaskOutcome struct {
	FinishedEventType string
	PassedChannel     string
	FailedChannel     string
}

func (o TaskOutcome) passedChannel() string {
	if strings.TrimSpace(o.PassedChannel) == "" {
		return PassedOutputChannel
	}
	return o.PassedChannel
}

func (o TaskOutcome) failedChannel() string {
	if strings.TrimSpace(o.FailedChannel) == "" {
		return FailedOutputChannel
	}
	return o.FailedChannel
}

// AfterTaskCreated records the broker task id on the execution and schedules the
// first status poll. Components call this right after BrokerClient.CreateTask so
// the shared poll/webhook/cancel and live-log machinery can take over.
func AfterTaskCreated(ctx core.ExecutionContext, taskID string) error {
	if err := ctx.ExecutionState.SetKV("task_id", taskID); err != nil {
		return fmt.Errorf("set task id in kv: %w", err)
	}
	if err := mergeRunnerBrokerTaskID(ctx.Metadata, taskID); err != nil {
		return fmt.Errorf("runner execution metadata: %w", err)
	}
	return ctx.Requests.ScheduleActionCall(HookActionPoll, map[string]any{"task_id": taskID}, pollInterval)
}

// PollTask fetches the broker task status, refreshes live-log metadata, and
// emits the finished event once the task reaches a terminal state. It reschedules
// itself while the task is still running.
func PollTask(ctx core.ActionHookContext, outcome TaskOutcome) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	taskID, ok := ctx.Parameters["task_id"].(string)
	if !ok {
		return fmt.Errorf("task_id is missing from parameters")
	}

	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	task, err := broker.FetchTaskStatus(taskID)
	if err != nil {
		ctx.Logger.WithError(err).Warn("runner: broker poll failed, will retry")
		return ctx.Requests.ScheduleActionCall(HookActionPoll, map[string]any{"task_id": taskID}, pollInterval)
	}

	sink := taskLogFromBrokerTask(task)
	if err := mergeRunnerTaskLog(ctx.Metadata, taskID, sink); err != nil {
		ctx.Logger.WithError(err).Warn("runner: execution metadata update failed")
	}

	if task.IsInTerminalState() {
		return processBrokerTaskStatus(ctx.ExecutionState, task, outcome)
	}

	return ctx.Requests.ScheduleActionCall(HookActionPoll, map[string]any{"task_id": taskID}, pollInterval)
}

// HandleTaskWebhook processes a broker task webhook callback, refreshes live-log
// metadata, and emits the finished event on the resolved execution.
func HandleTaskWebhook(ctx core.WebhookRequestContext, outcome TaskOutcome) (int, *core.WebhookResponseBody, error) {
	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("new broker client: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(ctx.Body, &raw); err != nil {
		raw = nil
	}

	task, err := broker.ProcessWebhook(ctx.Body)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("process webhook: %w", err)
	}

	if !task.IsInTerminalState() {
		ctx.Logger.Warn("runner: broker webhook received non-terminal state")
	}

	executionCtx, err := ctx.FindExecutionByKV("task_id", task.TaskID)
	if err != nil {
		return http.StatusNotFound, nil, nil
	}

	sink := taskLogFromBrokerTask(task)
	if sink == nil {
		sink = taskLogFromRawWebhook(raw)
	}
	if executionCtx.Metadata != nil {
		if err := mergeRunnerTaskLog(executionCtx.Metadata, task.TaskID, sink); err != nil {
			ctx.Logger.WithError(err).Warn("runner: execution metadata update failed")
		}
	}

	if err := processBrokerTaskStatus(executionCtx.ExecutionState, task, outcome); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("process task status: %w", err)
	}

	return http.StatusOK, nil, nil
}

func processBrokerTaskStatus(state core.ExecutionStateContext, task *Task, outcome TaskOutcome) error {
	if state.IsFinished() {
		return nil
	}

	if !task.IsInTerminalState() {
		return fmt.Errorf("task is not in terminal state")
	}

	channel := outcome.failedChannel()
	if strings.ToLower(strings.TrimSpace(task.Status)) == "succeeded" && task.effectiveExitCode() == 0 {
		channel = outcome.passedChannel()
	}

	out := map[string]any{"status": task.Status, "exit_code": task.effectiveExitCode()}
	if v := brokerResultAsAny(task.Result); v != nil {
		out["result"] = v
	}
	return state.Emit(channel, outcome.FinishedEventType, []any{out})
}

// CancelBrokerTask cancels the broker task linked to the execution, if any.
func CancelBrokerTask(ctx core.ExecutionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	taskID, err := ctx.ExecutionState.GetKV("task_id")
	if err != nil {
		if errors.Is(err, core.ErrExecutionKVNotFound) {
			return nil
		}
		return fmt.Errorf("get task_id kv: %w", err)
	}

	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return err
	}

	if err := broker.CancelTask(taskID); err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}
	return nil
}
