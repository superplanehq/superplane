package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__applyStatusWithOptions(t *testing.T) {
	t.Run("posts transition body with comment and resolution when resolution is on screen", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"31","name":"Resolve","to":{"id":"10003","name":"Done"},"fields":{"resolution":{"required":false},"comment":{"required":false}}}]}`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = applyStatusWithOptions(client, "TEST-1", "Done", DoTransitionOptions{
			Comment:    "Ship it",
			Resolution: "Done",
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		// Confirms we request transitions.fields so the resolution check has data to work with.
		assert.Contains(t, httpContext.Requests[0].URL.String(), "expand=transitions.fields")

		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "31", payload["transition"].(map[string]any)["id"])
		assert.Equal(t, "Done", payload["fields"].(map[string]any)["resolution"].(map[string]any)["name"])
		assert.Contains(t, payload["update"].(map[string]any), "comment")
	})

	t.Run("prefers a transition whose screen exposes resolution when several reach the same status", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"transitions":[
						{"id":"41","name":"Close","to":{"id":"10003","name":"Done"}},
						{"id":"42","name":"Resolve","to":{"id":"10003","name":"Done"},"fields":{"resolution":{"required":false}}}
					]}`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = applyStatusWithOptions(client, "TEST-1", "Done", DoTransitionOptions{Resolution: "Done"})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 2)
		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "42", payload["transition"].(map[string]any)["id"])
	})

	t.Run("returns a clear error when resolution is requested but no transition exposes it", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"31","name":"Close","to":{"id":"10003","name":"Done"},"fields":{"summary":{"required":false}}}]}`)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = applyStatusWithOptions(client, "TEST-1", "Done", DoTransitionOptions{Resolution: "Done"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "transition to \"Done\" does not allow setting a resolution")
		assert.Contains(t, err.Error(), "Close")
		// Important: we did not call POST /transitions when the precheck fails — only the GET.
		require.Len(t, httpContext.Requests, 1)
	})

	t.Run("prefers a transition whose screen exposes comment when several reach the same status", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"transitions":[
						{"id":"41","name":"Close","to":{"id":"10003","name":"Done"}},
						{"id":"42","name":"Comment & Close","to":{"id":"10003","name":"Done"},"fields":{"comment":{"required":false}}}
					]}`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = applyStatusWithOptions(client, "TEST-1", "Done", DoTransitionOptions{Comment: "Closing"})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 2)
		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "42", payload["transition"].(map[string]any)["id"])
		assert.Contains(t, payload["update"].(map[string]any), "comment")
	})

	t.Run("attaches the comment optimistically when no transition screen lists it", func(t *testing.T) {
		// Jira accepts update.comment on most transitions even when the screen
		// metadata omits a comment field, so a comment must not block the move.
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"31","name":"Close","to":{"id":"10003","name":"Done"},"fields":{"summary":{"required":false}}}]}`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = applyStatusWithOptions(client, "TEST-1", "Done", DoTransitionOptions{Comment: "Closing"})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 2)
		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "31", payload["transition"].(map[string]any)["id"])
		assert.Contains(t, payload["update"].(map[string]any), "comment")
	})

	t.Run("uses the first matching transition when no fields are requested", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"31","name":"Close","to":{"id":"10003","name":"Done"}}]}`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = applyStatusWithOptions(client, "TEST-1", "Done", DoTransitionOptions{})
		require.NoError(t, err)
	})

	t.Run("returns helpful error when target status is unreachable", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"21","name":"Start","to":{"id":"10002","name":"In Progress"}}]}`)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = applyStatusWithOptions(client, "TEST-1", "Done", DoTransitionOptions{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), `"Done"`)
		assert.Contains(t, err.Error(), "In Progress")
	})
}

func Test__ApplyCompletionStatus(t *testing.T) {
	t.Run("defaults to a reachable Done-category status named Done", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"transitions":[
						{"id":"21","name":"Start","to":{"id":"10002","name":"In Progress","statusCategory":{"key":"indeterminate"}}},
						{"id":"31","name":"Resolve","to":{"id":"10003","name":"Done","statusCategory":{"key":"done"}},"fields":{"resolution":{"required":true}}}
					]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","name":"Done"}]`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = ApplyCompletionStatus(client, "TEST-1", "", DoTransitionOptions{Comment: "The SuperPlane task completed."})
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 3)
		assert.Contains(t, httpContext.Requests[1].URL.Path, "/resolution")

		body, err := io.ReadAll(httpContext.Requests[2].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "31", payload["transition"].(map[string]any)["id"])
		assert.Equal(t, "Done", payload["fields"].(map[string]any)["resolution"].(map[string]any)["name"])
	})

	t.Run("uses the chosen column name", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"transitions":[
						{"id":"21","name":"Start QA","to":{"id":"10004","name":"QA","statusCategory":{"key":"indeterminate"}}}
					]}`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = ApplyCompletionStatus(client, "TEST-1", "QA", DoTransitionOptions{})
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "21", payload["transition"].(map[string]any)["id"])
	})

	t.Run("returns an error when the chosen column is unreachable", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"21","name":"Start","to":{"id":"10002","name":"In Progress","statusCategory":{"key":"indeterminate"}}}]}`)),
				},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = ApplyCompletionStatus(client, "TEST-1", "Done", DoTransitionOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"Done"`)
	})
}

func Test__StatusUnmarshalJSON(t *testing.T) {
	var nested Status
	require.NoError(t, json.Unmarshal([]byte(`{"id":"1","name":"Done","statusCategory":{"key":"done"}}`), &nested))
	assert.Equal(t, "DONE", nested.Category)

	var flat Status
	require.NoError(t, json.Unmarshal([]byte(`{"id":"1","name":"Done","statusCategory":"DONE"}`), &flat))
	assert.Equal(t, "DONE", flat.Category)
}

func Test__IssueAlreadyInColumn(t *testing.T) {
	issue := &Issue{Fields: map[string]any{
		"status": map[string]any{
			"name":           "Done",
			"statusCategory": map[string]any{"key": "done"},
		},
	}}
	assert.True(t, IssueAlreadyInColumn(issue, ""))
	assert.True(t, IssueAlreadyInColumn(issue, "Done"))
	assert.False(t, IssueAlreadyInColumn(issue, "QA"))
}

func Test__unmarshalWebhookPayloads(t *testing.T) {
	t.Run("wraps a single object", func(t *testing.T) {
		payloads, err := unmarshalWebhookPayloads[IssueWebhookPayload]([]byte(
			`{"webhookEvent":"jira:issue_created","issue":{"key":"ENG-1"}}`,
		))
		require.NoError(t, err)
		require.Len(t, payloads, 1)
		assert.Equal(t, issueEventCreated, payloads[0].WebhookEvent)
		require.NotNil(t, payloads[0].Issue)
		assert.Equal(t, "ENG-1", payloads[0].Issue.Key)
	})

	t.Run("parses a JSON array", func(t *testing.T) {
		payloads, err := unmarshalWebhookPayloads[IssueWebhookPayload]([]byte(`[
			{"webhookEvent":"jira:issue_created","issue":{"key":"ENG-1"}},
			{"webhookEvent":"jira:issue_updated","issue":{"key":"ENG-2"}}
		]`))
		require.NoError(t, err)
		require.Len(t, payloads, 2)
		assert.Equal(t, "ENG-1", payloads[0].Issue.Key)
		assert.Equal(t, "ENG-2", payloads[1].Issue.Key)
	})

	t.Run("rejects an empty body", func(t *testing.T) {
		_, err := unmarshalWebhookPayloads[IssueWebhookPayload]([]byte("  "))
		require.ErrorContains(t, err, "request body is empty")
	})
}
