package logfire

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestClient_NewClient_UsesRegionFromAPIKeyAndSecret(t *testing.T) {
	t.Parallel()

	ctx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey":  "lf_api_eu_123",
			"baseURL": "   ",
		},
		Secrets: map[string]core.IntegrationSecret{
			readTokenSecretName: {Name: readTokenSecretName, Value: []byte("read-token")},
		},
	}

	client, err := NewClient(&contexts.HTTPContext{}, ctx)
	require.NoError(t, err)

	assert.Equal(t, "lf_api_eu_123", client.APIKey)
	assert.Equal(t, "read-token", client.ReadToken)
	assert.Equal(t, logfireEUBaseURL, client.BaseURL)
	assert.Equal(t, logfireEUAPIBaseURL, client.APIBaseURL)
}

func TestClient_ValidateCredentials(t *testing.T) {
	t.Parallel()

	t.Run("missing read token", func(t *testing.T) {
		t.Parallel()

		client := &Client{}
		err := client.ValidateCredentials()
		require.ErrorContains(t, err, "logfire read token is required")
	})

	t.Run("unauthorized token", func(t *testing.T) {
		t.Parallel()

		client := &Client{
			BaseURL:   logfireUSBaseURL,
			ReadToken: "invalid",
			http: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"error":"unauthorized"}`))},
				},
			},
		}

		err := client.ValidateCredentials()
		require.ErrorContains(t, err, "invalid Logfire read token")
	})
}

func TestClient_ExecuteQuery_SetsAuthAndQueryParams(t *testing.T) {
	t.Parallel()

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"columns":[],"rows":[]}`))},
		},
	}

	client := &Client{
		BaseURL:   "https://logfire-us.pydantic.dev",
		ReadToken: "rt_123",
		http:      httpCtx,
	}

	_, err := client.ExecuteQuery(QueryRequest{
		SQL:          "SELECT 1",
		MinTimestamp: "2025-01-01T00:00:00Z",
		MaxTimestamp: "2025-01-01T23:59:59Z",
		Limit:        10,
		RowOriented:  true,
	})
	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)

	req := httpCtx.Requests[0]
	assert.Equal(t, "Bearer rt_123", req.Header.Get("Authorization"))
	assert.Equal(t, "/v1/query", req.URL.Path)
	assert.Equal(t, "SELECT 1", req.URL.Query().Get("sql"))
	assert.Equal(t, "2025-01-01T00:00:00Z", req.URL.Query().Get("min_timestamp"))
	assert.Equal(t, "2025-01-01T23:59:59Z", req.URL.Query().Get("max_timestamp"))
	assert.Equal(t, "10", req.URL.Query().Get("limit"))
	assert.Equal(t, "true", req.URL.Query().Get("row_oriented"))
}

func TestClient_CreateReadToken_EmptyTokenReturned(t *testing.T) {
	t.Parallel()

	client := &Client{
		APIKey:     "api_123",
		APIBaseURL: logfireUSAPIBaseURL,
		http: &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"token":"   "}`))},
			},
		},
	}

	_, err := client.CreateReadToken("project_1", defaultReadTokenName)
	require.ErrorContains(t, err, "read token not returned by Logfire")
}

func TestClient_DeriveAPIBaseURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, logfireEUAPIBaseURL, deriveAPIBaseURL("https://logfire-eu.pydantic.dev"))
	assert.Equal(t, logfireEUAPIBaseURL, deriveAPIBaseURL("https://api-eu.pydantic.dev"))
	assert.Equal(t, logfireUSAPIBaseURL, deriveAPIBaseURL("https://logfire-us.pydantic.dev"))
}
