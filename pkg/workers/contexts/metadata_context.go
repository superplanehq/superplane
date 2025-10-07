package contexts

import (
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/primitives"
	"gorm.io/datatypes"
)

type MetadataContext struct {
	execution *models.WorkflowNodeExecution
}

func (m *MetadataContext) Get(key string) (any, bool) {
	data := m.execution.Metadata.Data()
	if data == nil {
		return nil, false
	}
	val, ok := data[key]
	return val, ok
}

func (m *MetadataContext) Set(key string, value any) {
	data := m.execution.Metadata.Data()
	if data == nil {
		data = make(map[string]any)
	}
	data[key] = value
	m.execution.Metadata = datatypes.NewJSONType(data)
}

func (m *MetadataContext) GetAll() map[string]any {
	return m.execution.Metadata.Data()
}

// NewMetadataContext creates a MetadataContext for a workflow node execution
func NewMetadataContext(execution *models.WorkflowNodeExecution) primitives.MetadataContext {
	return &MetadataContext{execution: execution}
}
