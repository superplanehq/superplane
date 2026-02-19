package terraformcloud

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

func Test__TerraformCloud__Sync(t *testing.T) {
	c := &TerraformCloud{}

	t.Run("success verifying API token -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"id":"user-abc","attributes":{"username":"testuser","email":"test@example.com"}}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{},
			Configuration: map[string]any{
				"apiToken": "test-token",
				"hostname": "app.terraform.io",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://app.terraform.io/api/v2/account/details", httpContext.Requests[0].URL.String())
		assert.Equal(t, "Bearer test-token", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("failure verifying API token -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"status":"401","title":"Unauthorized"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{},
			Configuration: map[string]any{
				"apiToken": "invalid-token",
				"hostname": "app.terraform.io",
			},
		}

		err := c.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
	})
}
