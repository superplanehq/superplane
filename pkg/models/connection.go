package models

import (
	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Connection struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	CanvasID       uuid.UUID
	TargetID       uuid.UUID
	TargetType     string
	SourceID       uuid.UUID
	SourceName     string
	SourceType     string
	Filters        datatypes.JSONSlice[Filter]
	FilterOperator string
}

func (c *Connection) Accept(event *Event) (bool, error) {
	return ApplyFilters(c.Filters, c.FilterOperator, event)
}

func ListConnectionsForSource(sourceID uuid.UUID, connectionType string) ([]Connection, error) {
	var connections []Connection
	err := database.Conn().
		Where("source_id = ?", sourceID).
		Where("source_type = ?", connectionType).
		Find(&connections).
		Error

	if err != nil {
		return nil, err
	}

	return connections, nil
}

func ListConnectionsForTarget(targetID uuid.UUID, targetType string) ([]Connection, error) {
	var connections []Connection
	err := database.Conn().
		Where("target_id = ?", targetID).
		Where("target_type = ?", targetType).
		Find(&connections).
		Error

	if err != nil {
		return nil, err
	}

	return connections, nil
}

func FindConnection(targetID uuid.UUID, targetType string, sourceName string) (*Connection, error) {
	var connection Connection
	err := database.Conn().
		Where("target_id = ?", targetID).
		Where("target_type = ?", targetType).
		Where("source_name = ?", sourceName).
		First(&connection).
		Error

	if err != nil {
		return nil, err
	}

	return &connection, nil
}

func ListConnections(targetID uuid.UUID, targetType string) ([]Connection, error) {
	return ListConnectionsInTransaction(database.Conn(), targetID, targetType)
}

func ListConnectionsInTransaction(tx *gorm.DB, targetID uuid.UUID, targetType string) ([]Connection, error) {
	var connections []Connection
	err := tx.
		Where("target_id = ?", targetID).
		Where("target_type = ?", targetType).
		Find(&connections).
		Error

	if err != nil {
		return nil, err
	}

	return connections, nil
}

type StageNotifier interface {
	NotifyStageUpdated(stage *Stage)
}

func UpdateConnectionSourceNameInTransaction(tx *gorm.DB, canvasID uuid.UUID, sourceID uuid.UUID, sourceType string, oldName string, newName string, notifier StageNotifier) error {
	// Update connection source names
	if err := tx.
		Model(&Connection{}).
		Where("source_id = ?", sourceID).
		Where("source_type = ?", sourceType).
		Where("source_name = ?", oldName).
		Update("source_name", newName).
		Error; err != nil {
		return err
	}

	var stages []Stage
	if err := tx.Where("canvas_id = ?", canvasID).
		Where(`EXISTS (
			SELECT 1 FROM jsonb_array_elements(input_mappings) AS mapping
			WHERE mapping->'when'->'triggered_by'->>'connection' = ?
		)`, oldName).
		Find(&stages).Error; err != nil {
		return err
	}

	updatedStages := updateStageConnectionReferences(stages, oldName, newName)

	for _, stage := range updatedStages {
		if err := tx.Save(&stage).Error; err != nil {
			log.Errorf("Failed to update stage input mappings: %v", err)
			return err
		}

		if notifier != nil {
			log.Infof("Stage updated due to stage name change: %s", stage.ID)
			notifier.NotifyStageUpdated(&stage)
		}
	}

	return nil
}

func updateStageConnectionReferences(stages []Stage, oldName, newName string) []Stage {
	var updatedStages []Stage

	for _, stage := range stages {
		updated := false

		for i := range stage.InputMappings {
			mapping := &stage.InputMappings[i]

			if mapping.When != nil && mapping.When.TriggeredBy != nil && mapping.When.TriggeredBy.Connection == oldName {
				mapping.When.TriggeredBy.Connection = newName
				updated = true
			}

			for j := range mapping.Values {
				value := &mapping.Values[j]
				if value.ValueFrom != nil && value.ValueFrom.EventData != nil && value.ValueFrom.EventData.Connection == oldName {
					value.ValueFrom.EventData.Connection = newName
					updated = true
				}
			}
		}

		if updated {
			updatedStages = append(updatedStages, stage)
		}
	}

	return updatedStages
}
