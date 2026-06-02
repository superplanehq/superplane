package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	CanvasMemorySourceNode   = "node"
	CanvasMemorySourceManual = "manual"
)

type CanvasMemory struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CreatedAt time.Time
	UpdatedAt time.Time
	CanvasID  uuid.UUID
	Namespace string
	Values    datatypes.JSONType[any]
	Source    string
}

func (CanvasMemory) TableName() string {
	return "canvas_memories"
}

func AddCanvasMemoryRecordInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string, values any) (CanvasMemory, error) {
	return AddCanvasMemoryRecordWithSourceInTransaction(tx, canvasID, namespace, values, CanvasMemorySourceNode)
}

func AddCanvasMemoryRecordWithSourceInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string, values any, source string) (CanvasMemory, error) {
	if source == "" {
		source = CanvasMemorySourceNode
	}

	record := CanvasMemory{
		CanvasID:  canvasID,
		Namespace: namespace,
		Values:    datatypes.NewJSONType(values),
		Source:    source,
	}

	err := tx.Create(&record).Error
	return record, err
}

func AddCanvasMemoryInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string, values any) error {
	_, err := AddCanvasMemoryRecordInTransaction(tx, canvasID, namespace, values)
	return err
}

func AddCanvasMemory(canvasID uuid.UUID, namespace string, values any) error {
	return AddCanvasMemoryInTransaction(database.Conn(), canvasID, namespace, values)
}

// CanvasMemoryNamespaceSourceInTransaction returns the source associated with
// existing rows in a namespace, or empty string if the namespace has no rows.
func CanvasMemoryNamespaceSourceInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string) (string, error) {
	var source string
	err := tx.
		Model(&CanvasMemory{}).
		Where("canvas_id = ? AND namespace = ?", canvasID, namespace).
		Limit(1).
		Pluck("source", &source).
		Error
	if err != nil {
		return "", err
	}

	return source, nil
}

func CanvasMemoryNamespaceSource(canvasID uuid.UUID, namespace string) (string, error) {
	return CanvasMemoryNamespaceSourceInTransaction(database.Conn(), canvasID, namespace)
}

// ReplaceManualCanvasMemoryNamespaceInTransaction replaces all manual-source
// rows in a namespace with one row per entry. Used by both create (no
// existing rows) and update (existing manual rows). The caller is
// responsible for enforcing origin-lock before invoking this.
func ReplaceManualCanvasMemoryNamespaceInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string, entries []any) error {
	err := tx.
		Where("canvas_id = ? AND namespace = ? AND source = ?", canvasID, namespace, CanvasMemorySourceManual).
		Delete(&CanvasMemory{}).
		Error
	if err != nil {
		return err
	}

	for _, entry := range entries {
		_, err := AddCanvasMemoryRecordWithSourceInTransaction(tx, canvasID, namespace, entry, CanvasMemorySourceManual)
		if err != nil {
			return err
		}
	}

	return nil
}

func ReplaceManualCanvasMemoryNamespace(canvasID uuid.UUID, namespace string, entries []any) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		return ReplaceManualCanvasMemoryNamespaceInTransaction(tx, canvasID, namespace, entries)
	})
}

func ListCanvasMemoriesInTransaction(tx *gorm.DB, canvasID uuid.UUID) ([]CanvasMemory, error) {
	var records []CanvasMemory
	err := tx.
		Where("canvas_id = ?", canvasID).
		Order("created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func ListCanvasMemories(canvasID uuid.UUID) ([]CanvasMemory, error) {
	return ListCanvasMemoriesInTransaction(database.Conn(), canvasID)
}

func ListCanvasMemoriesByNamespaceInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string) ([]CanvasMemory, error) {
	var records []CanvasMemory
	err := tx.
		Where("canvas_id = ? AND namespace = ?", canvasID, namespace).
		Order("created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}

	return records, nil
}

func ListCanvasMemoriesByNamespace(canvasID uuid.UUID, namespace string) ([]CanvasMemory, error) {
	return ListCanvasMemoriesByNamespaceInTransaction(database.Conn(), canvasID, namespace)
}

func ListCanvasMemoriesByNamespaceAndMatchesInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string, matches map[string]any) ([]CanvasMemory, error) {
	if len(matches) == 0 {
		return []CanvasMemory{}, fmt.Errorf("at least one match expression is required")
	}

	matchesJSON, err := json.Marshal(matches)
	if err != nil {
		return nil, err
	}

	var records []CanvasMemory

	err = tx.
		Where("canvas_id = ? AND namespace = ?", canvasID, namespace).
		Where("values @> ?::jsonb", matchesJSON).
		Order("created_at DESC").
		Find(&records).
		Error

	if err != nil {
		return nil, err
	}

	return records, nil
}

func ListCanvasMemoriesByNamespaceAndMatches(canvasID uuid.UUID, namespace string, matches map[string]any) ([]CanvasMemory, error) {
	return ListCanvasMemoriesByNamespaceAndMatchesInTransaction(database.Conn(), canvasID, namespace, matches)
}

