package posthog

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NormalizeHost(t *testing.T) {
	t.Run("empty falls back to US cloud", func(t *testing.T) {
		assert.Equal(t, DefaultHost, NormalizeHost(""))
		assert.Equal(t, DefaultHost, NormalizeHost("   "))
	})

	t.Run("trailing slashes are trimmed", func(t *testing.T) {
		assert.Equal(t, "https://eu.posthog.com", NormalizeHost("https://eu.posthog.com/"))
		assert.Equal(t, "https://posthog.internal", NormalizeHost("  https://posthog.internal//  "))
	})

	t.Run("already normalized host is unchanged", func(t *testing.T) {
		assert.Equal(t, "https://eu.posthog.com", NormalizeHost("https://eu.posthog.com"))
	})
}

func Test__NewClient(t *testing.T) {
	t.Run("missing API key returns error", func(t *testing.T) {
		_, err := NewClient(&contexts.HTTPContext{}, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "  "},
		})

		require.ErrorContains(t, err, "personal API key is required")
	})

	t.Run("missing host falls back to US cloud", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "phx_test"},
		})

		require.NoError(t, err)
		assert.Equal(t, DefaultHost, client.Host)
	})

	t.Run("configured host is used", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "phx_test", "host": "https://eu.posthog.com/"},
		})

		require.NoError(t, err)
		assert.Equal(t, "https://eu.posthog.com", client.Host)
	})
}

func Test__Client__Query(t *testing.T) {
	t.Run("sends HogQL query and parses response", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"results":[["pageview","user_1"]],"columns":["event","distinct_id"],` +
							`"types":[["event","String"],["distinct_id","String"]],` +
							`"hogql":"SELECT event, distinct_id FROM events LIMIT 1","hasMore":false}`,
					)),
				},
			},
		}

		client, err := NewClient(httpContext, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "phx_test", "host": "https://eu.posthog.com"},
		})
		require.NoError(t, err)

		response, err := client.Query("12345", QueryRequest{Query: "SELECT event, distinct_id FROM events LIMIT 1"})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://eu.posthog.com/api/projects/12345/query/", request.URL.String())
		assert.Equal(t, "Bearer phx_test", request.Header.Get("Authorization"))

		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		assert.JSONEq(t,
			`{"query":{"kind":"HogQLQuery","query":"SELECT event, distinct_id FROM events LIMIT 1"}}`,
			string(body),
		)

		assert.Equal(t, []string{"event", "distinct_id"}, response.Columns)
		require.Len(t, response.Results, 1)
	})

	t.Run("placeholder values are sent with the query", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"results":[],"columns":[]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "phx_test"},
		})
		require.NoError(t, err)

		_, err = client.Query("12345", QueryRequest{
			Query:  "SELECT event FROM events WHERE event IN {events}",
			Values: map[string]any{"events": []string{"signed up"}},
		})
		require.NoError(t, err)

		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.JSONEq(t,
			`{"query":{"kind":"HogQLQuery","query":"SELECT event FROM events WHERE event IN {events}",`+
				`"values":{"events":["signed up"]}}}`,
			string(body),
		)
	})

	t.Run("no values are sent when there are none", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"results":[],"columns":[]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "phx_test"},
		})
		require.NoError(t, err)

		_, err = client.Query("12345", QueryRequest{Query: "SELECT 1"})
		require.NoError(t, err)

		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"query":{"kind":"HogQLQuery","query":"SELECT 1"}}`, string(body))
	})

	t.Run("error status is surfaced as APIError", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"detail":"Syntax error"}`)),
				},
			},
		}

		client, err := NewClient(httpContext, &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "phx_test"},
		})
		require.NoError(t, err)

		_, err = client.Query("12345", QueryRequest{Query: "NOT HOGQL"})
		require.Error(t, err)

		apiErr := &APIError{}
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	})
}

func Test__RowsToMaps(t *testing.T) {
	t.Run("keys each row by column name", func(t *testing.T) {
		rows := RowsToMaps(&QueryResponse{
			Columns: []string{"event", "distinct_id"},
			Results: [][]any{
				{"pageview", "user_1"},
				{"signup", "user_2"},
			},
		})

		require.Len(t, rows, 2)
		assert.Equal(t, map[string]any{"event": "pageview", "distinct_id": "user_1"}, rows[0])
		assert.Equal(t, map[string]any{"event": "signup", "distinct_id": "user_2"}, rows[1])
	})

	t.Run("unnamed columns fall back to their position", func(t *testing.T) {
		rows := RowsToMaps(&QueryResponse{
			Columns: []string{"event", ""},
			Results: [][]any{{"pageview", 12, true}},
		})

		require.Len(t, rows, 1)
		assert.Equal(t, map[string]any{"event": "pageview", "column_1": 12, "column_2": true}, rows[0])
	})

	t.Run("no results yields an empty slice", func(t *testing.T) {
		rows := RowsToMaps(&QueryResponse{Columns: []string{"event"}})
		assert.Empty(t, rows)
	})
}
