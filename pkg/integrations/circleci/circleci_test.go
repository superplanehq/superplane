package circleci

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

func Test__CircleCI__Sync(t *testing.T) {
	c := &CircleCI{}

	t.Run("success verifying API token -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"user-123","login":"testuser","name":"Test User"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Projects: []string{}},
			Configuration: map[string]any{
				"apiToken": "token-123",
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
		assert.Equal(t, "https://circleci.com/api/v2/me", httpContext.Requests[0].URL.String())
		assert.Equal(t, "token-123", httpContext.Requests[0].Header.Get("Circle-Token"))
	})

	t.Run("failure verifying API token -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Projects: []string{}},
			Configuration: map[string]any{
				"apiToken": "invalid-token",
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
