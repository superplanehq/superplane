package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__InvokeNodeTriggerHook__StartRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "start-node"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Name:   triggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"templates": []any{
						map[string]any{
							"name":    "Hello World",
							"payload": map[string]any{"message": "Hello, World!"},
						},
					},
				}),
			},
		},
		nil,
	)

	authedCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated context -> error", func(t *testing.T) {
		_, err := InvokeNodeTriggerHook(
			context.Background(),
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			triggerNodeID,
			"run",
			map[string]any{"template": "Hello World"},
			"http://localhost",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not authenticated")
	})

	t.Run("unknown hook -> error", func(t *testing.T) {
		_, err := InvokeNodeTriggerHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			triggerNodeID,
			"nope",
			map[string]any{},
			"http://localhost",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("missing template parameter -> error", func(t *testing.T) {
		_, err := InvokeNodeTriggerHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			triggerNodeID,
			"run",
			map[string]any{},
			"http://localhost",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "template")
	})

	t.Run("successful run persists event on default channel", func(t *testing.T) {
		resp, err := InvokeNodeTriggerHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			triggerNodeID,
			"run",
			map[string]any{"template": "Hello World"},
			"http://localhost",
		)
		require.NoError(t, err)
		require.NotNil(t, resp)

		result := resp.Result.AsMap()
		assert.Equal(t, "Hello World", result["template"])

		events, err := models.ListCanvasEvents(canvas.ID, triggerNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		event := events[0]
		assert.Equal(t, "default", event.Channel)
		assert.Equal(t, models.CanvasEventStatePending, event.State)

		data, ok := event.Data.Data().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "manual.run", data["type"])

		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello, World!", inner["message"])
	})

	t.Run("payload override replaces configured payload", func(t *testing.T) {
		resp, err := InvokeNodeTriggerHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			triggerNodeID,
			"run",
			map[string]any{
				"template": "Hello World",
				"payload":  map[string]any{"message": "Override"},
			},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(canvas.ID, triggerNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		result := resp.Result.AsMap()
		assert.Equal(t, "Hello World", result["template"])

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)

		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Override", inner["message"])
	})

	t.Run("non-trigger node -> error", func(t *testing.T) {
		componentNodeID := "component-node"
		canvasWithComponent, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID:        componentNodeID,
					Name:          componentNodeID,
					Type:          models.NodeTypeComponent,
					Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
					Configuration: datatypes.NewJSONType(map[string]any{}),
				},
			},
			nil,
		)

		_, err := InvokeNodeTriggerHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvasWithComponent.ID,
			componentNodeID,
			"run",
			map[string]any{"template": "Hello World"},
			"http://localhost",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a trigger")
	})
}

func Test__InvokeNodeTriggerHook__ScheduleRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "schedule-node"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Name:   triggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"type":            "minutes",
					"minutesInterval": 5,
				}),
			},
		},
		nil,
	)

	authedCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("internal hook cannot be called by user", func(t *testing.T) {
		_, err := InvokeNodeTriggerHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			triggerNodeID,
			"emitEvent",
			map[string]any{},
			"http://localhost",
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be invoked by user")
	})

	t.Run("run emits scheduler tick and returns event id", func(t *testing.T) {
		resp, err := InvokeNodeTriggerHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			canvas.ID,
			triggerNodeID,
			"run",
			map[string]any{},
			"http://localhost",
		)
		require.NoError(t, err)
		require.NotNil(t, resp)

		result := resp.Result.AsMap()
		eventID, ok := result["event_id"].(string)
		require.True(t, ok)
		require.NotEmpty(t, eventID)

		events, err := models.ListCanvasEvents(canvas.ID, triggerNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		assert.Equal(t, events[0].ID.String(), eventID)
		assert.Equal(t, "default", events[0].Channel)
		assert.Equal(t, models.CanvasEventStatePending, events[0].State)

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "scheduler.tick", data["type"])
	})
}
