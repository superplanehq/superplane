package contexts

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type CanvasMemoryContext struct {
	tx       *gorm.DB
	canvasID uuid.UUID
}

func NewCanvasMemoryContext(tx *gorm.DB, canvasID uuid.UUID) *CanvasMemoryContext {
	return &CanvasMemoryContext{tx: tx, canvasID: canvasID}
}

func (c *CanvasMemoryContext) Add(namespace string, values any) error {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	return models.AddCanvasMemoryInTransaction(c.tx, c.canvasID, namespace, values)
}
