package canvases

import (
	"context"
	"errors"
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
		require.NotNil(t, errors.Unwrap(err))
		assert.Contains(t, errors.Unwrap(err).Error(), "template")
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

	t.Run("run resolves template payload expressions", func(t *testing.T) {
		expressionNodeID := "start-node-expression"
		expressionCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: expressionNodeID,
					Name:   expressionNodeID,
					Type:   models.NodeTypeTrigger,
					Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
					Configuration: datatypes.NewJSONType(map[string]any{
						"templates": []any{
							map[string]any{
								"name": "Timed",
								"payload": map[string]any{
									"generatedAt": `{{ now().Format("2006-01-02") }}`,
								},
							},
						},
					}),
				},
			},
			nil,
		)

		resp, err := InvokeNodeTriggerHook(
			authedCtx,
			r.AuthService,
			r.Encryptor,
			r.Registry,
			r.Organization.ID,
			expressionCanvas.ID,
			expressionNodeID,
			"run",
			map[string]any{"template": "Timed"},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(expressionCanvas.ID, expressionNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		result := resp.Result.AsMap()
		assert.Equal(t, "Timed", result["template"])

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)

		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		generatedAt, ok := inner["generatedAt"].(string)
		require.True(t, ok)
		assert.NotContains(t, generatedAt, "{{")
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, generatedAt)
	})

	t.Run("run resolves plain now() expression", func(t *testing.T) {
		expressionNodeID := "start-node-expression-now"
		expressionCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: expressionNodeID,
					Name:   expressionNodeID,
					Type:   models.NodeTypeTrigger,
					Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
					Configuration: datatypes.NewJSONType(map[string]any{
						"templates": []any{
							map[string]any{
								"name": "Timed",
								"payload": map[string]any{
									"message": "{{ now() }}",
								},
							},
						},
					}),
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
			expressionCanvas.ID,
			expressionNodeID,
			"run",
			map[string]any{"template": "Timed"},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(expressionCanvas.ID, expressionNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		message, ok := inner["message"].(string)
		require.True(t, ok)
		assert.NotContains(t, message, "{{")
	})

	t.Run("run resolves template payload expressions from JSON string payload", func(t *testing.T) {
		expressionNodeID := "start-node-expression-json"
		expressionCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: expressionNodeID,
					Name:   expressionNodeID,
					Type:   models.NodeTypeTrigger,
					Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
					Configuration: datatypes.NewJSONType(map[string]any{
						"templates": []any{
							map[string]any{
								"name":    "Timed",
								"payload": "{\n  \"generatedAt\": \"{{ now().Format(\"2006-01-02\") }}\"\n}",
							},
						},
					}),
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
			expressionCanvas.ID,
			expressionNodeID,
			"run",
			map[string]any{"template": "Timed"},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(expressionCanvas.ID, expressionNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)

		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		generatedAt, ok := inner["generatedAt"].(string)
		require.True(t, ok)
		assert.NotContains(t, generatedAt, "{{")
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, generatedAt)
	})

	t.Run("run resolves template payload expressions using hook parameters", func(t *testing.T) {
		expressionNodeID := "start-node-expression-parameters"
		expressionCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: expressionNodeID,
					Name:   expressionNodeID,
					Type:   models.NodeTypeTrigger,
					Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
					Configuration: datatypes.NewJSONType(map[string]any{
						"templates": []any{
							map[string]any{
								"name": "Parameterized",
								"payload": map[string]any{
									"message": `{{ parameters["message"] }}`,
								},
							},
						},
					}),
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
			expressionCanvas.ID,
			expressionNodeID,
			"run",
			map[string]any{
				"template": "Parameterized",
				"message":  "Hello from hook parameter",
			},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(expressionCanvas.ID, expressionNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello from hook parameter", inner["message"])
	})

	t.Run("run resolves multiline text template parameters from an inline form payload", func(t *testing.T) {
		expressionNodeID := "start-node-text-parameter"
		expressionCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: expressionNodeID,
					Name:   "Submit task",
					Type:   models.NodeTypeTrigger,
					Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
					Configuration: datatypes.NewJSONType(map[string]any{
						"templates": []any{
							map[string]any{
								"name": "task",
								"parameters": []any{
									map[string]any{
										"name":        "prompt",
										"title":       "Task",
										"type":        "text",
										"placeholder": "Describe a small test task…",
									},
								},
								"payload": map[string]any{
									"title": `{{ parameters["prompt"] }}`,
								},
							},
						},
					}),
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
			expressionCanvas.ID,
			expressionNodeID,
			"run",
			map[string]any{
				"template": "task",
				"prompt":   "debug prompt",
			},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(expressionCanvas.ID, expressionNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "debug prompt", inner["title"])
	})

	t.Run("run resolves template payload expressions using configured template parameter defaults", func(t *testing.T) {
		expressionNodeID := "start-node-expression-parameter-default"
		expressionCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: expressionNodeID,
					Name:   expressionNodeID,
					Type:   models.NodeTypeTrigger,
					Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
					Configuration: datatypes.NewJSONType(map[string]any{
						"templates": []any{
							map[string]any{
								"name": "Parameterized",
								"payload": map[string]any{
									"message": `{{ parameters["message"] }}`,
								},
								"parameters": []any{
									map[string]any{
										"name":          "message",
										"type":          "string",
										"defaultString": "Hello from configured defaults",
									},
								},
							},
						},
					}),
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
			expressionCanvas.ID,
			expressionNodeID,
			"run",
			map[string]any{
				"template": "Parameterized",
			},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(expressionCanvas.ID, expressionNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Hello from configured defaults", inner["message"])
	})

	t.Run("run resolves template payload expressions using select parameter defaults", func(t *testing.T) {
		expressionNodeID := "start-node-expression-select-default"
		expressionCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: expressionNodeID,
					Name:   expressionNodeID,
					Type:   models.NodeTypeTrigger,
					Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
					Configuration: datatypes.NewJSONType(map[string]any{
						"templates": []any{
							map[string]any{
								"name": "Parameterized",
								"payload": map[string]any{
									"provider": `{{ parameters["provider"] }}`,
								},
								"parameters": []any{
									map[string]any{
										"name":          "provider",
										"type":          "select",
										"defaultString": "openai",
										"options": []any{
											map[string]any{"label": "OpenAI", "value": "openai"},
											map[string]any{"label": "Anthropic", "value": "anthropic"},
										},
									},
								},
							},
						},
					}),
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
			expressionCanvas.ID,
			expressionNodeID,
			"run",
			map[string]any{
				"template": "Parameterized",
			},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(expressionCanvas.ID, expressionNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "openai", inner["provider"])
	})

	t.Run("run resolves template payload expressions using select hook parameter override", func(t *testing.T) {
		expressionNodeID := "start-node-expression-select-override"
		expressionCanvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				{
					NodeID: expressionNodeID,
					Name:   expressionNodeID,
					Type:   models.NodeTypeTrigger,
					Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
					Configuration: datatypes.NewJSONType(map[string]any{
						"templates": []any{
							map[string]any{
								"name": "Parameterized",
								"payload": map[string]any{
									"provider": `{{ parameters["provider"] }}`,
								},
								"parameters": []any{
									map[string]any{
										"name":          "provider",
										"type":          "select",
										"defaultString": "openai",
										"options": []any{
											map[string]any{"label": "OpenAI", "value": "openai"},
											map[string]any{"label": "Anthropic", "value": "anthropic"},
										},
									},
								},
							},
						},
					}),
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
			expressionCanvas.ID,
			expressionNodeID,
			"run",
			map[string]any{
				"template": "Parameterized",
				"provider": "anthropic",
			},
			"http://localhost",
		)
		require.NoError(t, err)

		events, err := models.ListCanvasEvents(expressionCanvas.ID, expressionNodeID, 1, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		data, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		inner, ok := data["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "anthropic", inner["provider"])
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
