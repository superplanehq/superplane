package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNameAlreadyUsed = fmt.Errorf("name already used")

type Canvas struct {
	ID             uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Name           string
	CreatedAt      *time.Time
	CreatedBy      uuid.UUID
	UpdatedAt      *time.Time
	OrganizationID uuid.UUID

	Organization *Organization `gorm:"foreignKey:OrganizationID;references:ID"`
}

func (Canvas) TableName() string {
	return "canvases"
}

func (c *Canvas) CreateEventSource(name string, key []byte, resourceId *uuid.UUID) (*EventSource, error) {
	return c.CreateEventSourceInTransaction(database.Conn(), name, key, resourceId)
}

// NOTE: caller must encrypt the key before calling this method.
func (c *Canvas) CreateEventSourceInTransaction(tx *gorm.DB, name string, key []byte, resourceId *uuid.UUID) (*EventSource, error) {
	now := time.Now()

	eventSource := EventSource{
		Name:       name,
		CanvasID:   c.ID,
		CreatedAt:  &now,
		UpdatedAt:  &now,
		Key:        key,
		ResourceID: resourceId,
		State:      EventSourceStatePending,
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
	encryptor crypto.Encryptor,
	name, createdBy string,
	conditions []StageCondition,
	executor StageExecutor,
	resource *Resource,
	connections []Connection,
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

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Returning{}).Create(&stage).Error
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				return ErrNameAlreadyUsed
			}

			return err
		}

		err = c.createStageExecutor(tx, encryptor, stage, executor, resource)
		if err != nil {
			return err
		}

		for _, i := range connections {
			c := i
			c.TargetID = stage.ID
			c.TargetType = ConnectionTargetTypeStage
			err := tx.Create(&c).Error
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stage, nil
}

func (c *Canvas) createStageExecutor(tx *gorm.DB, encryptor crypto.Encryptor, stage *Stage, executor StageExecutor, resource *Resource) error {
	now := time.Now()

	//
	// If the executor is not associated with an integration/resource,
	// we just associate it with the stage, and return.
	//
	if resource == nil {
		executor.StageID = stage.ID
		return tx.Create(&executor).Error
	}

	//
	// Check if there is already a resource created for this.
	// If not, create one and attach it to a new event source.
	//
	r, err := FindResourceInTransaction(tx, resource.IntegrationID, resource.ResourceType, resource.ResourceName)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		r = resource
		r.CreatedAt = &now
		err := tx.Clauses(clause.Returning{}).Create(&r).Error
		if err != nil {
			return fmt.Errorf("error creating executor resource: %v", err)
		}

		_, key, err := genNewEventSourceKey(context.Background(), encryptor, resource.ResourceName)
		if err != nil {
			return err
		}

		_, err = c.CreateEventSourceInTransaction(tx, r.ResourceName, key, &r.ID)
		if err != nil {
			return fmt.Errorf("error creating event source for resource: %v", err)
		}
	}

	executor.ResourceID = r.ID

	//
	// Create stage executor
	//
	executor.StageID = stage.ID
	return tx.Create(&executor).Error
}

func (c *Canvas) UpdateStage(
	id, requesterID string,
	conditions []StageCondition,
	executor StageExecutor,
	resource *Resource,
	connections []Connection,
	inputs []InputDefinition,
	inputMappings []InputMapping,
	outputs []OutputDefinition,
	secrets []ValueDefinition,
) error {
	now := time.Now()

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("target_id = ?", id).Delete(&Connection{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing connections: %v", err)
		}

		if err := tx.Where("stage_id = ?", id).Delete(&StageExecutor{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing executor: %v", err)
		}

		for _, connection := range connections {
			connection.TargetID = uuid.Must(uuid.Parse(id))
			connection.TargetType = ConnectionTargetTypeStage
			if err := tx.Create(&connection).Error; err != nil {
				return fmt.Errorf("failed to create connection: %v", err)
			}
		}

		if resource != nil {
			resource.CreatedAt = &now
			err := tx.Clauses(clause.Returning{}).Create(&resource).Error
			if err != nil {
				return fmt.Errorf("error creating executor resource: %v", err)
			}

			executor.ResourceID = resource.ID
		}

		executor.StageID = uuid.MustParse(id)
		err := tx.Create(&executor).Error
		if err != nil {
			return err
		}

		now := time.Now()
		err = tx.Model(&Stage{}).
			Where("id = ?", id).
			Update("updated_at", now).
			Update("updated_by", requesterID).
			Update("conditions", datatypes.NewJSONSlice(conditions)).
			Update("inputs", datatypes.NewJSONSlice(inputs)).
			Update("input_mappings", datatypes.NewJSONSlice(inputMappings)).
			Update("outputs", datatypes.NewJSONSlice(outputs)).
			Update("secrets", datatypes.NewJSONSlice(secrets)).
			Error

		if err != nil {
			return fmt.Errorf("failed to update stage: %v", err)
		}

		return nil
	})
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

func CreateCanvas(requesterID uuid.UUID, orgID uuid.UUID, name string) (*Canvas, error) {
	now := time.Now()
	canvas := Canvas{
		Name:           name,
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

func genNewEventSourceKey(ctx context.Context, encryptor crypto.Encryptor, name string) (string, []byte, error) {
	plainKey, _ := crypto.Base64String(32)
	encrypted, err := encryptor.Encrypt(ctx, []byte(plainKey), []byte(name))
	if err != nil {
		return "", nil, err
	}

	return plainKey, encrypted, nil
}
