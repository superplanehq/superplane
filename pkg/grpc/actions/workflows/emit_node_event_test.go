package workflows

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__EmitNodeEvent(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	t.Run("workflow not found -> error", func(t *testing.T) {
		_, err := EmitNodeEvent(
			ctx,
			r.Organization.ID,
			uuid.New(),
			"node-1",
			"default",
			map[string]any{"test": "data"},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflow not found")
	})

	t.Run("node not found -> error", func(t *testing.T) {
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "First Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		_, err := EmitNodeEvent(
			ctx,
			r.Organization.ID,
			workflow.ID,
			"non-existent-node",
			"default",
			map[string]any{"test": "data"},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node not found")
	})

	t.Run("successful event emission creates database record", func(t *testing.T) {
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		testData := map[string]any{
			"message": "hello world",
			"count":   42,
		}

		response, err := EmitNodeEvent(
			ctx,
			r.Organization.ID,
			workflow.ID,
			"node-1",
			"test-channel",
			testData,
		)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.NotEmpty(t, response.EventId)

		eventID, err := uuid.Parse(response.EventId)
		require.NoError(t, err)

		event, err := models.FindWorkflowEvent(eventID)
		require.NoError(t, err)

		assert.Equal(t, workflow.ID, event.WorkflowID)
		assert.Equal(t, "node-1", event.NodeID)
		assert.Equal(t, "test-channel", event.Channel)
		assert.Equal(t, models.WorkflowEventStatePending, event.State)
		assert.NotNil(t, event.CreatedAt)

		eventData := event.Data.Data()
		dataMap, ok := eventData.(map[string]any)
		require.True(t, ok, "event data should be a map")
		assert.Equal(t, "hello world", dataMap["message"])
		assert.Equal(t, float64(42), dataMap["count"])
	})

	t.Run("custom name is resolved from node configuration", func(t *testing.T) {
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		node, err := workflow.FindNode("node-1")
		require.NoError(t, err)
		node.Configuration = datatypes.NewJSONType(map[string]any{
			"customName": "Run: {{ $.message }}",
		})
		require.NoError(t, database.Conn().Save(node).Error)

		response, err := EmitNodeEvent(
			ctx,
			r.Organization.ID,
			workflow.ID,
			"node-1",
			"default",
			map[string]any{"message": "hello"},
		)
		require.NoError(t, err)

		eventID, err := uuid.Parse(response.EventId)
		require.NoError(t, err)

		event, err := models.FindWorkflowEvent(eventID)
		require.NoError(t, err)
		require.NotNil(t, event.CustomName)
		assert.Equal(t, "Run: hello", *event.CustomName)
	})

	t.Run("successful event emission publishes RabbitMQ message", func(t *testing.T) {
		amqpURL, _ := config.RabbitMQURL()
		testconsumer := testconsumer.New(amqpURL, messages.WorkflowEventCreatedRoutingKey)
		testconsumer.Start()
		defer testconsumer.Stop()

		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		_, err := EmitNodeEvent(
			ctx,
			r.Organization.ID,
			workflow.ID,
			"node-1",
			"default",
			map[string]any{"test": "data"},
		)

		require.NoError(t, err)

		assert.True(t, testconsumer.HasReceivedMessage())
	})

	t.Run("invalid organization ID -> error", func(t *testing.T) {
		_, err := EmitNodeEvent(
			ctx,
			uuid.New(),
			uuid.New(),
			"node-1",
			"default",
			map[string]any{"test": "data"},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workflow not found")
	})

	t.Run("empty node ID -> error", func(t *testing.T) {
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		_, err := EmitNodeEvent(
			ctx,
			r.Organization.ID,
			workflow.ID,
			"",
			"default",
			map[string]any{"test": "data"},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "node not found")
	})

	t.Run("nil data map is handled gracefully", func(t *testing.T) {
		workflow, _ := support.CreateWorkflow(
			t,
			r.Organization.ID,
			r.User,
			[]models.WorkflowNode{
				{
					NodeID: "node-1",
					Name:   "Test Node",
					Type:   models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					}),
				},
			},
			[]models.Edge{},
		)

		response, err := EmitNodeEvent(
			ctx,
			r.Organization.ID,
			workflow.ID,
			"node-1",
			"default",
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.NotEmpty(t, response.EventId)

		eventID, err := uuid.Parse(response.EventId)
		require.NoError(t, err)

		event, err := models.FindWorkflowEvent(eventID)
		require.NoError(t, err)

		eventData := event.Data.Data()
		assert.Nil(t, eventData)
	})
}
