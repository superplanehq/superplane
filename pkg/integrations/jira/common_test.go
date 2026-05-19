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
	t.Run("posts transition body with comment and resolution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"transitions":[{"id":"31","name":"Resolve","to":{"id":"10003","name":"Done"}}]}`)),
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

		body, err := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(body, &payload))
		assert.Equal(t, "31", payload["transition"].(map[string]any)["id"])
		assert.Equal(t, "Done", payload["fields"].(map[string]any)["resolution"].(map[string]any)["name"])
		assert.Contains(t, payload["update"].(map[string]any), "comment")
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
