package models

import (
	"fmt"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Connection struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	TargetID       uuid.UUID
	TargetType     string
	SourceID       uuid.UUID
	SourceName     string
	SourceType     string
	Filters        datatypes.JSONSlice[Filter]
	FilterOperator string
}

func (c *Connection) Accept(event *Event) (bool, error) {
	if len(c.Filters) == 0 {
		return true, nil
	}

	switch c.FilterOperator {
	case FilterOperatorOr:
		return c.any(event)

	case FilterOperatorAnd:
		return c.all(event)

	default:
		return false, fmt.Errorf("invalid filter operator: %s", c.FilterOperator)
	}
}

func (c *Connection) all(event *Event) (bool, error) {
	for _, filter := range c.Filters {
		ok, err := filter.Evaluate(event)
		if err != nil {
			return false, fmt.Errorf("error evaluating filter: %v", err)
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

func (c *Connection) any(event *Event) (bool, error) {
	for _, filter := range c.Filters {
		ok, err := filter.Evaluate(event)
		if err != nil {
			return false, fmt.Errorf("error evaluating filter: %v", err)
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
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
