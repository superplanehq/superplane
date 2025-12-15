package contexts

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

type ExecutionMetadataContext struct {
	execution *models.WorkflowNodeExecution
}

func NewExecutionMetadataContext(execution *models.WorkflowNodeExecution) *ExecutionMetadataContext {
	return &ExecutionMetadataContext{execution: execution}
}

func (m *ExecutionMetadataContext) Get() any {
	return m.execution.Metadata.Data()
}

func (m *ExecutionMetadataContext) Set(value any) {
	b, err := json.Marshal(value)
	if err != nil {
		return
	}

	var v map[string]any
	err = json.Unmarshal(b, &v)
	if err != nil {
		return
	}

	m.execution.Metadata = datatypes.NewJSONType(v)
}
