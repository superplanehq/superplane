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

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
			},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})
}

func Test__GetIncident__Execute(t *testing.T) {
	component := &GetIncident{}

	t.Run("successful get incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "abc123-def456",
							"type": "incidents",
							"attributes": {
								"sequential_id": 42,
								"title": "API latency spike",
								"slug": "api-latency-spike",
								"status": "started",
								"summary": "High latency detected",
								"severity": {"id": "sev-001", "name": "sev1"},
								"url": "https://app.rootly.com/incidents/abc123-def456",
								"started_at": "2026-01-19T12:00:00Z",
								"mitigated_at": null,
								"resolved_at": null,
								"user": {"id": "user-001", "full_name": "Jane Smith"},
								"started_by": {"id": "user-002", "full_name": "John Doe"}
							},
							"relationships": {
								"services": {
									"data": [
										{"type": "services", "id": "svc-001"}
									]
								},
								"groups": {
									"data": []
								},
								"events": {
									"data": [
										{"type": "incident_events", "id": "evt-001"}
									]
								},
								"action_items": {
									"data": null
								}
							}
						},
						"included": [
							{
								"id": "svc-001",
								"type": "services",
								"attributes": {
									"name": "Production API",
									"slug": "production-api"
								}
							},
							{
								"id": "evt-001",
								"type": "incident_events",
								"attributes": {
									"kind": "incident_created",
									"created_at": "2026-01-19T12:00:00Z"
								}
							}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "default", execCtx.Channel)
		assert.Equal(t, "rootly.incident", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)

		// Verify the request URL includes the include params
		require.Len(t, httpContext.Requests, 1)
		requestURL := httpContext.Requests[0].URL.String()
		assert.Contains(t, requestURL, "/incidents/abc123-def456")
		assert.Contains(t, requestURL, "include=services,groups,events,action_items")

		// Verify the emitted payload structure
		payload := execCtx.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "abc123-def456", data["id"])
		assert.Equal(t, "API latency spike", data["title"])
		assert.Equal(t, "started", data["status"])

		// Verify resolved services
		services := data["services"].([]map[string]any)
		require.Len(t, services, 1)
		assert.Equal(t, "svc-001", services[0]["id"])
		assert.Equal(t, "Production API", services[0]["name"])

		// Verify resolved events
		events := data["events"].([]map[string]any)
		require.Len(t, events, 1)
		assert.Equal(t, "evt-001", events[0]["id"])

		// Verify null relationship
		assert.Nil(t, data["action_items"])

		// Verify empty array relationship
		groups := data["groups"].([]map[string]any)
		assert.Len(t, groups, 0)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "nonexistent",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.False(t, execCtx.Finished)
	})

	t.Run("minimal response without includes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "inc-minimal",
							"type": "incidents",
							"attributes": {
								"title": "Minimal incident",
								"status": "started"
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execCtx := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "inc-minimal",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		require.Len(t, execCtx.Payloads, 1)

		payload := execCtx.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "inc-minimal", data["id"])
		assert.Equal(t, "Minimal incident", data["title"])
		assert.Equal(t, "started", data["status"])
	})
}
