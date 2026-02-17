package flyio

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListApps__Execute__Success(t *testing.T) {
	c := &ListApps{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 200,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"total_apps": 2,
					"apps": [
						{"id": "app1", "name": "my-app-1", "status": "deployed", "machine_count": 2},
						{"id": "app2", "name": "my-app-2", "status": "suspended", "machine_count": 0}
					]
				}`)),
			},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
			"orgSlug":  "personal",
		},
	}

	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Configuration: map[string]any{
			"orgSlug": "personal",
		},
	}

	err := c.Execute(ctx)
	require.NoError(t, err)

	// Verify request
	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Contains(t, req.URL.String(), "org_slug=personal")

	// Verify output
	require.True(t, executionState.Finished)
	require.True(t, executionState.Passed)
	require.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
	require.Equal(t, "flyio.appList", executionState.Type)

	require.Len(t, executionState.Payloads, 1)
	payload := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "personal", payload["orgSlug"])
	assert.Equal(t, 2, payload["count"])

	apps := payload["apps"].([]map[string]any)
	assert.Len(t, apps, 2)
	assert.Equal(t, "my-app-1", apps[0]["name"])
	assert.Equal(t, "deployed", apps[0]["status"])
	assert.Equal(t, 2, apps[0]["machineCount"])
}

func Test__ListApps__Execute__DefaultOrgFromIntegration(t *testing.T) {
	c := &ListApps{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"total_apps": 0, "apps": []}`)),
			},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
			"orgSlug":  "default-org",
		},
	}

	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Configuration:  map[string]any{
			// No orgSlug provided in component config
		},
	}

	err := c.Execute(ctx)
	require.NoError(t, err)

	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Contains(t, req.URL.String(), "org_slug=default-org")
}

func Test__ListApps__Execute__Error(t *testing.T) {
	c := &ListApps{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 500,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": "internal server error"}`)),
			},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
			"orgSlug":  "personal",
		},
	}

	executionState := &contexts.ExecutionStateContext{}

	ctx := core.ExecutionContext{
		HTTP:           mockHTTP,
		Integration:    mockIntegration,
		ExecutionState: executionState,
		Configuration: map[string]any{
			"orgSlug": "personal",
		},
	}

	err := c.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list apps")
}
