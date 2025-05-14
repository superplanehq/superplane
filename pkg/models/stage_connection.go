package models

import (
	"fmt"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

const (
	FilterTypeData    = "data"
	FilterTypeHeader  = "header"
	FilterOperatorAnd = "and"
	FilterOperatorOr  = "or"
)

type StageConnection struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	StageID        uuid.UUID
	SourceID       uuid.UUID
	SourceName     string
	SourceType     string
	Filters        datatypes.JSONSlice[StageConnectionFilter]
	FilterOperator string
}

func (c *StageConnection) Accept(event *Event) (bool, error) {
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

func (c *StageConnection) all(event *Event) (bool, error) {
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

func (c *StageConnection) any(event *Event) (bool, error) {
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

type StageConnectionFilter struct {
	Type   string
	Data   *DataFilter
	Header *HeaderFilter
}

func (f *StageConnectionFilter) EvaluateExpression(event *Event) (bool, error) {
	switch f.Type {
	case FilterTypeData:
		return event.EvaluateBoolExpression(f.Data.Expression, FilterTypeData)
	case FilterTypeHeader:
		return event.EvaluateBoolExpression(f.Header.Expression, FilterTypeHeader)
	default:
		return false, fmt.Errorf("invalid filter type: %s", f.Type)
	}
}

func (f *StageConnectionFilter) Evaluate(event *Event) (bool, error) {
	switch f.Type {
	case FilterTypeData:
		return f.EvaluateExpression(event)
	case FilterTypeHeader:
		return f.EvaluateExpression(event)

	default:
		return false, fmt.Errorf("invalid filter type: %s", f.Type)
	}
}

type DataFilter struct {
	Expression string
}

type HeaderFilter struct {
	Expression string
}

func ListConnectionsForSource(sourceID uuid.UUID, connectionType string) ([]StageConnection, error) {
	var connections []StageConnection
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

func FindStageConnection(stageID uuid.UUID, sourceName string) (*StageConnection, error) {
	var connection StageConnection
	err := database.Conn().
		Where("stage_id = ?", stageID).
		Where("source_name = ?", sourceName).
		First(&connection).
		Error

	if err != nil {
		return nil, err
	}

	return &connection, nil
}

func ListConnectionsForStage(stageID uuid.UUID) ([]StageConnection, error) {
	var connections []StageConnection
	err := database.Conn().
		Where("stage_id = ?", stageID).
		Find(&connections).
		Error

	if err != nil {
		return nil, err
	}

	return connections, nil
}
