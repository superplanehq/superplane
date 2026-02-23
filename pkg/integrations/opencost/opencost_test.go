package opencost

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

func Test__OpenCost__Sync(t *testing.T) {
	integration := &OpenCost{}

	t.Run("successful sync marks integration as ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"code":200,"status":"success","data":[{"cluster-one":{"name":"cluster-one","totalCost":5.0}}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiURL": "http://opencost:9003",
			},
		}

		err := integration.Sync(core.SyncContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
	})

	t.Run("failed API call returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"internal error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiURL": "http://opencost:9003",
			},
		}

		err := integration.Sync(core.SyncContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.ErrorContains(t, err, "failed to verify OpenCost connection")
	})

	t.Run("missing API URL returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		err := integration.Sync(core.SyncContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.Error(t, err)
	})
}
