package runner

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// Shared symbols for runner subpackages (e.g. runner/claude).

const (
	MachineTypeFieldName              = configurationFieldMachineType
	HookPoll                          = hookActionPoll
	MaxExecutionTimeoutSecondsRequest = maxExecutionTimeoutSecondsRequest
)

func MachineTypeOptions() []configuration.FieldOption {
	return machineTypeSelectOptions
}

func IntPtr(v int) *int { return intPtr(v) }

func AfterRunnerTaskCreated(ctx core.ExecutionContext, taskID string) error {
	return afterRunnerTaskCreated(ctx, taskID)
}

func PollBrokerTask(ctx core.ActionHookContext, finishedEventType string) error {
	return pollBrokerTask(ctx, finishedEventType)
}

func HandleBrokerWebhook(ctx core.WebhookRequestContext, finishedEventType string) (int, *core.WebhookResponseBody, error) {
	return handleBrokerWebhook(ctx, finishedEventType)
}

func CancelBrokerTask(ctx core.ExecutionContext) error {
	return cancelBrokerTask(ctx)
}

func ProcessBrokerTaskStatus(state core.ExecutionStateContext, task *Task, finishedEventType string) error {
	return processBrokerTaskStatus(state, task, finishedEventType)
}
