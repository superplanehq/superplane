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
