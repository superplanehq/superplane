package runner

import (
	"encoding/json"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/runners"
)

const (
	ExecutionMetadataBrokerTaskID = "runner_broker_task_id"
	ExecutionMetadataTaskLog      = "runner_task_log"
)

// TaskLogSink matches the fleet-manager JSON shape for CloudWatch-backed live logs.
// Kept as an alias to runners.FleetTaskLog for public API compatibility.
type TaskLogSink = runners.FleetTaskLog

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

func taskLogFromFleetTask(t *runners.FleetTask) *TaskLogSink {
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

func taskLogFromRawWebhook(raw map[string]any) *TaskLogSink {
	if raw == nil {
		return nil
	}
	if v, ok := raw["task_log"]; ok && v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var sink TaskLogSink
		if err := json.Unmarshal(b, &sink); err != nil {
			return nil
		}
		if strings.TrimSpace(sink.Type) != "" {
			return &sink
		}
	}
	g, _ := raw["cloudwatch_log_group"].(string)
	s, _ := raw["cloudwatch_log_stream"].(string)
	t := &runners.FleetTask{
		CloudWatchLogGroup:  g,
		CloudWatchLogStream: s,
	}
	return taskLogFromFleetTask(t)
}
