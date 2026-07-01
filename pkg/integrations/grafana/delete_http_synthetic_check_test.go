package grafana

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteHTTPSyntheticCheck__Execute(t *testing.T) {
	component := &DeleteHTTPSyntheticCheck{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			grafanaSyntheticDataSourceResponse(),
			grafanaSyntheticCheckGetResponse(`{
				"id": 101,
				"job": "API health",
				"target": "https://api.example.com/health",
				"frequency": 60000,
				"timeout": 3000,
				"enabled": true,
				"basicMetricsOnly": true,
				"settings": {"http": {"method": "GET"}},
				"probes": [1]
			}`),
			grafanaSyntheticHTTPResponse(`{"msg":"Check deleted","checkId":101}`),
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"syntheticCheck": "101",
		},
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "grafana-token",
			},
		},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "grafana.syntheticCheck.deleted", execCtx.Type)
	require.Len(t, execCtx.Payloads, 1)
	payload := execCtx.Payloads[0].(map[string]any)
	data := payload["data"].(DeleteHTTPSyntheticCheckOutput)
	assert.Equal(t, "101", data.SyntheticCheck)
	assert.True(t, data.Deleted)
}
