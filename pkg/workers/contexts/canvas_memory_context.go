package contexts

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type CanvasMemoryContext struct {
	tx        *gorm.DB
	canvasID  uuid.UUID
	onChanged func()
}

func NewCanvasMemoryContext(tx *gorm.DB, canvasID uuid.UUID) *CanvasMemoryContext {
	return &CanvasMemoryContext{tx: tx, canvasID: canvasID}
}

// WithChangeCallback registers a callback invoked whenever a mutating operation
// (Add/Delete/Update/UpdateNamespace) succeeds. Used to signal callers that
// memory was modified so they can publish a websocket event post-commit.
func (c *CanvasMemoryContext) WithChangeCallback(onChanged func()) *CanvasMemoryContext {
	c.onChanged = onChanged
	return c
}

func (c *CanvasMemoryContext) notifyChanged() {
	if c.onChanged != nil {
		c.onChanged()
	}
}

func (c *CanvasMemoryContext) Add(namespace string, values any) error {
	_, err := c.AddRecord(namespace, values)
	return err
}

func (c *CanvasMemoryContext) AddRecord(namespace string, values any) (core.CanvasMemoryRecord, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return core.CanvasMemoryRecord{}, fmt.Errorf("namespace is required")
	}

	if err := c.ensureNodeWritable(namespace); err != nil {
		return core.CanvasMemoryRecord{}, err
	}

	record, err := models.AddCanvasMemoryRecordInTransaction(c.tx, c.canvasID, namespace, values)
	if err != nil {
		return core.CanvasMemoryRecord{}, err
	}

	c.notifyChanged()
	return canvasMemoryRecord(record), nil
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

	if err := c.ensureNodeWritable(namespace); err != nil {
		return nil, err
	}

	records, err := models.DeleteCanvasMemoriesByNamespaceAndMatchesInTransaction(c.tx, c.canvasID, namespace, matches)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		c.notifyChanged()
	}

	deletedValues := make([]any, 0, len(records))
	for _, record := range records {
		deletedValues = append(deletedValues, record.Values.Data())
	}

	return deletedValues, nil
}

func (c *CanvasMemoryContext) Update(namespace string, matches map[string]any, values map[string]any) ([]any, error) {
	records, err := c.UpdateRecords(namespace, matches, values)
	if err != nil {
		return nil, err
	}

	return canvasMemoryRecordValues(records), nil
}

func (c *CanvasMemoryContext) UpdateRecords(namespace string, matches map[string]any, values map[string]any) ([]core.CanvasMemoryRecord, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if err := c.ensureNodeWritable(namespace); err != nil {
		return nil, err
	}

	records, err := models.UpdateCanvasMemoriesByNamespaceAndMatchesInTransaction(c.tx, c.canvasID, namespace, matches, values)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		c.notifyChanged()
	}

	return canvasMemoryRecords(records), nil
}

func (c *CanvasMemoryContext) UpdateNamespace(namespace string, values map[string]any) ([]any, error) {
	records, err := c.UpdateNamespaceRecords(namespace, values)
	if err != nil {
		return nil, err
	}

	return canvasMemoryRecordValues(records), nil
}

func (c *CanvasMemoryContext) UpdateNamespaceRecords(namespace string, values map[string]any) ([]core.CanvasMemoryRecord, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if err := c.ensureNodeWritable(namespace); err != nil {
		return nil, err
	}

	records, err := models.UpdateCanvasMemoriesByNamespaceInTransaction(c.tx, c.canvasID, namespace, values)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		c.notifyChanged()
	}

	return canvasMemoryRecords(records), nil
}

// ensureNodeWritable returns an error if the namespace is currently owned by
// manually-created memory, preventing node executions from mutating it.
func (c *CanvasMemoryContext) ensureNodeWritable(namespace string) error {
	source, err := models.CanvasMemoryNamespaceSourceInTransaction(c.tx, c.canvasID, namespace)
	if err != nil {
		return fmt.Errorf("failed to check memory namespace ownership: %w", err)
	}

	if source == models.CanvasMemorySourceManual {
		return fmt.Errorf("cannot modify manually-managed memory namespace %q", namespace)
	}

	return nil
}

func canvasMemoryRecord(record models.CanvasMemory) core.CanvasMemoryRecord {
	return core.CanvasMemoryRecord{
		ID:     record.ID,
		Values: record.Values.Data(),
	}
}

func canvasMemoryRecords(records []models.CanvasMemory) []core.CanvasMemoryRecord {
	out := make([]core.CanvasMemoryRecord, 0, len(records))
	for _, record := range records {
		out = append(out, canvasMemoryRecord(record))
	}
	return out
}

func canvasMemoryRecordValues(records []core.CanvasMemoryRecord) []any {
	values := make([]any, 0, len(records))
	for _, record := range records {
		values = append(values, record.Values)
	}
	return values
}
