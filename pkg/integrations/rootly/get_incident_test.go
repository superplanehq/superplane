package rootly

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetIncident__Setup(t *testing.T) {
	component := &GetIncident{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("missing incidentId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
			},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "abc123-def456-789ghi",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetIncident__Execute(t *testing.T) {
	component := &GetIncident{}

	t.Run("successful incident retrieval", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "abc123-def456-789ghi",
							"type": "incidents",
							"attributes": {
								"title": "Database connection timeout",
								"summary": "Users experiencing slow response times",
								"status": "investigating",
								"severity": "sev1",
								"started_at": "2024-02-11T10:00:00Z",
								"resolved_at": null,
								"mitigated_at": "2024-02-11T10:30:00Z",
								"url": "https://app.rootly.com/incidents/abc123"
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
				"incidentId": "abc123-def456-789ghi",
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
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://api.rootly.com/v1/incidents/abc123-def456-789ghi", req.URL.String())
		assert.Equal(t, "application/vnd.api+json", req.Header.Get("Content-Type"))
		assert.Equal(t, "application/vnd.api+json", req.Header.Get("Accept"))
		assert.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"Incident not found"}`)),
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
				"incidentId": "nonexistent-incident",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.ErrorContains(t, err, "failed to get incident")
	})

	t.Run("missing API key -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{}, // No API key
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "abc123-def456-789ghi",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.ErrorContains(t, err, "error creating client")
	})
}

func Test__GetIncident__Name(t *testing.T) {
	component := &GetIncident{}
	assert.Equal(t, "rootly.getIncident", component.Name())
}

func Test__GetIncident__Label(t *testing.T) {
	component := &GetIncident{}
	assert.Equal(t, "Get Incident", component.Label())
}

func Test__GetIncident__Description(t *testing.T) {
	component := &GetIncident{}
	assert.Equal(t, "Retrieve a single incident from Rootly by ID", component.Description())
}

func Test__GetIncident__Icon(t *testing.T) {
	component := &GetIncident{}
	assert.Equal(t, "search", component.Icon())
}

func Test__GetIncident__Color(t *testing.T) {
	component := &GetIncident{}
	assert.Equal(t, "blue", component.Color())
}

func Test__GetIncident__Configuration(t *testing.T) {
	component := &GetIncident{}
	config := component.Configuration()

	assert.Len(t, config, 1)
	assert.Equal(t, "incidentId", config[0].Name)
	assert.Equal(t, "Incident ID", config[0].Label)
	assert.True(t, config[0].Required)
}

func Test__GetIncident__OutputChannels(t *testing.T) {
	component := &GetIncident{}
	channels := component.OutputChannels(nil)
	
	assert.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__GetIncident__Documentation(t *testing.T) {
	component := &GetIncident{}
	doc := component.Documentation()
	
	assert.NotEmpty(t, doc)
	assert.Contains(t, doc, "Get Incident component")
	assert.Contains(t, doc, "Use Cases")
	assert.Contains(t, doc, "Configuration")
	assert.Contains(t, doc, "Output")
	assert.Contains(t, doc, "Examples")
}