package dash0

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Dash0__ListResources(t *testing.T) {
	d := &Dash0{}

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{}
		httpContext := &contexts.HTTPContext{}

		resources, err := d.ListResources("unknown", core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Integration: integrationContext,
		})

		require.NoError(t, err)
		require.Empty(t, resources)
		require.Empty(t, httpContext.Requests)
	})

	t.Run("returns check rules", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(
						strings.NewReader(`[{"id":"rule-1","name":"CPU high"},{"id":"rule-2","name":"Latency"}]`),
					),
				},
			},
		}

		resources, err := d.ListResources("check-rule", core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Integration: integrationContext,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "check-rule", resources[0].Type)
		assert.Equal(t, "CPU high", resources[0].Name)
		assert.Equal(t, "rule-1", resources[0].ID)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/alerting/check-rules")
	})

	t.Run("returns synthetic checks", func(t *testing.T) {
		integrationContext := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"metadata":{"name":"Login health","labels":{"dash0.com/id":"check-1"}}}]`,
					)),
				},
			},
		}

		resources, err := d.ListResources("synthetic-check", core.ListResourcesContext{
			Logger:      logrus.NewEntry(logrus.New()),
			HTTP:        httpContext,
			Integration: integrationContext,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "synthetic-check", resources[0].Type)
		assert.Equal(t, "Login health", resources[0].Name)
		assert.Equal(t, "check-1", resources[0].ID)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/synthetic-checks")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "dataset=default")
	})
}
