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

func (c *CanvasMemoryContext) Find(namespace string, matches map[string]any) ([]any, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	records, err := models.ListCanvasMemoriesByNamespaceAndMatchesInTransaction(c.tx, c.canvasID, namespace, matches)
	if err != nil {
		return nil, err
	}

	values := make([]any, 0, len(records))
	for _, record := range records {
		values = append(values, record.Values.Data())
	}

	return values, nil
}

func (c *CanvasMemoryContext) FindFirst(namespace string, matches map[string]any) (any, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	record, err := models.FindFirstCanvasMemoryByNamespaceAndMatchesInTransaction(c.tx, c.canvasID, namespace, matches)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}

	return record.Values.Data(), nil
}

func (c *CanvasMemoryContext) Delete(namespace string, matches map[string]any) ([]any, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	records, err := models.DeleteCanvasMemoriesByNamespaceAndMatchesInTransaction(c.tx, c.canvasID, namespace, matches)
	if err != nil {
		return nil, err
	}

	deletedValues := make([]any, 0, len(records))
	for _, record := range records {
		deletedValues = append(deletedValues, record.Values.Data())
	}

	return deletedValues, nil
}

func (c *CanvasMemoryContext) Update(namespace string, matches map[string]any, values map[string]any) ([]any, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	records, err := models.UpdateCanvasMemoriesByNamespaceAndMatchesInTransaction(c.tx, c.canvasID, namespace, matches, values)
	if err != nil {
		return nil, err
	}

	updatedValues := make([]any, 0, len(records))
	for _, record := range records {
		updatedValues = append(updatedValues, record.Values.Data())
	}

	return updatedValues, nil
}
