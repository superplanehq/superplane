package runner

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	executionKVMachineType = "machine_type"
	executionKVFleetID     = "fleet_id"
)

func storeRunnerFleetKV(ctx core.ExecutionContext, machineType string) error {
	machineType = strings.TrimSpace(machineType)
	if machineType == "" {
		machineType = configurationMachineType(ctx.Configuration)
	}
	if err := ctx.ExecutionState.SetKV(executionKVMachineType, machineType); err != nil {
		return err
	}

	fleetID, err := resolveBrokerFleetID(machineType)
	if err != nil {
		fleetID = machineType
	}
	return ctx.ExecutionState.SetKV(executionKVFleetID, fleetID)
}

func RecordRunnerComputeUsage(
	usage core.UsageRecorder,
	logger *log.Entry,
	state core.ExecutionStateContext,
	configuration any,
	task *Task,
) {
	if usage == nil || task == nil {
		return
	}
	if task.ClaimedAt == nil || task.FinishedAt == nil {
		if logger != nil {
			logger.Warn("runner: skip compute usage, claimed_at or finished_at is missing")
		}
		return
	}

	seconds := billableSeconds(task.FinishedAt.Sub(*task.ClaimedAt))
	machineType := resolveComputeMachineType(state, configuration)
	fleetID := resolveComputeFleetID(state, machineType)
	taskID := task.brokerTaskID()

	record := core.ComputeUsageRecord{
		MachineType:     machineType,
		FleetID:         fleetID,
		DurationSeconds: seconds,
		IdempotencyKey:  models.UsageIdempotencyKeyRunner + ":compute:" + taskID,
	}
	if err := usage.RecordCompute(record); err != nil && logger != nil {
		logger.WithError(err).Error("failed to record runner compute usage")
	}
}

func resolveComputeMachineType(state core.ExecutionStateContext, configuration any) string {
	machineType := executionKV(state, executionKVMachineType)
	if machineType == "" {
		machineType = configurationMachineType(configuration)
	}
	if machineType == "" {
		return "unknown"
	}
	return machineType
}

func resolveComputeFleetID(state core.ExecutionStateContext, machineType string) string {
	if fleetID := executionKV(state, executionKVFleetID); fleetID != "" {
		return fleetID
	}
	fleetID, err := resolveBrokerFleetID(machineType)
	if err != nil {
		return ""
	}
	return fleetID
}

func executionKV(state core.ExecutionStateContext, key string) string {
	if state == nil {
		return ""
	}
	value, err := state.GetKV(key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func configurationMachineType(configuration any) string {
	if v := configurationString(configuration, "machine_type"); v != "" {
		return v
	}
	return configurationString(configuration, "machineType")
}
