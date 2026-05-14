package cloudflare

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListCFDTunnels__UnmarshalsMetadataObject(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"success": true,
					"result": [
						{
							"id": "tun-1",
							"name": "edge",
							"metadata": {"user_visible": true, "team": "net"}
						}
					]
				}`)),
			},
		},
	}

	client := &Client{
		Token:   "tok",
		http:    httpContext,
		BaseURL: baseURL,
	}

	tunnels, err := client.ListCFDTunnels("acc123")
	require.NoError(t, err)
	require.Len(t, tunnels, 1)
	assert.Equal(t, "tun-1", tunnels[0].ID)
	assert.Equal(t, "edge", tunnels[0].Name)
	assert.JSONEq(t, `{"user_visible": true, "team": "net"}`, string(tunnels[0].Metadata))
}
