package linear

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

const testAPIKey = "lin_api_test_key"

func newAuthorizedIntegration() *contexts.IntegrationContext {
	return newAuthorizedIntegrationWithMetadata(Metadata{})
}

func newAuthorizedIntegrationWithMetadata(metadata Metadata) *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": testAPIKey},
		Metadata:      metadata,
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func Test__NewClient(t *testing.T) {
	t.Run("missing API key -> error", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{}}

		_, err := NewClient(&contexts.HTTPContext{}, integration)
		require.ErrorContains(t, err, "error reading API key")
	})

	t.Run("blank API key -> error", func(t *testing.T) {
		integration := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "   "}}

		_, err := NewClient(&contexts.HTTPContext{}, integration)
		require.ErrorContains(t, err, "missing Linear API key")
	})

	t.Run("valid configuration", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, newAuthorizedIntegration())
		require.NoError(t, err)
		assert.Equal(t, testAPIKey, client.APIKey)
	})
}

func Test__Client__SendsRawAPIKey(t *testing.T) {
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

	// Linear expects the bare key - a "Bearer " prefix is rejected.
	assert.Equal(t, testAPIKey, request.Header.Get("Authorization"))
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
