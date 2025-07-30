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
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name           string
	Description    string
	CreatedAt      *time.Time
	CreatedBy      uuid.UUID
	UpdatedAt      *time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	OrganizationID uuid.UUID

	Organization *Organization `gorm:"foreignKey:OrganizationID;references:ID"`
}

func (Canvas) TableName() string {
	return "canvases"
}

func (c *Canvas) CreateEventSource(name string, description string, key []byte, scope string, resourceId *uuid.UUID) (*EventSource, error) {
	return c.CreateEventSourceInTransaction(database.Conn(), name, description, key, scope, resourceId)
}

// NOTE: caller must encrypt the key before calling this method.
func (c *Canvas) CreateEventSourceInTransaction(tx *gorm.DB, name, description string, key []byte, scope string, resourceId *uuid.UUID) (*EventSource, error) {
	now := time.Now()

	eventSource := EventSource{
		Name:        name,
		Description: description,
		CanvasID:    c.ID,
		CreatedAt:   &now,
		UpdatedAt:   &now,
		Key:         key,
		ResourceID:  resourceId,
		State:       EventSourceStatePending,
		Scope:       scope,
	}

	err := tx.
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
		Where("scope = ?", EventSourceScopeExternal).
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

func (c *Canvas) FindConnectionGroupByName(name string) (*ConnectionGroup, error) {
	var connectionGroup ConnectionGroup

	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Where("name = ?", name).
		First(&connectionGroup).
		Error

	if err != nil {
		return nil, err
	}

	return &connectionGroup, nil
}

func (c *Canvas) FindConnectionGroupByID(id uuid.UUID) (*ConnectionGroup, error) {
	var connectionGroup ConnectionGroup

	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Where("id = ?", id).
		First(&connectionGroup).
		Error

	if err != nil {
		return nil, err
	}

	return &connectionGroup, nil
}

// NOTE: the caller must decrypt the key before using it
func (c *Canvas) FindEventSourceByID(id uuid.UUID) (*EventSource, error) {
	var eventSource EventSource
	err := database.Conn().
		Where("id = ?", id).
		Where("canvas_id = ?", c.ID).
		Where("scope = ?", EventSourceScopeExternal).
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

func (c *Canvas) ListConnectionGroups() ([]ConnectionGroup, error) {
	var connectionGroups []ConnectionGroup

	err := database.Conn().
		Where("canvas_id = ?", c.ID).
		Order("name ASC").
		Find(&connectionGroups).
		Error

	if err != nil {
		return nil, err
	}

	return connectionGroups, nil
}

func (c *Canvas) CreateStage(
	name, createdBy string,
	conditions []StageCondition,
	inputs []InputDefinition,
	inputMappings []InputMapping,
	outputs []OutputDefinition,
	secrets []ValueDefinition,
) (*Stage, error) {
	return c.CreateStageInTransaction(
		database.Conn(),
		name,
		createdBy,
		conditions,
		inputs,
		inputMappings,
		outputs,
		secrets,
	)
}

func (c *Canvas) CreateStageInTransaction(
	tx *gorm.DB,
	name, createdBy string,
	conditions []StageCondition,
	inputs []InputDefinition,
	inputMappings []InputMapping,
	outputs []OutputDefinition,
	secrets []ValueDefinition,
) (*Stage, error) {
	now := time.Now()

	stage := &Stage{
		CanvasID:      c.ID,
		Name:          name,
		Conditions:    datatypes.NewJSONSlice(conditions),
		CreatedAt:     &now,
		CreatedBy:     uuid.Must(uuid.Parse(createdBy)),
		Inputs:        datatypes.NewJSONSlice(inputs),
		InputMappings: datatypes.NewJSONSlice(inputMappings),
		Outputs:       datatypes.NewJSONSlice(outputs),
		Secrets:       datatypes.NewJSONSlice(secrets),
	}

	err := tx.Clauses(clause.Returning{}).Create(&stage).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil, ErrNameAlreadyUsed
		}

		return nil, err
	}

	return stage, nil
}

func (c *Canvas) DeleteInTransaction(tx *gorm.DB) error {
	return tx.
		Where("id = ?", c.ID).
		Delete(&Canvas{}).
		Error
}

func ListCanvases() ([]Canvas, error) {
	var canvases []Canvas

	err := database.Conn().
		Order("name ASC").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func ListCanvasesByIDs(ids []string, organizationID string) ([]Canvas, error) {
	var canvases []Canvas

	if organizationID != "" {
		err := database.Conn().
			Where("organization_id = ?", organizationID).
			Where("id IN (?)", ids).
			Order("name ASC").
			Find(&canvases).
			Error

		if err != nil {
			return nil, err
		}

		return canvases, nil
	}

	err := database.Conn().
		Where("id IN (?)", ids).
		Order("name ASC").
		Find(&canvases).
		Error

	if err != nil {
		return nil, err
	}

	return canvases, nil
}

func FindCanvasByID(id string) (*Canvas, error) {
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

func FindCanvasByName(name string) (*Canvas, error) {
	canvas := Canvas{}

	err := database.Conn().
		Where("name = ?", name).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func CreateCanvas(requesterID uuid.UUID, orgID uuid.UUID, name string, description string) (*Canvas, error) {
	now := time.Now()
	canvas := Canvas{
		Name:           name,
		Description:    description,
		OrganizationID: orgID,
		CreatedAt:      &now,
		CreatedBy:      requesterID,
		UpdatedAt:      &now,
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

// GetCanvasIDs returns only the IDs of all canvases
func GetCanvasIDs() ([]string, error) {
	var canvasIDs []string
	err := database.Conn().Model(&Canvas{}).
		Select("id").
		Pluck("id", &canvasIDs).Error

	if err != nil {
		return nil, err
	}

	return canvasIDs, nil
}
