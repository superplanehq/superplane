package contexts

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ExecutionMetadataContext struct {
	tx        *gorm.DB
	execution *models.WorkflowNodeExecution
}

func NewExecutionMetadataContext(tx *gorm.DB, execution *models.WorkflowNodeExecution) *ExecutionMetadataContext {
	return &ExecutionMetadataContext{tx: tx, execution: execution}
}

func (m *ExecutionMetadataContext) Get() any {
	return m.execution.Metadata.Data()
}

func (m *ExecutionMetadataContext) Set(value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var v map[string]any
	err = json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	m.execution.Metadata = datatypes.NewJSONType(v)
	return m.tx.Model(m.execution).
		Update("metadata", v).
		Error
}
