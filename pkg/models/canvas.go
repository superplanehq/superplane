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

var (
	ErrNameAlreadyUsed         = fmt.Errorf("name already used")
	ErrInvitationAlreadyExists = fmt.Errorf("invitation already exists")
)

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
	deletedName := fmt.Sprintf("%s-deleted-%d", c.Name, time.Now().Unix())

	err := tx.Model(c).
		Where("id = ?", c.ID).
		Update("name", deletedName).
		Error
	if err != nil {
		return err
	}

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

func FindUnscopedCanvasByID(id string) (*Canvas, error) {
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

func FindUnscopedSoftDeletedCanvasByID(id string) (*Canvas, error) {
	canvas := Canvas{}

	err := database.Conn().
		Unscoped().
		Where("id = ?", id).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindCanvasByID(id string, organizationID uuid.UUID) (*Canvas, error) {
	canvas := Canvas{}

	err := database.Conn().
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&canvas).
		Error

	if err != nil {
		return nil, err
	}

	return &canvas, nil
}

func FindCanvasByName(name string, organizationID uuid.UUID) (*Canvas, error) {
	canvas := Canvas{}

	err := database.Conn().
		Where("name = ? AND organization_id = ?", name, organizationID).
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
