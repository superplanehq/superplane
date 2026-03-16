package elastic

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnCaseStatusChange__Setup(t *testing.T) {
	t.Run("valid config -> initializes metadata and schedules poll", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      meta,
			Requests:      requests,
		}
		err := (&OnCaseStatusChange{}).Setup(ctx)
		require.NoError(t, err)
		assert.NotNil(t, meta.Metadata)
		assert.Equal(t, onCaseStatusChangePollAction, requests.Action)
	})

	t.Run("re-save preserves existing checkpoint", func(t *testing.T) {
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z"}}
		requests := &contexts.RequestContext{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{},
			Metadata:      meta,
			Requests:      requests,
		}
		err := (&OnCaseStatusChange{}).Setup(ctx)
		require.NoError(t, err)
		saved, ok := meta.Metadata.(OnCaseStatusChangeMetadata)
		require.True(t, ok)
		assert.Equal(t, "2024-01-01T00:00:00Z", saved.LastPollTime)
	})
}

func Test__OnCaseStatusChange__Poll(t *testing.T) {
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"url":       "https://elastic.example.com",
		"kibanaUrl": "https://kibana.example.com",
		"authType":  "apiKey",
		"apiKey":    "test-api-key",
	}}

	casesResponse := `{
		"cases": [
			{
				"id": "case-1",
				"title": "Production incident",
				"status": "in-progress",
				"severity": "high",
				"version": "WzE3LDFd",
				"tags": ["prod"],
				"description": "Error rate spike",
				"created_at": "2024-06-01T10:00:00.000Z",
				"updated_at": "2024-06-01T12:01:00.000Z"
			},
			{
				"id": "case-2",
				"title": "DB issue",
				"status": "closed",
				"severity": "low",
				"version": "WzE4LDFd",
				"tags": [],
				"description": "Resolved",
				"created_at": "2024-06-01T09:00:00.000Z",
				"updated_at": "2024-06-01T12:02:00.000Z"
			}
		]
	}`

	t.Run("emits event for each updated case", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(casesResponse)),
				},
			},
		}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:          onCaseStatusChangePollAction,
			Configuration: map[string]any{},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        events,
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, events.Count())
		assert.Equal(t, onCaseStatusChangePollAction, requests.Action)

		saved := meta.Metadata.(OnCaseStatusChangeMetadata)
		assert.Equal(t, "2024-06-01T12:02:00.000Z", saved.LastPollTime)
	})

	t.Run("poll uses correct Kibana URL and auth", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"cases":[]}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:          onCaseStatusChangePollAction,
			Configuration: map[string]any{},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        &contexts.EventContext{},
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Contains(t, req.URL.String(), "kibana.example.com")
		assert.Contains(t, req.URL.String(), "/api/cases")
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))
	})

	t.Run("status filter: only matching statuses emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(casesResponse)),
				},
			},
		}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00.000Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:          onCaseStatusChangePollAction,
			Configuration: map[string]any{"statuses": []string{"closed"}},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        events,
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
		assert.Equal(t, "elastic.case.status.changed", events.Payloads[0].Type)
		data := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "closed", data["status"])
	})

	t.Run("no updated cases -> checkpoint unchanged", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"cases":[]}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-06-01T12:00:00Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:          onCaseStatusChangePollAction,
			Configuration: map[string]any{},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        &contexts.EventContext{},
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		saved := meta.Metadata.(OnCaseStatusChangeMetadata)
		assert.Equal(t, "2024-06-01T12:00:00Z", saved.LastPollTime)
	})

	t.Run("Kibana error -> schedules next poll without error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"internal"}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{Metadata: OnCaseStatusChangeMetadata{LastPollTime: "2024-01-01T00:00:00Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnCaseStatusChange{}).HandleAction(core.TriggerActionContext{
			Name:          onCaseStatusChangePollAction,
			Configuration: map[string]any{},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        &contexts.EventContext{},
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		assert.Equal(t, onCaseStatusChangePollAction, requests.Action)
	})
}
