package contexts

import (
	"fmt"
	"math"
	"reflect"
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

	records, err := models.ListCanvasMemoriesByNamespaceInTransaction(c.tx, c.canvasID, namespace)
	if err != nil {
		return nil, err
	}

	values := make([]any, 0, len(records))
	for _, record := range records {
		value := record.Values.Data()
		if !canvasMemoryValueMatches(value, matches) {
			continue
		}
		values = append(values, value)
	}

	return values, nil
}

func (c *CanvasMemoryContext) FindFirst(namespace string, matches map[string]any) (any, error) {
	values, err := c.Find(namespace, matches)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}

	return values[0], nil
}

func canvasMemoryValueMatches(value any, matches map[string]any) bool {
	if len(matches) == 0 {
		return true
	}

	valueMap, ok := value.(map[string]any)
	if !ok {
		return false
	}

	for key, expected := range matches {
		actual, exists := valueMap[key]
		if !exists {
			return false
		}
		if !canvasMemoryValuesEqual(actual, expected) {
			return false
		}
	}

	return true
}

func canvasMemoryValuesEqual(actual, expected any) bool {
	if actualNumber, ok := toCanvasMemoryComparableFloat(actual); ok {
		expectedNumber, expectedOk := toCanvasMemoryComparableFloat(expected)
		if !expectedOk {
			return false
		}
		return math.Abs(actualNumber-expectedNumber) < 1e-9
	}

	return reflect.DeepEqual(actual, expected)
}

func toCanvasMemoryComparableFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}
