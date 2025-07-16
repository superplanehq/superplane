package builders

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StageBuilder struct {
	ctx              context.Context
	encryptor        crypto.Encryptor
	canvas           *models.Canvas
	requesterID      uuid.UUID
	existingStage    *models.Stage
	newStage         *models.Stage
	executorResource *models.Resource
	executorType     string
	executorSpec     *models.ExecutorSpec
	connections      []models.Connection
}

func NewStageBuilder() *StageBuilder {
	return &StageBuilder{
		ctx: context.Background(),
		newStage: &models.Stage{
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
		},
	}
}

func (b *StageBuilder) WithExistingStage(existingStage *models.Stage) *StageBuilder {
	b.existingStage = existingStage
	return b
}

func (b *StageBuilder) WithEncryptor(encryptor crypto.Encryptor) *StageBuilder {
	b.encryptor = encryptor
	return b
}

func (b *StageBuilder) WithContext(ctx context.Context) *StageBuilder {
	b.ctx = ctx
	return b
}

func (b *StageBuilder) InCanvas(canvas *models.Canvas) *StageBuilder {
	b.canvas = canvas
	return b
}

func (b *StageBuilder) WithName(name string) *StageBuilder {
	b.newStage.Name = name
	return b
}

func (b *StageBuilder) WithRequester(requesterID uuid.UUID) *StageBuilder {
	b.requesterID = requesterID
	return b
}

func (b *StageBuilder) WithConditions(conditions []models.StageCondition) *StageBuilder {
	b.newStage.Conditions = datatypes.NewJSONSlice(conditions)
	return b
}

func (b *StageBuilder) WithInputs(inputs []models.InputDefinition) *StageBuilder {
	b.newStage.Inputs = datatypes.NewJSONSlice(inputs)
	return b
}

func (b *StageBuilder) WithInputMappings(inputMappings []models.InputMapping) *StageBuilder {
	b.newStage.InputMappings = datatypes.NewJSONSlice(inputMappings)
	return b
}

func (b *StageBuilder) WithOutputs(outputs []models.OutputDefinition) *StageBuilder {
	b.newStage.Outputs = datatypes.NewJSONSlice(outputs)
	return b
}

func (b *StageBuilder) WithSecrets(secrets []models.ValueDefinition) *StageBuilder {
	b.newStage.Secrets = datatypes.NewJSONSlice(secrets)
	return b
}

func (b *StageBuilder) WithConnections(connections []models.Connection) *StageBuilder {
	b.connections = connections
	return b
}

func (b *StageBuilder) WithExecutorType(executorType string) *StageBuilder {
	b.executorType = executorType
	return b
}

func (b *StageBuilder) WithExecutorResource(resource *models.Resource) *StageBuilder {
	b.executorResource = resource
	return b
}

func (b *StageBuilder) WithExecutorSpec(spec *models.ExecutorSpec) *StageBuilder {
	b.executorSpec = spec
	return b
}

func (b *StageBuilder) Create() (*models.Stage, error) {
	now := time.Now()
	stage := &models.Stage{
		CanvasID:      b.canvas.ID,
		Name:          b.newStage.Name,
		Conditions:    b.newStage.Conditions,
		CreatedAt:     &now,
		CreatedBy:     b.requesterID,
		Inputs:        b.newStage.Inputs,
		InputMappings: b.newStage.InputMappings,
		Outputs:       b.newStage.Outputs,
		Secrets:       b.newStage.Secrets,
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {

		//
		// Create the stage record
		//
		err := tx.Clauses(clause.Returning{}).Create(&stage).Error
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				return models.ErrNameAlreadyUsed
			}

			return err
		}

		//
		// Create the connections for the new stage
		//
		for _, connection := range b.connections {
			err := stage.AddConnection(tx, connection)
			if err != nil {
				return err
			}
		}

		//
		// Create the stage executor
		//
		eventSource, err := b.findOrCreateEventSource(tx)
		if err != nil {
			return err
		}

		executor := models.StageExecutor{
			Type:       b.executorType,
			Spec:       datatypes.NewJSONType(*b.executorSpec),
			ResourceID: *eventSource.ResourceID,
			StageID:    stage.ID,
		}

		return tx.Create(&executor).Error
	})

	if err != nil {
		return nil, err
	}

	return stage, nil
}

func (b *StageBuilder) findOrCreateEventSource(tx *gorm.DB) (*models.EventSource, error) {
	//
	// If this stage is not using an integration, it does not need an event source.
	//
	if b.executorResource == nil {
		return nil, nil
	}

	//
	// Check if the resource already exists.
	// If it does, return the event source for it.
	//
	r, err := models.FindResourceInTransaction(tx, b.executorResource.IntegrationID, b.executorResource.ResourceType, b.executorResource.ResourceName)
	if err == nil {
		return models.FindEventSourceForResourceInTransaction(tx, r.ID)
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	//
	// If it doesn't, create it and attach it to a new event source.
	//
	now := time.Now()
	r = b.executorResource
	r.CreatedAt = &now
	err = tx.Clauses(clause.Returning{}).Create(&r).Error
	if err != nil {
		return nil, err
	}

	_, key, err := crypto.NewRandomKey(b.ctx, b.encryptor, b.executorResource.ResourceName)
	if err != nil {
		return nil, err
	}

	eventSource, err := b.canvas.CreateEventSourceInTransaction(tx, r.ResourceName, key, models.EventSourceScopeInternal, &r.ID)
	if err != nil {
		return nil, err
	}

	return eventSource, nil
}

func (b *StageBuilder) Update() (*models.Stage, error) {
	if b.existingStage == nil {
		return nil, fmt.Errorf("no existing stage specified")
	}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {

		//
		// Delete existing connections and executor
		//
		if err := tx.Where("target_id = ?", b.existingStage.ID).Delete(&models.Connection{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing connections: %v", err)
		}

		if err := tx.Where("stage_id = ?", b.existingStage.ID).Delete(&models.StageExecutor{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing executor: %v", err)
		}

		//
		// Update the stage record.
		//
		now := time.Now()
		err := tx.Model(b.existingStage).
			Update("updated_at", now).
			Update("updated_by", b.requesterID).
			Update("conditions", b.newStage.Conditions).
			Update("inputs", b.newStage.Inputs).
			Update("input_mappings", b.newStage.InputMappings).
			Update("outputs", b.newStage.Outputs).
			Update("secrets", b.newStage.Secrets).
			Error

		if err != nil {
			return fmt.Errorf("failed to update stage: %v", err)
		}

		//
		// Re-create the connections for the new stage
		//
		for _, connection := range b.connections {
			err := b.existingStage.AddConnection(tx, connection)
			if err != nil {
				return err
			}
		}

		//
		// Re-create the stage executor
		//
		eventSource, err := b.findOrCreateEventSource(tx)
		if err != nil {
			return err
		}

		executor := models.StageExecutor{
			Type:       b.executorType,
			Spec:       datatypes.NewJSONType(*b.executorSpec),
			ResourceID: *eventSource.ResourceID,
			StageID:    b.existingStage.ID,
		}

		return tx.Create(&executor).Error
	})

	if err != nil {
		return nil, err
	}

	return b.existingStage, nil
}
