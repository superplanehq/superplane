package runner

import (
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
)

// Shared symbols for runner subpackages (e.g. runner/claude).

const (
	MachineTypeFieldName              = configurationFieldMachineType
	HookPoll                          = hookActionPoll
	MaxExecutionTimeoutSecondsRequest = maxExecutionTimeoutSecondsRequest
)

func IntPtr(v int) *int { return intPtr(v) }

func AfterRunnerTaskCreated(
	ctx core.ExecutionContext,
	taskID string,
) error {
	return afterRunnerTaskCreated(ctx, taskID)
}

func PollTask(ctx core.ActionHookContext, finishedEventType string) error {
	return pollTask(ctx, finishedEventType)
}

func CancelTask(ctx core.ExecutionContext, finishedEventType string) error {
	return cancelTask(ctx, finishedEventType)
}

func RevokeOpenRouterChildKey(httpCtx core.HTTPContext, state core.ExecutionStateContext, hosted core.HostedLLMContext, logger *logrus.Entry) {
	_ = revokeOpenRouterChildKey(httpCtx, state, hosted, logger)
}

func ProcessTaskStatus(state core.ExecutionStateContext, task *core.RunnerTask, finishedEventType, organizationID string) error {
	return processTaskStatus(state, task, finishedEventType, organizationID, nil, nil, nil)
}
