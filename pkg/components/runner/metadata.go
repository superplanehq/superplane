package runner

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	runnermodels "github.com/superplanehq/superplane/pkg/runners/models"
)

const (
	ExecutionMetadataBrokerTaskID = "runner_broker_task_id"
	ExecutionMetadataTaskLog      = "runner_task_log"
)

// TaskLogSink matches the fleet-manager JSON shape for CloudWatch-backed live logs.
type TaskLogSink = runnermodels.FleetTaskLog

func mergeExecutionMetadata(meta core.MetadataWriter, patch map[string]any) error {
	if meta == nil {
		return nil
	}
	cur := meta.Get()
	m, ok := cur.(map[string]any)
	if !ok || m == nil {
		m = map[string]any{}
	}
	for k, v := range patch {
		if v == nil {
			delete(m, k)
			continue
		}
		m[k] = v
	}
	return meta.Set(m)
}

func mergeRunnerBrokerTaskID(meta core.MetadataWriter, brokerTaskID string) error {
	brokerTaskID = strings.TrimSpace(brokerTaskID)
	if brokerTaskID == "" {
		return nil
	}
	return mergeExecutionMetadata(meta, map[string]any{
		ExecutionMetadataBrokerTaskID: brokerTaskID,
	})
}

func mergeRunnerTaskLog(meta core.MetadataWriter, brokerTaskID string, sink *TaskLogSink) error {
	brokerTaskID = strings.TrimSpace(brokerTaskID)
	patch := map[string]any{}
	if brokerTaskID != "" {
		patch[ExecutionMetadataBrokerTaskID] = brokerTaskID
	}
	if sink != nil && strings.TrimSpace(sink.Type) != "" {
		patch[ExecutionMetadataTaskLog] = sink
	}
	if len(patch) == 0 {
		return nil
	}
	return mergeExecutionMetadata(meta, patch)
}

// FinishFleetTask merges task log metadata and emits the terminal runner event.
func FinishFleetTask(meta core.MetadataWriter, state core.ExecutionStateContext, fleetTask *runnermodels.FleetTask, brokerTaskID string) error {
	sink := taskLogFromFleetTask(fleetTask)
	if err := mergeRunnerTaskLog(meta, brokerTaskID, sink); err != nil {
		return err
	}
	return (&Runner{}).processTaskStatus(state, fleetTask)
}

func taskLogFromFleetTask(t *runnermodels.FleetTask) *TaskLogSink {
	if t == nil {
		return nil
	}
	if t.TaskLog != nil && strings.TrimSpace(t.TaskLog.Type) != "" {
		return t.TaskLog
	}
	g := strings.TrimSpace(t.CloudWatchLogGroup)
	s := strings.TrimSpace(t.CloudWatchLogStream)
	if g == "" || s == "" {
		return nil
	}
	return &TaskLogSink{
		Type: "cloudwatch",
		CloudWatch: &struct {
			LogGroupName  string `json:"log_group_name"`
			LogStreamName string `json:"log_stream_name"`
			Region        string `json:"region,omitempty"`
		}{
			LogGroupName:  g,
			LogStreamName: s,
		},
	}
}
