package linear

import (
	"bytes"
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

const (
	testAccessToken  = "lin_oauth_test_access_token"
	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
)

func newAuthorizedIntegration() *contexts.IntegrationContext {
	return newAuthorizedIntegrationWithMetadata(Metadata{})
}

func newAuthorizedIntegrationWithMetadata(metadata Metadata) *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"clientId":     testClientID,
			"clientSecret": testClientSecret,
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			OAuthAccessToken:  {Name: OAuthAccessToken, Value: []byte(testAccessToken)},
			OAuthRefreshToken: {Name: OAuthRefreshToken, Value: []byte("lin_oauth_test_refresh_token")},
		},
		Metadata: metadata,
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func Test__NewClient(t *testing.T) {
	t.Run("no access token secret -> error", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{}}

		_, err := NewClient(&contexts.HTTPContext{}, integration)
		require.ErrorContains(t, err, "missing Linear access token")
	})

	t.Run("blank access token -> error", func(t *testing.T) {
		integration := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("   ")},
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, integration)
		require.ErrorContains(t, err, "missing Linear access token")
	})

	t.Run("nil HTTP context -> error", func(t *testing.T) {
		_, err := NewClient(nil, newAuthorizedIntegration())
		require.ErrorContains(t, err, "missing HTTP context")
	})

	t.Run("valid configuration", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, newAuthorizedIntegration())
		require.NoError(t, err)
		assert.Equal(t, testAccessToken, client.AccessToken)
	})
}

func Test__Client__SendsBearerToken(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"viewer":{"id":"u1","name":"Jane"},"organization":{"id":"o1","name":"Acme","urlKey":"acme"}}}`),
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	viewer, err := client.GetViewer()
	require.NoError(t, err)
	assert.Equal(t, "Jane", viewer.User.Name)
	assert.Equal(t, "acme", viewer.Organization.URLKey)

	require.Len(t, httpContext.Requests, 1)
	request := httpContext.Requests[0]

	// OAuth tokens require the Bearer prefix - unlike personal API keys.
	assert.Equal(t, "Bearer "+testAccessToken, request.Header.Get("Authorization"))
	assert.Equal(t, APIURL, request.URL.String())
	assert.Equal(t, http.MethodPost, request.Method)
}

func Test__Client__ReturnsGraphQLErrors(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"errors":[{"message":"Authentication required"}]}`),
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	_, err = client.GetViewer()
	require.ErrorContains(t, err, "Authentication required")
}

func Test__Client__ListTeams(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t1","key":"ENG","name":"Engineering"}]}}}`),
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	teams, err := client.ListTeams()
	require.NoError(t, err)
	require.Len(t, teams, 1)
	assert.Equal(t, "ENG", teams[0].Key)
}

// variablesFromRequest returns the GraphQL variables sent on the nth request.
// Reading a request body consumes it, so the body is put back to keep the
// helper safe to call more than once for the same request.
func variablesFromRequest(t *testing.T, httpContext *contexts.HTTPContext, n int) map[string]any {
	t.Helper()

	require.Greater(t, len(httpContext.Requests), n)
	body := readAndRestoreBody(t, httpContext.Requests[n])

	payload := struct {
		Variables map[string]any `json:"variables"`
	}{}

	require.NoError(t, json.Unmarshal(body, &payload))
	return payload.Variables
}

// queryFromRequest returns the GraphQL query document sent on the nth request.
func queryFromRequest(t *testing.T, httpContext *contexts.HTTPContext, n int) string {
	t.Helper()

	require.Greater(t, len(httpContext.Requests), n)
	body := readAndRestoreBody(t, httpContext.Requests[n])

	payload := struct {
		Query string `json:"query"`
	}{}

	require.NoError(t, json.Unmarshal(body, &payload))
	return payload.Query
}

func readAndRestoreBody(t *testing.T, request *http.Request) []byte {
	t.Helper()

	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	request.Body = io.NopCloser(bytes.NewReader(body))

	return body
}

