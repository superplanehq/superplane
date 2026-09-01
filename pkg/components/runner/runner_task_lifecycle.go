package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
)

func afterRunnerTaskCreated(ctx core.ExecutionContext, taskID string) error {
	if err := ctx.ExecutionState.SetKV("task_id", taskID); err != nil {
		return fmt.Errorf("set task id in kv: %w", err)
	}
	if err := mergeRunnerBrokerTaskID(ctx.Metadata, taskID); err != nil {
		return fmt.Errorf("runner execution metadata: %w", err)
	}
	return ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{
		"task_id":         taskID,
		"organization_id": ctx.OrganizationID,
	}, pollInterval)
}

func pollBrokerTask(ctx core.ActionHookContext, finishedEventType string) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	taskID, ok := ctx.Parameters["task_id"].(string)
	if !ok {
		return fmt.Errorf("task_id is missing from parameters")
	}
	organizationID, _ := ctx.Parameters["organization_id"].(string)

	broker, err := NewBrokerClient(ctx.HTTP)
	if err != nil {
		return fmt.Errorf("new broker client: %w", err)
	}

	task, err := broker.FetchTaskStatus(taskID)
	if err != nil {
		ctx.Logger.WithError(err).Warn("runner: broker poll failed, will retry")
		return ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{
			"task_id":         taskID,
			"organization_id": organizationID,
		}, pollInterval)
	}

	sink := taskLogFromBrokerTask(task)
	if err := mergeRunnerTaskLog(ctx.Metadata, taskID, sink); err != nil {
		ctx.Logger.WithError(err).Warn("runner: execution metadata update failed")
	}

	if task.IsInTerminalState() {
		return processBrokerTaskStatus(ctx.ExecutionState, task, finishedEventType, organizationID, ctx.Logger, ctx.Usage, ctx.Configuration)
	}

	return ctx.Requests.ScheduleActionCall(hookActionPoll, map[string]any{
		"task_id":         taskID,
		"organization_id": organizationID,
	}, pollInterval)
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

	taskID := task.brokerTaskID()
	executionCtx, err := ctx.FindExecutionByKV("task_id", taskID)
	if err != nil {
		return http.StatusNotFound, nil, nil
	}

	sink := taskLogFromBrokerTask(task)
	if sink == nil {
		sink = taskLogFromRawWebhook(raw)
	}
	if executionCtx.Metadata != nil {
		if err := mergeRunnerTaskLog(executionCtx.Metadata, taskID, sink); err != nil {
			ctx.Logger.WithError(err).Warn("runner: execution metadata update failed")
		}
	}

	if err := processBrokerTaskStatus(executionCtx.ExecutionState, task, finishedEventType, executionCtx.OrganizationID, ctx.Logger, executionCtx.Usage, executionCtx.Configuration); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("process task status: %w", err)
	}

	return http.StatusOK, nil, nil
}

func processBrokerTaskStatus(
	state core.ExecutionStateContext,
	task *Task,
	finishedEventType string,
	organizationID string,
	logger *log.Entry,
	usage core.UsageRecorder,
	configuration any,
) error {
	if state.IsFinished() {
		return nil
	}

	if !task.IsInTerminalState() {
		return fmt.Errorf("task is not in terminal state")
	}

	publishRunnerUsage(organizationID, task, logger)
	RecordRunnerLLMUsage(usage, logger, finishedEventType, configuration, task.Result)

	channel := FailedOutputChannel
	if strings.ToLower(strings.TrimSpace(task.Status)) == "succeeded" && task.effectiveExitCode() == 0 {
		channel = PassedOutputChannel
	}

	out := map[string]any{"status": task.Status, "exit_code": task.effectiveExitCode()}
	if errMsg := strings.TrimSpace(task.Error); errMsg != "" {
		out["error"] = errMsg
	}
	if v := brokerResultAsAny(task.Result); v != nil {
		out["result"] = v
	}
	return state.Emit(channel, finishedEventType, []any{out})
}

func publishRunnerUsage(organizationID string, task *Task, logger *log.Entry) {
	organizationID = strings.TrimSpace(organizationID)
	taskID := task.brokerTaskID()
	if organizationID == "" || taskID == "" {
		return
	}
	if task.ClaimedAt == nil || task.FinishedAt == nil {
		return
	}

	seconds := billableSeconds(task.FinishedAt.Sub(*task.ClaimedAt))
	if seconds == 0 {
		return
	}

	if err := messages.NewRunnerTaskFinishedMessage(organizationID, taskID, seconds).Publish(); err != nil {
		if logger != nil {
			logger.WithError(err).Warn("runner: failed to publish usage")
		}
	}
}

// billableSeconds rounds a task duration up to the next whole second. Clock skew
// between the runner and the broker can put finished_at before claimed_at, so
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
