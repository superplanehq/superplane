package coolify

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

func Test__Coolify_ListServices__Execute(t *testing.T) {
	component := &ListServices{}

	t.Run("emits services payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[
							{"uuid":"svc1","name":"postgres","status":"running","server_uuid":"srv1"},
							{"uuid":"svc2","name":"plausible","fqdn":"https://analytics.example.com","status":"running"}
						]`,
					)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, ListServicesPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, 2, data["count"])

		services, ok := data["services"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, services, 2)
		assert.Equal(t, "svc1", services[0]["uuid"])
		assert.Equal(t, "postgres", services[0]["name"])
		assert.Equal(t, "srv1", services[0]["serverUuid"])
		assert.Equal(t, "https://analytics.example.com", services[1]["fqdn"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://coolify.example.com/api/v1/services", httpCtx.Requests[0].URL.String())
	})
}
