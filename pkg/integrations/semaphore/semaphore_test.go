package semaphore

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

func Test__Semaphore__Sync(t *testing.T) {
	s := &Semaphore{}

	t.Run("success listing projects -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("[]")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Projects: []string{}},
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/projects", httpContext.Requests[0].URL.String())
	})

	t.Run("failure listing projects -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("server error")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Projects: []string{}},
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/projects", httpContext.Requests[0].URL.String())
	})
}