func FindFirstCanvasMemoryByNamespaceAndMatchesInTransaction(tx *gorm.DB, canvasID uuid.UUID, namespace string, matches map[string]any) (*CanvasMemory, error) {
	if len(matches) == 0 {
		return nil, fmt.Errorf("at least one match expression is required")
	}

	matchesJSON, err := json.Marshal(matches)
	if err != nil {
		return nil, err
	}

	var record CanvasMemory

	err = tx.
		Where("canvas_id = ? AND namespace = ?", canvasID, namespace).
		Where("values @> ?::jsonb", matchesJSON).
		Order("created_at DESC").
		Limit(1).
		First(&record).
		Error

	if err != nil {
		return nil, nil
	}

	return &record, nil
}

func FindFirstCanvasMemoryByNamespaceAndMatches(canvasID uuid.UUID, namespace string, matches map[string]any) (*CanvasMemory, error) {
	return FindFirstCanvasMemoryByNamespaceAndMatchesInTransaction(database.Conn(), canvasID, namespace, matches)
}

func DeleteCanvasMemory(canvasID, memoryID uuid.UUID) error {
	return DeleteCanvasMemoryInTransaction(database.Conn(), canvasID, memoryID)
}

func DeleteCanvasMemoryInTransaction(tx *gorm.DB, canvasID, memoryID uuid.UUID) error {
	return tx.
		Where("canvas_id = ? AND id = ?", canvasID, memoryID).
		Delete(&CanvasMemory{}).
		Error
}

func DeleteCanvasMemoriesByNamespaceAndMatchesInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	namespace string,
	matches map[string]any,
) ([]CanvasMemory, error) {
	if len(matches) == 0 {
		return []CanvasMemory{}, fmt.Errorf("at least one match expression is required")
	}

	matchesJSON, err := json.Marshal(matches)
	if err != nil {
		return nil, err
	}

	var deletedRecords []CanvasMemory
	err = tx.Raw(
		`WITH deleted AS (
			DELETE FROM canvas_memories
			WHERE canvas_id = ? AND namespace = ? AND values @> ?::jsonb
			RETURNING *
		)
		SELECT * FROM deleted ORDER BY created_at DESC`,
		canvasID,
		namespace,
		matchesJSON,
	).Scan(&deletedRecords).Error
	if err != nil {
		return nil, err
	}

	return deletedRecords, nil
}

func DeleteCanvasMemoriesByNamespaceAndMatches(canvasID uuid.UUID, namespace string, matches map[string]any) ([]CanvasMemory, error) {
	return DeleteCanvasMemoriesByNamespaceAndMatchesInTransaction(database.Conn(), canvasID, namespace, matches)
}

func UpdateCanvasMemoriesByNamespaceAndMatchesInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	namespace string,
	matches map[string]any,
	values map[string]any,
) ([]CanvasMemory, error) {
	if len(matches) == 0 {
		return []CanvasMemory{}, fmt.Errorf("at least one match expression is required")
	}
	if len(values) == 0 {
		return []CanvasMemory{}, fmt.Errorf("at least one value expression is required")
	}

	matchesJSON, err := json.Marshal(matches)
	if err != nil {
		return nil, err
	}
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	var updatedRecords []CanvasMemory
	err = tx.Raw(
		`WITH updated AS (
			UPDATE canvas_memories
			SET values = values || ?::jsonb, updated_at = NOW()
			WHERE canvas_id = ? AND namespace = ? AND values @> ?::jsonb
			RETURNING *
		)
		SELECT * FROM updated ORDER BY created_at DESC`,
		valuesJSON,
		canvasID,
		namespace,
		matchesJSON,
	).Scan(&updatedRecords).Error
	if err != nil {
		return nil, err
	}

	return updatedRecords, nil
}

func UpdateCanvasMemoriesByNamespaceAndMatches(canvasID uuid.UUID, namespace string, matches map[string]any, values map[string]any) ([]CanvasMemory, error) {
	return UpdateCanvasMemoriesByNamespaceAndMatchesInTransaction(database.Conn(), canvasID, namespace, matches, values)
}

func UpdateCanvasMemoriesByNamespaceInTransaction(
	tx *gorm.DB,
	canvasID uuid.UUID,
	namespace string,
	values map[string]any,
) ([]CanvasMemory, error) {
	if len(values) == 0 {
		return []CanvasMemory{}, fmt.Errorf("at least one value expression is required")
	}

	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	var updatedRecords []CanvasMemory
	err = tx.Raw(
		`WITH updated AS (
			UPDATE canvas_memories
			SET values = values || ?::jsonb, updated_at = NOW()
			WHERE canvas_id = ? AND namespace = ?
			RETURNING *
		)
		SELECT * FROM updated ORDER BY created_at DESC`,
		valuesJSON,
		canvasID,
		namespace,
	).Scan(&updatedRecords).Error
	if err != nil {
		return nil, err
	}

	return updatedRecords, nil
}

func UpdateCanvasMemoriesByNamespace(canvasID uuid.UUID, namespace string, values map[string]any) ([]CanvasMemory, error) {
	return UpdateCanvasMemoriesByNamespaceInTransaction(database.Conn(), canvasID, namespace, values)
}
