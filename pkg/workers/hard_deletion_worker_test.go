package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__HardDeletionWorker(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{
		Source:      true,
		Integration: true,
	})
	defer r.Close()

	cleanupService := NewResourceCleanupService(r.Registry)
	worker := NewHardDeletionWorker(r.Registry, cleanupService)
	// Use shorter grace period for testing
	worker.GracePeriod = time.Millisecond * 100

	t.Run("hard delete stage with full dependency chain", func(t *testing.T) {
		// Create a stage with full dependency chain
		stage := models.Stage{
			CanvasID:      r.Canvas.ID,
			Name:          "test-stage-hard-delete",
			Description:   "Test Stage for Hard Delete",
			ExecutorType:  models.ExecutorTypeHTTP,
			ExecutorName:  "test-executor",
			ExecutorSpec:  datatypes.JSON(`{}`),
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
		}
		err := database.Conn().Create(&stage).Error
		require.NoError(t, err)

		// Create an event
		event, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

		// Create a stage event
		stageEvent, err := models.CreateStageEventInTransaction(
			database.Conn(),
			stage.ID,
			event,
			models.StageEventStatePending,
			"",
			map[string]any{"test": "value"},
			"test-executor",
		)
		require.NoError(t, err)

		// Create a stage execution
		stageExecution, err := models.CreateStageExecution(r.Canvas.ID, stage.ID, stageEvent.ID)
		require.NoError(t, err)

		// Create a real resource first for the execution resource to reference
		resource, err := r.Integration.CreateResource("test-type", "test-external-id", "test-resource")
		require.NoError(t, err)

		// Create an execution resource
		executionResource, err := stageExecution.AddResource("test-resource-id", "test-type", resource.ID)
		require.NoError(t, err)

		// Create a connection
		connection := models.Connection{
			CanvasID:   r.Canvas.ID,
			SourceID:   r.Source.ID,
			SourceName: r.Source.Name,
			SourceType: models.SourceTypeEventSource,
			TargetID:   stage.ID,
			TargetType: models.ConnectionTargetTypeStage,
		}
		err = database.Conn().Create(&connection).Error
		require.NoError(t, err)

		// Soft delete the stage
		err = stage.Delete()
		require.NoError(t, err)

		// Wait for grace period
		time.Sleep(worker.GracePeriod + time.Millisecond*10)

		// Run the worker
		err = worker.processStages()
		require.NoError(t, err)

		// Verify stage is hard deleted
		var foundStage models.Stage
		err = database.Conn().Unscoped().Where("id = ?", stage.ID).First(&foundStage).Error
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "record not found")
		}

		// Verify stage event is deleted
		var foundStageEvent models.StageEvent
		err = database.Conn().Unscoped().Where("id = ?", stageEvent.ID).First(&foundStageEvent).Error
		assert.Error(t, err)

		// Verify stage execution is deleted
		var foundExecution models.StageExecution
		err = database.Conn().Unscoped().Where("id = ?", stageExecution.ID).First(&foundExecution).Error
		assert.Error(t, err)

		// Verify execution resource is deleted
		var foundResource models.ExecutionResource
		err = database.Conn().Unscoped().Where("id = ?", executionResource.ID).First(&foundResource).Error
		assert.Error(t, err)

		// Verify connection is deleted
		var foundConnection models.Connection
		err = database.Conn().Unscoped().Where("id = ?", connection.ID).First(&foundConnection).Error
		assert.Error(t, err)
	})

	t.Run("hard delete event source with dependencies", func(t *testing.T) {
		// Create an event source
		eventSource := models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "test-event-source-hard-delete",
			Key:        []byte(`test-key`),
			Scope:      models.EventSourceScopeExternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}
		err := eventSource.Create()
		require.NoError(t, err)

		// Create an event from this source
		event, err := models.CreateEvent(eventSource.ID, eventSource.CanvasID, eventSource.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

		// Create a connection
		connection := models.Connection{
			CanvasID:   r.Canvas.ID,
			SourceID:   eventSource.ID,
			SourceName: eventSource.Name,
			SourceType: models.SourceTypeEventSource,
			TargetID:   uuid.New(),
			TargetType: models.ConnectionTargetTypeStage,
		}
		err = database.Conn().Create(&connection).Error
		require.NoError(t, err)

		// Soft delete the event source
		err = eventSource.Delete()
		require.NoError(t, err)

		// Wait for grace period
		time.Sleep(worker.GracePeriod + time.Millisecond*10)

		// Run the worker
		err = worker.processEventSources()
		require.NoError(t, err)

		// Verify event source is hard deleted
		var foundEventSource models.EventSource
		err = database.Conn().Unscoped().Where("id = ?", eventSource.ID).First(&foundEventSource).Error
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "record not found")
		}

		// Verify event is deleted
		var foundEvent models.Event
		err = database.Conn().Unscoped().Where("id = ?", event.ID).First(&foundEvent).Error
		assert.Error(t, err)

		// Verify connection is deleted
		var foundConnection models.Connection
		err = database.Conn().Unscoped().Where("id = ?", connection.ID).First(&foundConnection).Error
		assert.Error(t, err)
	})

	t.Run("hard delete connection group with field sets", func(t *testing.T) {
		// Create a connection group
		spec := models.ConnectionGroupSpec{
			GroupBy: &models.ConnectionGroupBySpec{
				Fields: []models.ConnectionGroupByField{
					{Name: "field1", Expression: "$.test"},
				},
			},
			Timeout:         300,
			TimeoutBehavior: models.ConnectionGroupTimeoutBehaviorNone,
		}

		connectionGroup, err := models.CreateConnectionGroup(
			r.Canvas.ID,
			"test-connection-group-hard-delete",
			"Test Connection Group for Hard Delete",
			r.User.String(),
			[]models.Connection{},
			spec,
		)
		require.NoError(t, err)

		// Create a field set
		fields := map[string]string{"field1": "value1"}
		fieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), fields, "test-hash")
		require.NoError(t, err)

		// Create an event for this connection group
		event, err := models.CreateEvent(connectionGroup.ID, connectionGroup.CanvasID, connectionGroup.Name, models.SourceTypeConnectionGroup, "group-event", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

		// Attach event to field set
		_, err = fieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

		// Create a connection
		connection := models.Connection{
			CanvasID:   r.Canvas.ID,
			SourceID:   connectionGroup.ID,
			SourceName: connectionGroup.Name,
			SourceType: models.SourceTypeConnectionGroup,
			TargetID:   uuid.New(),
			TargetType: models.ConnectionTargetTypeStage,
		}
		err = database.Conn().Create(&connection).Error
		require.NoError(t, err)

		// Soft delete the connection group
		err = connectionGroup.Delete()
		require.NoError(t, err)

		// Wait for grace period
		time.Sleep(worker.GracePeriod + time.Millisecond*10)

		// Run the worker
		err = worker.processConnectionGroups()
		require.NoError(t, err)

		// Verify connection group is hard deleted
		var foundConnectionGroup models.ConnectionGroup
		err = database.Conn().Unscoped().Where("id = ?", connectionGroup.ID).First(&foundConnectionGroup).Error
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "record not found")
		}

		// Verify field set is deleted (field sets don't use soft delete)
		var foundFieldSet models.ConnectionGroupFieldSet
		err = database.Conn().Where("id = ?", fieldSet.ID).First(&foundFieldSet).Error
		assert.Error(t, err)

		// Verify event is deleted
		var foundEvent models.Event
		err = database.Conn().Unscoped().Where("id = ?", event.ID).First(&foundEvent).Error
		assert.Error(t, err)

		// Verify connection is deleted
		var foundConnection models.Connection
		err = database.Conn().Unscoped().Where("id = ?", connection.ID).First(&foundConnection).Error
		assert.Error(t, err)
	})

	t.Run("respects grace period - skips recently deleted items", func(t *testing.T) {
		// Create and soft delete a stage
		stage := models.Stage{
			CanvasID:      r.Canvas.ID,
			Name:          "test-stage-grace-period",
			Description:   "Test Stage for Grace Period",
			ExecutorType:  models.ExecutorTypeHTTP,
			ExecutorName:  "test-executor",
			ExecutorSpec:  datatypes.JSON(`{}`),
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
		}
		err := database.Conn().Create(&stage).Error
		require.NoError(t, err)

		err = stage.Delete()
		require.NoError(t, err)

		// Don't wait for grace period - should skip this stage
		err = worker.processStages()
		require.NoError(t, err)

		// Verify stage still exists in soft deleted state
		var foundStage models.Stage
		err = database.Conn().Unscoped().Where("id = ?", stage.ID).First(&foundStage).Error
		require.NoError(t, err)
		assert.NotNil(t, foundStage.DeletedAt)
	})

	t.Run("full worker tick processes all component types", func(t *testing.T) {
		// This test ensures the main Tick() method calls all processing functions
		// without errors when there are no soft deleted items
		err := worker.Tick()
		require.NoError(t, err)
	})
}
