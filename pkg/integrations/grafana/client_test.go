package grafana

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

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

