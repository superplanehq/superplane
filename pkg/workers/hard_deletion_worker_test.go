package workers

import (
	"testing"

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

	t.Run("hard delete stage with full dependency chain", func(t *testing.T) {

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

		event, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

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

		stageExecution, err := models.CreateStageExecution(r.Canvas.ID, stage.ID, stageEvent.ID)
		require.NoError(t, err)

		resource, err := r.Integration.CreateResource("test-type", "test-external-id", "test-resource")
		require.NoError(t, err)

		executionResource, err := stageExecution.AddResource("test-resource-id", "test-type", resource.ID)
		require.NoError(t, err)

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

		err = stage.Delete()
		require.NoError(t, err)

		err = worker.processStages()
		require.NoError(t, err)

		var foundStage models.Stage
		err = database.Conn().Unscoped().Where("id = ?", stage.ID).First(&foundStage).Error
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "record not found")
		}

		var foundStageEvent models.StageEvent
		err = database.Conn().Unscoped().Where("id = ?", stageEvent.ID).First(&foundStageEvent).Error
		assert.Error(t, err)

		var foundExecution models.StageExecution
		err = database.Conn().Unscoped().Where("id = ?", stageExecution.ID).First(&foundExecution).Error
		assert.Error(t, err)

		var foundResource models.ExecutionResource
		err = database.Conn().Unscoped().Where("id = ?", executionResource.ID).First(&foundResource).Error
		assert.Error(t, err)

		var foundConnection models.Connection
		err = database.Conn().Unscoped().Where("id = ?", connection.ID).First(&foundConnection).Error
		assert.Error(t, err)
	})

	t.Run("hard delete event source with dependencies", func(t *testing.T) {

		eventSource := models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "test-event-source-hard-delete",
			Key:        []byte(`test-key`),
			Scope:      models.EventSourceScopeExternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}
		err := eventSource.Create()
		require.NoError(t, err)

		event, err := models.CreateEvent(eventSource.ID, eventSource.CanvasID, eventSource.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

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

		err = eventSource.Delete()
		require.NoError(t, err)

		err = worker.processEventSources()
		require.NoError(t, err)

		var foundEventSource models.EventSource
		err = database.Conn().Unscoped().Where("id = ?", eventSource.ID).First(&foundEventSource).Error
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "record not found")
		}

		var foundEvent models.Event
		err = database.Conn().Unscoped().Where("id = ?", event.ID).First(&foundEvent).Error
		assert.NoError(t, err)

		var foundConnection models.Connection
		err = database.Conn().Unscoped().Where("id = ?", connection.ID).First(&foundConnection).Error
		assert.Error(t, err)
	})

	t.Run("hard delete event source with stage executions referencing stage events", func(t *testing.T) {
		eventSource := models.EventSource{
			CanvasID:   r.Canvas.ID,
			Name:       "test-event-source-with-executions",
			Key:        []byte(`test-key`),
			Scope:      models.EventSourceScopeExternal,
			EventTypes: datatypes.NewJSONSlice([]models.EventType{}),
		}
		err := eventSource.Create()
		require.NoError(t, err)

		stage := models.Stage{
			CanvasID:      r.Canvas.ID,
			Name:          "test-stage-for-event-source",
			Description:   "Test Stage for Event Source",
			ExecutorType:  models.ExecutorTypeHTTP,
			ExecutorName:  "test-executor",
			ExecutorSpec:  datatypes.JSON(`{}`),
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
		}
		err = database.Conn().Create(&stage).Error
		require.NoError(t, err)

		event, err := models.CreateEvent(eventSource.ID, eventSource.CanvasID, eventSource.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

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

		stageExecution, err := models.CreateStageExecution(r.Canvas.ID, stage.ID, stageEvent.ID)
		require.NoError(t, err)

		connection := models.Connection{
			CanvasID:   r.Canvas.ID,
			SourceID:   eventSource.ID,
			SourceName: eventSource.Name,
			SourceType: models.SourceTypeEventSource,
			TargetID:   stage.ID,
			TargetType: models.ConnectionTargetTypeStage,
		}
		err = database.Conn().Create(&connection).Error
		require.NoError(t, err)

		err = eventSource.Delete()
		require.NoError(t, err)

		err = worker.processEventSources()
		require.NoError(t, err)

		var foundEventSource models.EventSource
		err = database.Conn().Unscoped().Where("id = ?", eventSource.ID).First(&foundEventSource).Error
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "record not found")
		}

		var foundStageEvent models.StageEvent
		err = database.Conn().Unscoped().Where("id = ?", stageEvent.ID).First(&foundStageEvent).Error
		assert.NoError(t, err)

		var foundExecution models.StageExecution
		err = database.Conn().Unscoped().Where("id = ?", stageExecution.ID).First(&foundExecution).Error
		assert.NoError(t, err)

		var foundEvent models.Event
		err = database.Conn().Unscoped().Where("id = ?", event.ID).First(&foundEvent).Error
		assert.NoError(t, err)

		var foundConnection models.Connection
		err = database.Conn().Unscoped().Where("id = ?", connection.ID).First(&foundConnection).Error
		assert.Error(t, err)
	})

	t.Run("hard delete connection group with field sets", func(t *testing.T) {

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

		fields := map[string]string{"field1": "value1"}
		fieldSet, err := connectionGroup.CreateFieldSet(database.Conn(), fields, "test-hash")
		require.NoError(t, err)

		event, err := models.CreateEvent(connectionGroup.ID, connectionGroup.CanvasID, connectionGroup.Name, models.SourceTypeConnectionGroup, "group-event", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

		_, err = fieldSet.AttachEvent(database.Conn(), event)
		require.NoError(t, err)

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

		err = connectionGroup.Delete()
		require.NoError(t, err)

		err = worker.processConnectionGroups()
		require.NoError(t, err)

		var foundConnectionGroup models.ConnectionGroup
		err = database.Conn().Unscoped().Where("id = ?", connectionGroup.ID).First(&foundConnectionGroup).Error
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "record not found")
		}

		var foundFieldSet models.ConnectionGroupFieldSet
		err = database.Conn().Where("id = ?", fieldSet.ID).First(&foundFieldSet).Error
		assert.Error(t, err)

		var foundEvent models.Event
		err = database.Conn().Unscoped().Where("id = ?", event.ID).First(&foundEvent).Error
		assert.NoError(t, err)

		var foundConnection models.Connection
		err = database.Conn().Unscoped().Where("id = ?", connection.ID).First(&foundConnection).Error
		assert.Error(t, err)
	})

	t.Run("hard delete stage without resource avoids webhook cleanup", func(t *testing.T) {
		stage := models.Stage{
			CanvasID:      r.Canvas.ID,
			Name:          "test-stage-no-resource",
			Description:   "Test Stage without Resource",
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

		event, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

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

		stageExecution, err := models.CreateStageExecution(r.Canvas.ID, stage.ID, stageEvent.ID)
		require.NoError(t, err)

		resource := models.Resource{
			IntegrationID: r.Integration.ID,
			ExternalID:    "test-resource-id",
			ResourceName:  "test-resource",
			ResourceType:  "test-type",
		}
		err = database.Conn().Create(&resource).Error
		require.NoError(t, err)

		executionResource, err := stageExecution.AddResource("test-exec-resource", "test-type", resource.ID)
		require.NoError(t, err)

		childResource := models.Resource{
			IntegrationID: r.Integration.ID,
			ParentID:      &resource.ID,
			ExternalID:    "child-resource-id",
			ResourceName:  "child-resource",
			ResourceType:  "child-type",
		}
		err = database.Conn().Create(&childResource).Error
		require.NoError(t, err)

		childExecutionResource, err := stageExecution.AddResource("child-exec-resource", "child-type", childResource.ID)
		require.NoError(t, err)

		err = stage.Delete()
		require.NoError(t, err)

		err = worker.processStages()
		require.NoError(t, err)

		var foundStage models.Stage
		err = database.Conn().Unscoped().Where("id = ?", stage.ID).First(&foundStage).Error
		assert.Error(t, err)

		var foundExecution models.StageExecution
		err = database.Conn().Unscoped().Where("id = ?", stageExecution.ID).First(&foundExecution).Error
		assert.Error(t, err)

		var foundExecutionResource models.ExecutionResource
		err = database.Conn().Unscoped().Where("id = ?", executionResource.ID).First(&foundExecutionResource).Error
		assert.Error(t, err)

		var foundChildExecutionResource models.ExecutionResource
		err = database.Conn().Unscoped().Where("id = ?", childExecutionResource.ID).First(&foundChildExecutionResource).Error
		assert.Error(t, err)
	})

	t.Run("test execution resource cleanup by parent resource", func(t *testing.T) {
		resource := models.Resource{
			IntegrationID: r.Integration.ID,
			ExternalID:    "test-resource-direct",
			ResourceName:  "test-resource-direct",
			ResourceType:  "test-type",
		}
		err := database.Conn().Create(&resource).Error
		require.NoError(t, err)

		stage := models.Stage{
			CanvasID:      r.Canvas.ID,
			Name:          "test-stage-for-direct-cleanup",
			Description:   "Test Stage",
			ExecutorType:  models.ExecutorTypeHTTP,
			ExecutorName:  "test-executor",
			ExecutorSpec:  datatypes.JSON(`{}`),
			Conditions:    datatypes.NewJSONSlice([]models.StageCondition{}),
			Inputs:        datatypes.NewJSONSlice([]models.InputDefinition{}),
			InputMappings: datatypes.NewJSONSlice([]models.InputMapping{}),
			Outputs:       datatypes.NewJSONSlice([]models.OutputDefinition{}),
			Secrets:       datatypes.NewJSONSlice([]models.ValueDefinition{}),
		}
		err = database.Conn().Create(&stage).Error
		require.NoError(t, err)

		event, err := models.CreateEvent(r.Source.ID, r.Source.CanvasID, r.Source.Name, models.SourceTypeEventSource, "push", []byte(`{}`), []byte(`{}`))
		require.NoError(t, err)

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

		stageExecution, err := models.CreateStageExecution(r.Canvas.ID, stage.ID, stageEvent.ID)
		require.NoError(t, err)

		executionResource, err := stageExecution.AddResource("test-exec-resource", "test-type", resource.ID)
		require.NoError(t, err)

		err = models.DeleteResourceWithChildren(resource.ID)
		require.NoError(t, err)

		var foundResource models.Resource
		err = database.Conn().Unscoped().Where("id = ?", resource.ID).First(&foundResource).Error
		assert.Error(t, err)

		var foundExecutionResource models.ExecutionResource
		err = database.Conn().Unscoped().Where("id = ?", executionResource.ID).First(&foundExecutionResource).Error
		assert.Error(t, err)
	})

	t.Run("full worker tick processes all component types", func(t *testing.T) {

		err := worker.Tick()
		require.NoError(t, err)
	})
}
