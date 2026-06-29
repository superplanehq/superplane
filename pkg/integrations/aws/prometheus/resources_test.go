package prometheus

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

func Test__ListWorkspaces(t *testing.T) {
	t.Run("missing region -> error", func(t *testing.T) {
		_, err := ListWorkspaces(core.ListResourcesContext{
			Integration: validIntegrationContext(),
			Parameters:  map[string]string{},
		}, workspaceResourceType)

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("valid request -> returns workspace resources", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"workspaces": [
							{"alias": "metrics", "workspaceId": "ws-abc123", "status": {"statusCode": "ACTIVE"}},
							{"workspaceId": "ws-def456", "status": {"statusCode": "CREATING"}}
						]
					}`)),
				},
			},
		}

		resources, err := ListWorkspaces(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: validIntegrationContext(),
			Parameters: map[string]string{
				"region": "us-east-1",
			},
		}, workspaceResourceType)

		require.NoError(t, err)
		assert.Equal(t, []core.IntegrationResource{
			{Type: workspaceResourceType, Name: "metrics", ID: "ws-abc123"},
			{Type: workspaceResourceType, Name: "ws-def456", ID: "ws-def456"},
		}, resources)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://aps.us-east-1.amazonaws.com/workspaces?maxResults=1000", httpContext.Requests[0].URL.String())
	})
}
