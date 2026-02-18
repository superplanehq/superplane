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

	t.Run("valid configuration with all fields", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
				"title":      "Updated title",
				"summary":    "Updated summary",
				"status":     "mitigated",
				"subStatus":  "sub-status-uuid-1",
				"severity":   "sev-uuid-123",
				"services":   []string{"svc-uuid-1"},
				"teams":      []string{"team-uuid-1"},
				"labels": []map[string]any{
					{"key": "env", "value": "production"},
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"title": "Updated title",
			},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"incidentId": "",
				"title":      "Updated title",
			},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("no update fields returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
			},
		})

		require.ErrorContains(t, err, "at least one field to update must be provided")
	})

	t.Run("incidentId with title only", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
				"title":      "New title",
			},
		})

		require.NoError(t, err)
	})

	t.Run("incidentId with status only", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
				"status":     "resolved",
			},
		})

		require.NoError(t, err)
	})

	t.Run("incidentId with subStatus only", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata: &contexts.MetadataContext{},
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
				"subStatus":  "sub-status-uuid-1",
			},
		})

		require.NoError(t, err)
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})
}

func Test__UpdateIncident__Execute(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("successful update emits incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "inc-uuid-123",
							"type": "incidents",
							"attributes": {
								"title": "Updated Incident",
								"sequential_id": 42,
								"slug": "updated-incident",
								"summary": "Updated summary",
								"status": "mitigated",
								"severity": "sev1",
								"started_at": "2026-01-19T12:00:00Z",
								"mitigated_at": "2026-01-19T13:30:00Z",
								"updated_at": "2026-01-19T13:30:00Z",
								"url": "https://app.rootly.com/incidents/inc-uuid-123"
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

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "inc-uuid-123",
				"title":      "Updated Incident",
				"summary":    "Updated summary",
				"status":     "mitigated",
				"subStatus":  "sub-status-uuid-1",
				"severity":   "sev-uuid-1",
				"services":   []string{"svc-uuid-1", "svc-uuid-2"},
				"teams":      []string{"team-uuid-1"},
				"labels": []map[string]any{
					{"key": "env", "value": "production"},
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "rootly.incident", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		// Verify request
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPatch, req.Method)
		assert.Contains(t, req.URL.String(), "/incidents/inc-uuid-123")
		assert.Equal(t, "application/vnd.api+json", req.Header.Get("Content-Type"))

		// Verify request body
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(body, &reqBody))

		data := reqBody["data"].(map[string]any)
		assert.Equal(t, "inc-uuid-123", data["id"])
		assert.Equal(t, "incidents", data["type"])

		attrs := data["attributes"].(map[string]any)
		assert.Equal(t, "Updated Incident", attrs["title"])
		assert.Equal(t, "Updated summary", attrs["summary"])
		assert.Equal(t, "mitigated", attrs["status"])
		assert.Equal(t, "sub-status-uuid-1", attrs["sub_status_id"])
		assert.Equal(t, "sev-uuid-1", attrs["severity_id"])

		serviceIDs := attrs["service_ids"].([]any)
		assert.Len(t, serviceIDs, 2)
		assert.Equal(t, "svc-uuid-1", serviceIDs[0])
		assert.Equal(t, "svc-uuid-2", serviceIDs[1])

		groupIDs := attrs["group_ids"].([]any)
		assert.Len(t, groupIDs, 1)
		assert.Equal(t, "team-uuid-1", groupIDs[0])

		labels := attrs["labels"].(map[string]any)
		assert.Equal(t, "production", labels["env"])
	})

	t.Run("API error returns error and does not emit", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errors": [{"title": "Record not found"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "nonexistent-id",
				"title":      "Updated title",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update incident")
		assert.False(t, execState.Passed)
		assert.Empty(t, execState.Channel)
	})
}
