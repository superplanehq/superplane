package grafana

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__readBaseURL__RejectsRelativeURL(t *testing.T) {
	_, err := readBaseURL(&contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL": "grafana.local",
		},
	})
	require.ErrorContains(t, err, "must include scheme and host")
}

func Test__readBaseURL__AcceptsAbsoluteHTTPURL(t *testing.T) {
	baseURL, err := readBaseURL(&contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseURL": "https://grafana.example.com/",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://grafana.example.com", baseURL)
}

func Test__Client__ExecRequest__AllowsExactMaxSize(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), maxResponseSize))),
			},
		},
	}

	client := &Client{
		BaseURL: "https://grafana.example.com",
		http:    httpContext,
	}

	body, status, err := client.execRequest(http.MethodGet, "/api/health", nil, "")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, body, maxResponseSize)
}

func Test__Client__ExecRequest__RejectsOverMaxSize(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), maxResponseSize+1))),
			},
		},
	}

	client := &Client{
		BaseURL: "https://grafana.example.com",
		http:    httpContext,
	}

	_, status, err := client.execRequest(http.MethodGet, "/api/health", nil, "")
	require.ErrorContains(t, err, "response too large")
	require.Equal(t, http.StatusOK, status)
}

func Test__Grafana__Sync__RejectsRelativeBaseURL(t *testing.T) {
	err := (&Grafana{}).Sync(core.SyncContext{
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL": "grafana.local",
			},
			Metadata: map[string]any{},
		},
	})
	require.ErrorContains(t, err, "must include scheme and host")
}
