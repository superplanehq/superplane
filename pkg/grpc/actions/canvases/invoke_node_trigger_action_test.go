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

func Test__InvokeNodeTriggerAction__StartRun(t *testing.T) {
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
		_, err := InvokeNodeTriggerAction(
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

	t.Run("unknown action -> error", func(t *testing.T) {
		_, err := InvokeNodeTriggerAction(
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
		_, err := InvokeNodeTriggerAction(
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
		resp, err := InvokeNodeTriggerAction(
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
		resp, err := InvokeNodeTriggerAction(
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
}
