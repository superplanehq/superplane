package contexts

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

type NodeMetadataContext struct {
	node *models.WorkflowNode
}

func NewNodeMetadataContext(node *models.WorkflowNode) core.MetadataContext {
	return &NodeMetadataContext{node: node}
}

func (m *NodeMetadataContext) Get() any {
	return m.node.Metadata.Data()
}

func (m *NodeMetadataContext) Set(value any) {
	b, err := json.Marshal(value)
	if err != nil {
		return
	}

	var v map[string]any
	err = json.Unmarshal(b, &v)
	if err != nil {
		return
	}

	m.node.Metadata = datatypes.NewJSONType(v)
}
