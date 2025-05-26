package models

import (
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

type EventSource struct {
	ID               uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	CanvasID         uuid.UUID
	Name             string
	Key              []byte
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
	LabelDefinitions datatypes.JSONSlice[LabelDefinition]
}

type LabelDefinition struct {
	Name      string  `json:"name"`
	ValueFrom *string `json:"value_from,omitempty"`
	Required  *bool   `json:"required,omitempty"`
}

func (s *EventSource) EvaluateLabels(event *Event) (map[string]string, error) {
	labels := map[string]string{}
	for _, labelDef := range s.LabelDefinitions {
		v, err := event.EvaluateStringExpression(*labelDef.ValueFrom)
		if err != nil {
			return nil, err
		}

		labels[labelDef.Name] = v
	}

	return labels, nil
}

func FindEventSource(id uuid.UUID) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("id = ?", id).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func (c *Canvas) ListEventSources() ([]EventSource, error) {
	var sources []EventSource
	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Find(&sources).
		Error

	if err != nil {
		return nil, err
	}

	return sources, nil
}
