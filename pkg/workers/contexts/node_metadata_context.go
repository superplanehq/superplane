package contexts

import (
	"encoding/json"

	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/*
 * Implementation of core.MetadataContext for nodes that are part of a live canvas.
 */
type NodeMetadataContext struct {
	tx   *gorm.DB
	node *models.CanvasNode
}

func NewNodeMetadataContext(tx *gorm.DB, node *models.CanvasNode) *NodeMetadataContext {
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

/*
 * Implementation of core.MetadataContext for nodes that are not yet part of a live canvas.
 * Nothing is persisted, so all write operations are no-ops.
 */
type ReadOnlyNodeMetadataContext struct {
	Metadata any
}

func NewReadOnlyNodeMetadataContext(metadata any) *ReadOnlyNodeMetadataContext {
	return &ReadOnlyNodeMetadataContext{Metadata: metadata}
}

func (m *ReadOnlyNodeMetadataContext) Get() any {
	return m.Metadata
}

func (m *ReadOnlyNodeMetadataContext) Set(value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var v map[string]any
	err = json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	m.Metadata = v
	return nil
}
