package contexts

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type NodeMetadataContext struct {
	tx   *gorm.DB
	node *models.WorkflowNode
}

func NewNodeMetadataContext(tx *gorm.DB, node *models.WorkflowNode) *NodeMetadataContext {
	return &NodeMetadataContext{tx: tx, node: node}
}

func (m *NodeMetadataContext) Get() any {
	return m.node.Metadata.Data()
}

func (m *NodeMetadataContext) Set(value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var v map[string]any
	err = json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	m.node.Metadata = datatypes.NewJSONType(v)
	return m.tx.
		Model(m.node).
		Update("metadata", v).
		Error
}

func (m *NodeMetadataContext) UpdateConfiguration(configMap map[string]any) error {
	m.node.Configuration = datatypes.NewJSONType(configMap)
	return m.tx.Model(m.node).Update("configuration", configMap).Error
}