func Test__Client__Pagination(t *testing.T) {
	t.Run("follows the cursor across pages", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t1","key":"ONE","name":"One"}],"pageInfo":{"hasNextPage":true,"endCursor":"cursor-1"}}}}`),
				jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t2","key":"TWO","name":"Two"}],"pageInfo":{"hasNextPage":true,"endCursor":"cursor-2"}}}}`),
				jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t3","key":"THREE","name":"Three"}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		teams, err := client.ListTeams()
		require.NoError(t, err)

		// Every page is merged, not just the first.
		require.Len(t, teams, 3)
		assert.Equal(t, []string{"ONE", "TWO", "THREE"}, []string{teams[0].Key, teams[1].Key, teams[2].Key})

		require.Len(t, httpContext.Requests, 3)
		assert.NotContains(t, variablesFromRequest(t, httpContext, 0), "after", "first page must not send a cursor")
		assert.Equal(t, "cursor-1", variablesFromRequest(t, httpContext, 1)["after"])
		assert.Equal(t, "cursor-2", variablesFromRequest(t, httpContext, 2)["after"])
	})

	t.Run("stops after a single page when there is no next page", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t1","key":"ONE","name":"One"}],"pageInfo":{"hasNextPage":false,"endCursor":"c1"}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		teams, err := client.ListTeams()
		require.NoError(t, err)
		assert.Len(t, teams, 1)
		assert.Len(t, httpContext.Requests, 1)
	})

	t.Run("stops when hasNextPage is true but the cursor is empty", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t1","key":"ONE","name":"One"}],"pageInfo":{"hasNextPage":true,"endCursor":""}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		teams, err := client.ListTeams()
		require.NoError(t, err)
		assert.Len(t, teams, 1)
		assert.Len(t, httpContext.Requests, 1, "an empty cursor must not trigger another page")
	})

	t.Run("gives up rather than looping forever on a stuck cursor", func(t *testing.T) {
		responses := make([]*http.Response, 0, maxPages+1)
		for range maxPages + 1 {
			responses = append(responses, jsonResponse(`{"data":{"teams":{"nodes":[{"id":"t1","key":"ONE","name":"One"}],"pageInfo":{"hasNextPage":true,"endCursor":"same-cursor"}}}}`))
		}

		client, err := NewClient(&contexts.HTTPContext{Responses: responses}, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.ListTeams()
		require.ErrorContains(t, err, "gave up paginating")
	})

	t.Run("paginates team members", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"team":{"members":{"nodes":[{"id":"u1","name":"One"}],"pageInfo":{"hasNextPage":true,"endCursor":"c1"}}}}}`),
				jsonResponse(`{"data":{"team":{"members":{"nodes":[{"id":"u2","name":"Two"}],"pageInfo":{"hasNextPage":false}}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		members, err := client.ListTeamMembers("t1")
		require.NoError(t, err)
		require.Len(t, members, 2)
		assert.Equal(t, "c1", variablesFromRequest(t, httpContext, 1)["after"])
		assert.Equal(t, "t1", variablesFromRequest(t, httpContext, 1)["teamId"], "filter variables persist across pages")
	})

	t.Run("paginates labels", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueLabels":{"nodes":[{"id":"l1","name":"bug"}],"pageInfo":{"hasNextPage":true,"endCursor":"c1"}}}}`),
				jsonResponse(`{"data":{"issueLabels":{"nodes":[{"id":"l2","name":"chore"}],"pageInfo":{"hasNextPage":false}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		labels, err := client.ListLabels("t1")
		require.NoError(t, err)
		assert.Len(t, labels, 2)
	})

	t.Run("paginates team projects", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"team":{"projects":{"nodes":[{"id":"p1","name":"One"}],"pageInfo":{"hasNextPage":true,"endCursor":"c1"}}}}}`),
				jsonResponse(`{"data":{"team":{"projects":{"nodes":[{"id":"p2","name":"Two"}],"pageInfo":{"hasNextPage":false}}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		projects, err := client.ListTeamProjects("t1")
		require.NoError(t, err)
		require.Len(t, projects, 2)
		assert.Equal(t, "c1", variablesFromRequest(t, httpContext, 1)["after"])
	})

	t.Run("paginates workflow states", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"workflowStates":{"nodes":[{"id":"s1","name":"Todo"}],"pageInfo":{"hasNextPage":true,"endCursor":"c1"}}}}`),
				jsonResponse(`{"data":{"workflowStates":{"nodes":[{"id":"s2","name":"Done"}],"pageInfo":{"hasNextPage":false}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		states, err := client.ListWorkflowStates("t1")
		require.NoError(t, err)
		assert.Len(t, states, 2)

		// Duplicate-type states are not valid issueCreate targets, so the
		// query must keep excluding them from the status picker.
		query := queryFromRequest(t, httpContext, 0)
		assert.Contains(t, query, `type: { neq: "duplicate" }`)
	})

	t.Run("missing team is still reported", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(`{"data":{"team":null}}`)},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.ListTeamMembers("nope")
		require.ErrorContains(t, err, "team nope not found")
	})
}

func Test__Client__CreateIssue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueCreate":{"success":true,"issue":{"id":"i1","identifier":"ENG-1","title":"Boom","labels":{"nodes":[{"id":"l1","name":"bug"}]}}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		issue, err := client.CreateIssue(map[string]any{"teamId": "t1", "title": "Boom"})
		require.NoError(t, err)
		assert.Equal(t, "ENG-1", issue.Identifier)

		// The labels connection is flattened for the emitted payload.
		require.Len(t, issue.Labels, 1)
		assert.Equal(t, "bug", issue.Labels[0].Name)
	})

	t.Run("unsuccessful response -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issueCreate":{"success":false,"issue":null}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateIssue(map[string]any{"teamId": "t1", "title": "Boom"})
		require.ErrorContains(t, err, "not created")
	})
}

func Test__Client__CreateWebhook(t *testing.T) {
	t.Run("scoped to a team", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"webhookCreate":{"success":true,"webhook":{"id":"w1","url":"https://sp.test/hook"}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		webhook, err := client.CreateWebhook("https://sp.test/hook", "s3cr3t", "SuperPlane", "t1", []string{IssueResourceType})
		require.NoError(t, err)
		assert.Equal(t, "w1", webhook.ID)

		input := webhookInputFromRequest(t, httpContext)
		assert.Equal(t, "t1", input["teamId"])
		assert.Equal(t, "s3cr3t", input["secret"])
		assert.Equal(t, []any{IssueResourceType}, input["resourceTypes"])
		assert.NotContains(t, input, "allPublicTeams")
	})

	t.Run("falls back to all public teams", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"webhookCreate":{"success":true,"webhook":{"id":"w1","url":"https://sp.test/hook"}}}}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateWebhook("https://sp.test/hook", "s3cr3t", "SuperPlane", "", []string{IssueResourceType})
		require.NoError(t, err)

		input := webhookInputFromRequest(t, httpContext)
		assert.Equal(t, true, input["allPublicTeams"])
		assert.NotContains(t, input, "teamId")
	})

	t.Run("admin permission missing -> surfaces Linear error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"You don't have permission to do this"}]}`),
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateWebhook("https://sp.test/hook", "s3cr3t", "SuperPlane", "t1", []string{IssueResourceType})
		require.ErrorContains(t, err, "You don't have permission to do this")
	})
}

func Test__Client__DeleteWebhook(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"webhookDelete":{"success":true}}}`),
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	require.NoError(t, client.DeleteWebhook("w1"))
}

func webhookInputFromRequest(t *testing.T, httpContext *contexts.HTTPContext) map[string]any {
	t.Helper()

	require.Len(t, httpContext.Requests, 1)
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	payload := struct {
		Variables struct {
			Input map[string]any `json:"input"`
		} `json:"variables"`
	}{}

	require.NoError(t, json.Unmarshal(body, &payload))
	return payload.Variables.Input
}
