package models

import (
	"fmt"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNameAlreadyUsed = fmt.Errorf("name already used")

type Canvas struct {
	ID        uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name      string
	CreatedAt *time.Time
	CreatedBy uuid.UUID
	UpdatedAt *time.Time
}

func (Canvas) TableName() string {
	return "canvases"
}

// NOTE: caller must encrypt the key before calling this method.
func (c *Canvas) CreateEventSource(name string, key []byte) (*EventSource, error) {
	now := time.Now()

	eventSource := EventSource{
		Name:      name,
		CanvasID:  c.ID,
		CreatedAt: &now,
		UpdatedAt: &now,
		Key:       key,
	}

	err := database.Conn().
		Clauses(clause.Returning{}).
		Create(&eventSource).
		Error

	if err == nil {
		return &eventSource, nil
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil, ErrNameAlreadyUsed
	}

	return nil, err
}

func (c *Canvas) FindEventSourceByName(name string) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Where("name = ?", name).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func (c *Canvas) FindStageByName(name string) (*Stage, error) {
	var stage Stage

	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Where("name = ?", name).
		First(&stage).
		Error

	if err != nil {
		return nil, err
	}

	return &stage, nil
}

// NOTE: the caller must decrypt the key before using it
func (c *Canvas) FindEventSourceByID(id uuid.UUID) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("id = ?", id).
		Where("canvas_id = ?", c.ID).
		First(&eventSource).
		Error

	if err != nil {
		return nil, err
	}

	return &eventSource, nil
}

func (c *Canvas) FindStageByID(id string) (*Stage, error) {
	var stage Stage

	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Where("id = ?", id).
		First(&stage).
		Error

	if err != nil {
		return nil, err
	}

	return &stage, nil
}

func (c *Canvas) ListStages() ([]Stage, error) {
	var stages []Stage

	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Order("name ASC").
		Find(&stages).
		Error

	if err != nil {
		return nil, err
	}

	return stages, nil
}

func (c *Canvas) CreateStage(name, createdBy string, conditions []StageCondition, template RunTemplate, connections []StageConnection) error {
	now := time.Now()
	ID := uuid.New()

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		stage := &Stage{
			ID:          ID,
			CanvasID:    c.ID,
			Name:        name,
			Conditions:  datatypes.NewJSONSlice(conditions),
			CreatedAt:   &now,
			CreatedBy:   uuid.Must(uuid.Parse(createdBy)),
			RunTemplate: datatypes.NewJSONType(template),
		}

		err := tx.Clauses(clause.Returning{}).Create(&stage).Error
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				return ErrNameAlreadyUsed
			}

			return err
		}

		for _, i := range connections {
			c := i
			c.StageID = ID
			err := tx.Create(&c).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (c *Canvas) UpdateStage(id, requesterID string, conditions []StageCondition, template RunTemplate, connections []StageConnection) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("stage_id = ?", id).Delete(&StageConnection{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing connections: %v", err)
		}

		for _, connection := range connections {
			connection.StageID = uuid.Must(uuid.Parse(id))
			if err := tx.Create(&connection).Error; err != nil {
				return fmt.Errorf("failed to create connection: %v", err)
			}
		}

		now := time.Now()
		err := tx.Model(&Stage{}).
			Where("id = ?", id).
			Update("updated_at", now).
			Update("updated_by", requesterID).
			Update("run_template", datatypes.NewJSONType(template)).
			Update("conditions", datatypes.NewJSONSlice(conditions)).
			Error

		if err != nil {
			return fmt.Errorf("failed to update stage timestamp: %v", err)
		}

		return nil
	})
}

func FindCanvas(id string) (*Canvas, error) {
	canvas := Canvas{}

	err := database.Conn().
		Where("id = ?", id).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func CreateCanvas(requesterID uuid.UUID, name string) (*Canvas, error) {
	now := time.Now()
	canvas := Canvas{
		Name:      name,
		CreatedAt: &now,
		CreatedBy: requesterID,
		UpdatedAt: &now,
	}

	err := database.Conn().
		Clauses(clause.Returning{}).
		Create(&canvas).
		Error

	if err == nil {
		return &canvas, nil
	}

	if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return nil, ErrNameAlreadyUsed
	}

	return nil, err
}
