package contexts

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

type MetadataContext struct {
	execution *models.WorkflowNodeExecution
}

func NewMetadataContext(execution *models.WorkflowNodeExecution) components.MetadataContext {
	return &MetadataContext{execution: execution}
}

func (m *MetadataContext) Get() any {
	return m.execution.Metadata.Data()
}

func (m *MetadataContext) Set(value any) {
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
