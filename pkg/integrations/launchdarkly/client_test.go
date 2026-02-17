package launchdarkly

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__resolveAPIBaseURL(t *testing.T) {
	t.Run("empty -> default US endpoint", func(t *testing.T) {
		assert.Equal(t, "https://app.launchdarkly.com/api/v2", resolveAPIBaseURL(""))
	})

	t.Run("EU host -> appends /api/v2", func(t *testing.T) {
		assert.Equal(t, "https://app.eu.launchdarkly.com/api/v2", resolveAPIBaseURL("https://app.eu.launchdarkly.com"))
	})

	t.Run("already has /api/v2 -> preserved", func(t *testing.T) {
		assert.Equal(t, "https://app.eu.launchdarkly.com/api/v2", resolveAPIBaseURL("https://app.eu.launchdarkly.com/api/v2/"))
	})
}

func Test__Client__GetFlag__encodesPathAndSendsToken(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"key":"my flag"}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiAccessToken": "token-123",
			"apiBaseUrl":     "https://app.eu.launchdarkly.com",
		},
	}

	client, err := NewClient(httpCtx, integrationCtx)
	require.NoError(t, err)

	statusCode, payload, _, err := client.GetFlag("my project", "my flag")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Equal(t, "my flag", payload["key"])

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://app.eu.launchdarkly.com/api/v2/flags/my%20project/my%20flag", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "token-123", httpCtx.Requests[0].Header.Get("Authorization"))
}

func Test__Client__DeleteFlag__encodesPath(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiAccessToken": "token-123",
		},
	}

	client, err := NewClient(httpCtx, integrationCtx)
	require.NoError(t, err)

	statusCode, _, err := client.DeleteFlag("my project", "my flag")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, statusCode)

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://app.launchdarkly.com/api/v2/flags/my%20project/my%20flag", httpCtx.Requests[0].URL.String())
}
