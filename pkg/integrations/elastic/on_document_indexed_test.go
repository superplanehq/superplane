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

func Test__OnDocumentIndexed__Setup(t *testing.T) {
	t.Run("missing index -> error", func(t *testing.T) {
		ctx := core.TriggerContext{
			Configuration: map[string]any{},
		}
		err := (&OnDocumentIndexed{}).Setup(ctx)
		require.ErrorContains(t, err, "index is required")
	})

	t.Run("valid config -> initializes metadata and schedules poll", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{"index": "my-index"},
			Metadata:      meta,
			Requests:      requests,
		}
		err := (&OnDocumentIndexed{}).Setup(ctx)
		require.NoError(t, err)
		assert.NotNil(t, meta.Metadata)
		assert.Equal(t, onDocumentIndexedPollAction, requests.Action)
	})

	t.Run("re-save preserves existing checkpoint", func(t *testing.T) {
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{LastTimestamp: "2024-01-01T00:00:00Z"}}
		requests := &contexts.RequestContext{}
		ctx := core.TriggerContext{
			Configuration: map[string]any{"index": "my-index"},
			Metadata:      meta,
			Requests:      requests,
		}
		err := (&OnDocumentIndexed{}).Setup(ctx)
		require.NoError(t, err)
		saved, ok := meta.Metadata.(OnDocumentIndexedMetadata)
		require.True(t, ok)
		assert.Equal(t, "2024-01-01T00:00:00Z", saved.LastTimestamp)
	})
}

func Test__OnDocumentIndexed__Poll(t *testing.T) {
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
		"url":      "https://elastic.example.com",
		"authType": "apiKey",
		"apiKey":   "test-api-key",
	}}

	t.Run("emits event for each new document", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"hits": {
							"hits": [
								{
									"_id": "doc-1",
									"_index": "my-index",
									"_source": {"@timestamp": "2024-06-01T12:01:00Z", "msg": "hello"}
								},
								{
									"_id": "doc-2",
									"_index": "my-index",
									"_source": {"@timestamp": "2024-06-01T12:02:00Z", "msg": "world"}
								}
							]
						}
					}`)),
				},
			},
		}
		events := &contexts.EventContext{}
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{LastTimestamp: "2024-06-01T12:00:00Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          onDocumentIndexedPollAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        events,
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, events.Count())
		assert.Equal(t, onDocumentIndexedPollAction, requests.Action)

		// Checkpoint advances to last document's timestamp
		saved := meta.Metadata.(OnDocumentIndexedMetadata)
		assert.Equal(t, "2024-06-01T12:02:00Z", saved.LastTimestamp)
	})

	t.Run("search uses correct URL and auth", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"hits":{"hits":[]}}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{LastTimestamp: "2024-01-01T00:00:00Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          onDocumentIndexedPollAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        &contexts.EventContext{},
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		req := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://elastic.example.com/my-index/_search", req.URL.String())
		assert.Equal(t, "ApiKey test-api-key", req.Header.Get("Authorization"))
	})

	t.Run("no new documents -> checkpoint unchanged", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"hits":{"hits":[]}}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{LastTimestamp: "2024-06-01T12:00:00Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          onDocumentIndexedPollAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        &contexts.EventContext{},
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		saved := meta.Metadata.(OnDocumentIndexedMetadata)
		assert.Equal(t, "2024-06-01T12:00:00Z", saved.LastTimestamp)
	})

	t.Run("Elasticsearch error -> schedules next poll without error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"internal"}`)),
				},
			},
		}
		meta := &contexts.MetadataContext{Metadata: OnDocumentIndexedMetadata{LastTimestamp: "2024-01-01T00:00:00Z"}}
		requests := &contexts.RequestContext{}

		_, err := (&OnDocumentIndexed{}).HandleAction(core.TriggerActionContext{
			Name:          onDocumentIndexedPollAction,
			Configuration: map[string]any{"index": "my-index"},
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Events:        &contexts.EventContext{},
			Metadata:      meta,
			Requests:      requests,
		})

		require.NoError(t, err)
		assert.Equal(t, onDocumentIndexedPollAction, requests.Action)
	})
}
