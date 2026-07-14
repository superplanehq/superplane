package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

func afterRunnerTaskCreated(ctx core.ExecutionContext, taskID string) error {
	if err := ctx.ExecutionState.SetKV("task_id", taskID); err != nil {
		return fmt.Errorf("set task id in kv: %w", err)
	}
	if err := mergeRunnerBrokerTaskID(ctx.Metadata, taskID); err != nil {
		return fmt.Errorf("runner execution metadata: %w", err)
	}
	return ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{"task_id": taskID}, pollInterval)
}

func pollBrokerTask(ctx core.ActionHookContext, finishedEventType string) error {
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
		return ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{"task_id": taskID}, pollInterval)
	}

	sink := taskLogFromBrokerTask(task)
	if err := mergeRunnerTaskLog(ctx.Metadata, taskID, sink); err != nil {
		ctx.Logger.WithError(err).Warn("runner: execution metadata update failed")
	}

	if task.IsInTerminalState() {
		return processBrokerTaskStatus(ctx.ExecutionState, task, finishedEventType)
	}

	return ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{"task_id": taskID}, pollInterval)
}

func handleBrokerWebhook(ctx core.WebhookRequestContext, finishedEventType string) (int, *core.WebhookResponseBody, error) {
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

	//
	// Once the execution is finished, its metadata and finished timestamp are
	// settled. Late or duplicate webhooks must not touch it again, otherwise the
	// metadata write would move the execution's finished_at timestamp.
	//
	if executionCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil, nil
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

	if err := processBrokerTaskStatus(executionCtx.ExecutionState, task, finishedEventType); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("process task status: %w", err)
	}

	return http.StatusOK, nil, nil
}

func processBrokerTaskStatus(state core.ExecutionStateContext, task *Task, finishedEventType string) error {
	if state.IsFinished() {
		return nil
	}

	if !task.IsInTerminalState() {
		return fmt.Errorf("task is not in terminal state")
	}

	channel := FailedOutputChannel
	if strings.ToLower(strings.TrimSpace(task.Status)) == "succeeded" && task.effectiveExitCode() == 0 {
		channel = PassedOutputChannel
	}

	out := map[string]any{"status": task.Status, "exit_code": task.effectiveExitCode()}
	if v := brokerResultAsAny(task.Result); v != nil {
		out["result"] = v
	}
	return state.Emit(channel, finishedEventType, []any{out})
}

func cancelBrokerTask(ctx core.ExecutionContext) error {
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
