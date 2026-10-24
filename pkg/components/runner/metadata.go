package runner

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ExecutionMetadataBrokerTaskID = "runner_broker_task_id"
	ExecutionMetadataTaskBackend  = "runner_task_backend"
)

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

func mergeRunnerTaskBackend(meta core.MetadataWriter, backend string) error {
	backend = normalizeTaskBackend(backend)
	return mergeExecutionMetadata(meta, map[string]any{
		ExecutionMetadataTaskBackend: backend,
	})
}

func TaskBackendFromExecutionMetadata(meta map[string]any) string {
	if meta == nil {
		return core.RunnerTaskBackendLegacy
	}
	return normalizeTaskBackend(fmt.Sprint(meta[ExecutionMetadataTaskBackend]))
}

func normalizeTaskBackend(backend string) string {
	if strings.TrimSpace(backend) == core.RunnerTaskBackendIntegrated {
		return core.RunnerTaskBackendIntegrated
	}
	return core.RunnerTaskBackendLegacy
}
