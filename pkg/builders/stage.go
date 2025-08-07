package builders

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type StageBuilder struct {
	ctx           context.Context
	encryptor     crypto.Encryptor
	registry      *registry.Registry
	canvasID      uuid.UUID
	requesterID   uuid.UUID
	existingStage *models.Stage
	newStage      *models.Stage
	resource      integrations.Resource
	integration   *models.Integration
	executorType  string
	executorSpec  []byte
	connections   []models.Connection
}

func NewStageBuilder(registry *registry.Registry) *StageBuilder {
	return &StageBuilder{
		ctx:      context.Background(),
		registry: registry,
		newStage: &models.Stage{
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
			Description:   "",
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

func (b *StageBuilder) InCanvas(canvasID uuid.UUID) *StageBuilder {
	b.canvasID = canvasID
	return b
}

func (b *StageBuilder) WithName(name string) *StageBuilder {
	b.newStage.Name = name
	return b
}

func (b *StageBuilder) WithDescription(description string) *StageBuilder {
	b.newStage.Description = description
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

func (b *StageBuilder) ForIntegration(integration *models.Integration) *StageBuilder {
	b.integration = integration
	return b
}

func (b *StageBuilder) ForResource(resource integrations.Resource) *StageBuilder {
	b.resource = resource
	return b
}

func (b *StageBuilder) WithExecutorSpec(spec []byte) *StageBuilder {
	b.executorSpec = spec
	return b
}

func (b *StageBuilder) Create() (*models.Stage, error) {
	err := b.validateExecutorSpec()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	stage := &models.Stage{
		CanvasID:      b.canvasID,
		Name:          b.newStage.Name,
		Description:   b.newStage.Description,
		Conditions:    b.newStage.Conditions,
		CreatedAt:     &now,
		CreatedBy:     b.requesterID,
		Inputs:        b.newStage.Inputs,
		InputMappings: b.newStage.InputMappings,
		Outputs:       b.newStage.Outputs,
		Secrets:       b.newStage.Secrets,
		ExecutorType:  b.executorType,
		ExecutorSpec:  datatypes.JSON(b.executorSpec),
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {

		//
		// Find or create the event source for the executor
		//
		resourceID, err := b.findOrCreateEventSourceForExecutor(tx)
		if err != nil {
			return err
		}

		stage.ResourceID = resourceID

		//
		// Create the stage record
		//
		err = tx.Clauses(clause.Returning{}).Create(&stage).Error
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

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stage, nil
}

func (b *StageBuilder) validateExecutorSpec() error {
	if b.executorSpec == nil {
		return fmt.Errorf("missing executor spec")
	}

	if b.integration == nil {
		executor, err := b.registry.NewExecutor(b.executorType)
		if err != nil {
			return err
		}

		return executor.Validate(b.ctx, b.executorSpec)
	}

	executor, err := b.registry.NewIntegrationExecutor(b.integration, b.resource)
	if err != nil {
		return err
	}

	return executor.Validate(b.ctx, b.executorSpec)
}

func (b *StageBuilder) findOrCreateEventSourceForExecutor(tx *gorm.DB) (*uuid.UUID, error) {
	//
	// If this stage is not using an integration, it does not need an event source.
	//
	if b.resource == nil {
		return nil, nil
	}

	//
	// The event source builder will ensure an event source for this resource
	// will be re-used if already exists, or a new one will be created.
	//
	eventSource, _, err := NewEventSourceBuilder(b.encryptor).
		WithTransaction(tx).
		WithContext(b.ctx).
		InCanvas(b.canvasID).
		WithName(b.integration.Name + "-" + b.resource.Name()).
		WithScope(models.EventSourceScopeInternal).
		ForIntegration(b.integration).
		ForResource(b.resource).
		Create()

	if err != nil {
		return nil, err
	}

	return eventSource.ResourceID, nil
}

func (b *StageBuilder) Update() (*models.Stage, error) {
	if b.existingStage == nil {
		return nil, fmt.Errorf("no existing stage specified")
	}

	err := b.validateExecutorSpec()
	if err != nil {
		return nil, err
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {

		//
		// Delete existing connections
		//
		if err := tx.Where("target_id = ?", b.existingStage.ID).Delete(&models.Connection{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing connections: %v", err)
		}

		//
		// Find or create the event source for the executor
		//
		resourceID, err := b.findOrCreateEventSourceForExecutor(tx)
		if err != nil {
			return err
		}

		//
		// Update the stage record.
		//
		now := time.Now()
		err = tx.Model(b.existingStage).
			Update("name", b.newStage.Name).
			Update("description", b.newStage.Description).
			Update("updated_at", now).
			Update("updated_by", b.requesterID).
			Update("conditions", b.newStage.Conditions).
			Update("inputs", b.newStage.Inputs).
			Update("input_mappings", b.newStage.InputMappings).
			Update("outputs", b.newStage.Outputs).
			Update("secrets", b.newStage.Secrets).
			Update("executor_type", b.executorType).
			Update("executor_spec", datatypes.JSON(b.executorSpec)).
			Update("resource_id", resourceID).
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

		return nil
	})

	if err != nil {
		return nil, err
	}

	return b.existingStage, nil
}
