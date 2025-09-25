package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__UpdateReferencesAfterNameUpdateInTransaction_WithInputMappings(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	user := uuid.New()
	org, err := CreateOrganization(uuid.New().String(), "test")
	require.NoError(t, err)
	canvas, err := CreateCanvas(user, org.ID, "test", "test")
	require.NoError(t, err)

	source := &EventSource{
		CanvasID:   canvas.ID,
		Name:       "source-1",
		Key:        []byte(`my-key`),
		Scope:      EventSourceScopeExternal,
		EventTypes: datatypes.NewJSONSlice([]EventType{}),
	}
	err = source.Create()
	require.NoError(t, err)

	connectionGroup, err := CreateConnectionGroup(
		canvas.ID,
		"test-group",
		"test group",
		user.String(),
		[]Connection{
			{SourceID: source.ID, SourceName: source.Name, SourceType: SourceTypeEventSource},
		},
		ConnectionGroupSpec{
			GroupBy: &ConnectionGroupBySpec{
				Fields: []ConnectionGroupByField{
					{Name: "user_id", Expression: "$.user_id"},
				},
			},
		},
	)
	require.NoError(t, err)

	now := time.Now()
	stage := &Stage{
		CanvasID:     canvas.ID,
		Name:         "test-stage",
		Description:  "test stage",
		CreatedAt:    &now,
		UpdatedAt:    &now,
		CreatedBy:    user,
		UpdatedBy:    user,
		ExecutorType: ExecutorTypeHTTP,
		ExecutorSpec: datatypes.JSON(`{}`),
		ExecutorName: "test-executor",
		Conditions:   datatypes.NewJSONSlice([]StageCondition{}),
		Inputs:       datatypes.NewJSONSlice([]InputDefinition{}),
		InputMappings: datatypes.NewJSONSlice([]InputMapping{
			{
				When: &InputMappingWhen{
					TriggeredBy: &WhenTriggeredBy{
						Connection: "source-1",
					},
				},
				Values: []ValueDefinition{
					{
						Name: "user_id",
						ValueFrom: &ValueDefinitionFrom{
							EventData: &ValueDefinitionFromEventData{
								Connection: "source-1",
								Expression: "user.id",
							},
						},
					},
					{
						Name: "message",
						ValueFrom: &ValueDefinitionFrom{
							EventData: &ValueDefinitionFromEventData{
								Connection: "source-1",
								Expression: "data.message",
							},
						},
					},
				},
			},
		}),
		Outputs: datatypes.NewJSONSlice([]OutputDefinition{}),
		Secrets: datatypes.NewJSONSlice([]ValueDefinition{}),
	}

	err = database.Conn().Create(stage).Error
	require.NoError(t, err)

	var retrievedStage Stage
	err = database.Conn().Where("id = ?", stage.ID).First(&retrievedStage).Error
	require.NoError(t, err)

	require.Len(t, retrievedStage.InputMappings, 1)
	require.NotNil(t, retrievedStage.InputMappings[0].When)
	require.NotNil(t, retrievedStage.InputMappings[0].When.TriggeredBy)
	assert.Equal(t, "source-1", retrievedStage.InputMappings[0].When.TriggeredBy.Connection)
	require.Len(t, retrievedStage.InputMappings[0].Values, 2)
	assert.Equal(t, "source-1", retrievedStage.InputMappings[0].Values[0].ValueFrom.EventData.Connection)
	assert.Equal(t, "source-1", retrievedStage.InputMappings[0].Values[1].ValueFrom.EventData.Connection)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		return UpdateReferencesAfterNameUpdateInTransaction(tx, source.CanvasID, source.ID, SourceTypeEventSource, "source-1", "updated-source-name", nil)
	})
	require.NoError(t, err)

	err = database.Conn().Where("id = ?", stage.ID).First(&retrievedStage).Error
	require.NoError(t, err)

	require.Len(t, retrievedStage.InputMappings, 1)
	require.NotNil(t, retrievedStage.InputMappings[0].When)
	require.NotNil(t, retrievedStage.InputMappings[0].When.TriggeredBy)
	assert.Equal(t, "updated-source-name", retrievedStage.InputMappings[0].When.TriggeredBy.Connection)
	require.Len(t, retrievedStage.InputMappings[0].Values, 2)
	assert.Equal(t, "updated-source-name", retrievedStage.InputMappings[0].Values[0].ValueFrom.EventData.Connection)
	assert.Equal(t, "updated-source-name", retrievedStage.InputMappings[0].Values[1].ValueFrom.EventData.Connection)

	connections, err := ListConnections(connectionGroup.ID, ConnectionTargetTypeConnectionGroup)
	require.NoError(t, err)
	require.Len(t, connections, 1)
	assert.Equal(t, "updated-source-name", connections[0].SourceName)
}

func Test__UpdateReferencesAfterNameUpdateInTransaction(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	user := uuid.New()
	org, err := CreateOrganization(uuid.New().String(), "test")
	require.NoError(t, err)
	canvas, err := CreateCanvas(user, org.ID, "test", "test")
	require.NoError(t, err)
	source := &EventSource{
		CanvasID:   canvas.ID,
		Name:       "source-1",
		Key:        []byte(`my-key`),
		Scope:      EventSourceScopeExternal,
		EventTypes: datatypes.NewJSONSlice([]EventType{}),
	}

	err = source.Create()
	require.NoError(t, err)

	connectionGroup, err := CreateConnectionGroup(
		canvas.ID,
		"test-group",
		"test group",
		user.String(),
		[]Connection{
			{SourceID: source.ID, SourceName: source.Name, SourceType: SourceTypeEventSource},
		},
		ConnectionGroupSpec{
			GroupBy: &ConnectionGroupBySpec{
				Fields: []ConnectionGroupByField{
					{Name: "version", Expression: "$.ref"},
				},
			},
		},
	)
	require.NoError(t, err)

	connections, err := ListConnections(connectionGroup.ID, ConnectionTargetTypeConnectionGroup)
	require.NoError(t, err)
	require.Len(t, connections, 1)
	assert.Equal(t, "source-1", connections[0].SourceName)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		return UpdateReferencesAfterNameUpdateInTransaction(tx, source.CanvasID, source.ID, SourceTypeEventSource, "source-1", "updated-source-name", nil)
	})
	require.NoError(t, err)

	connections, err = ListConnections(connectionGroup.ID, ConnectionTargetTypeConnectionGroup)
	require.NoError(t, err)
	require.Len(t, connections, 1)
	assert.Equal(t, "updated-source-name", connections[0].SourceName)
}
