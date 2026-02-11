package rootly

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIncident__Setup(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "abc123",
				"title":      "Updated Title",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing incident ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title": "Some Title",
			},
		})

		require.ErrorContains(t, err, "incident ID is required")
	})

	t.Run("empty incident ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
				"title":      "Some Title",
			},
		})

		require.ErrorContains(t, err, "incident ID is required")
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "not a map",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("only incident ID is required - all other fields optional", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "abc123",
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpdateIncident__Execute(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("successfully updates incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "abc123",
							"type": "incidents",
							"attributes": {
								"sequential_id": 42,
								"title": "Updated Incident Title",
								"slug": "updated-incident-title",
								"status": "mitigated",
								"updated_at": "2024-01-15T14:30:00Z"
							}
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "abc123",
				"title":      "Updated Incident Title",
				"status":     "mitigated",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "rootly.incident", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPut, req.Method)
		assert.Equal(t, "https://api.rootly.com/v1/incidents/abc123", req.URL.String())
		assert.Equal(t, "application/vnd.api+json", req.Header.Get("Content-Type"))
		assert.Equal(t, "application/vnd.api+json", req.Header.Get("Accept"))
		assert.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
	})

	t.Run("updates incident with all fields", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "abc123",
							"type": "incidents",
							"attributes": {
								"sequential_id": 42,
								"title": "New Title",
								"slug": "new-title",
								"status": "resolved",
								"updated_at": "2024-01-15T14:30:00Z"
							}
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "abc123",
				"title":      "New Title",
				"summary":    "New summary with details",
				"status":     "resolved",
				"severity":   "high",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var payload map[string]any
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)

		data := payload["data"].(map[string]any)
		assert.Equal(t, "incidents", data["type"])
		attributes := data["attributes"].(map[string]any)
		assert.Equal(t, "New Title", attributes["title"])
		assert.Equal(t, "New summary with details", attributes["summary"])
		assert.Equal(t, "resolved", attributes["status"])
		assert.Equal(t, "high", attributes["severity"])
	})

	t.Run("updates incident with labels", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "abc123",
							"type": "incidents",
							"attributes": {
								"sequential_id": 42,
								"title": "Test Incident",
								"slug": "test-incident",
								"status": "started",
								"updated_at": "2024-01-15T14:30:00Z"
							}
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "abc123",
				"labels": []map[string]any{
					{"key": "platform", "value": "backend-api"},
					{"key": "region", "value": "us-east-1"},
				},
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var payload map[string]any
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)

		data := payload["data"].(map[string]any)
		attributes := data["attributes"].(map[string]any)
		labelsSlugs := attributes["labels_slugs"].([]any)
		assert.Len(t, labelsSlugs, 2)
	})

	t.Run("handles API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Incident not found"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "nonexistent",
				"title":      "Test",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update incident")
	})

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}
		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error decoding configuration")
	})
}

func Test__UpdateIncident__Configuration(t *testing.T) {
	component := &UpdateIncident{}
	config := component.Configuration()

	t.Run("has incident ID field as required", func(t *testing.T) {
		var found bool
		var isRequired bool
		for _, field := range config {
			if field.Name == "incidentId" {
				found = true
				isRequired = field.Required
				break
			}
		}

		require.True(t, found, "incidentId field should exist")
		assert.True(t, isRequired, "incidentId should be required")
	})

	t.Run("has optional status field with correct options", func(t *testing.T) {
		var found bool
		var isRequired bool
		var optionsCount int
		for _, field := range config {
			if field.Name == "status" {
				found = true
				isRequired = field.Required
				if field.TypeOptions != nil && field.TypeOptions.Select != nil {
					optionsCount = len(field.TypeOptions.Select.Options)
				}
				break
			}
		}

		require.True(t, found, "status field should exist")
		assert.False(t, isRequired, "status should be optional")
		assert.Equal(t, 8, optionsCount, "should have 8 status options")
	})
}

func Test__UpdateIncident__OutputChannels(t *testing.T) {
	component := &UpdateIncident{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func Test__UpdateIncident__Metadata(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("has correct name", func(t *testing.T) {
		assert.Equal(t, "rootly.updateIncident", component.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		assert.Equal(t, "Update Incident", component.Label())
	})

	t.Run("has description", func(t *testing.T) {
		assert.NotEmpty(t, component.Description())
	})

	t.Run("has documentation", func(t *testing.T) {
		doc := component.Documentation()
		assert.NotEmpty(t, doc)
		assert.Contains(t, doc, "Update Incident")
		assert.Contains(t, doc, "Use Cases")
		assert.Contains(t, doc, "Configuration")
	})

	t.Run("has icon", func(t *testing.T) {
		assert.Equal(t, "alert-triangle", component.Icon())
	})
}
