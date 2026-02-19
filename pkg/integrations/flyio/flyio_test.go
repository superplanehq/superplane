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

// Test Sync() - Successful sync with valid token
func Test__FlyIO__Sync__Success(t *testing.T) {
	f := &FlyIO{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 200,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"total_apps": 2,
					"apps": [
						{"id": "app1", "name": "my-app-1", "status": "deployed"},
						{"id": "app2", "name": "my-app-2", "status": "deployed"}
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

	ctx := core.SyncContext{
		HTTP:        mockHTTP,
		Integration: mockIntegration,
		Configuration: map[string]any{
			"orgSlug": "personal",
		},
	}

	err := f.Sync(ctx)
	require.NoError(t, err)

	// Verify request
	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
	assert.Contains(t, req.URL.String(), "org_slug=personal")

	// Verify metadata was set
	metadata, ok := mockIntegration.Metadata.(Metadata)
	require.True(t, ok)
	assert.Len(t, metadata.Apps, 2)
	assert.Equal(t, "my-app-1", metadata.Apps[0].Name)
	assert.Equal(t, "my-app-2", metadata.Apps[1].Name)
}

// Test Sync() - Invalid API token
func Test__FlyIO__Sync__InvalidToken(t *testing.T) {
	f := &FlyIO{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 401,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error": "unauthorized"}`)),
			},
		},
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "invalid-token",
		},
	}

	ctx := core.SyncContext{
		HTTP:        mockHTTP,
		Integration: mockIntegration,
		Configuration: map[string]any{
			"orgSlug": "personal",
		},
	}

	err := f.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing apps")
}

// Test Sync() - Network error (simulated by no matching response)
func Test__FlyIO__Sync__NetworkError(t *testing.T) {
	f := &FlyIO{}

	mockHTTP := &contexts.HTTPContext{
		Responses: []*http.Response{}, // No responses will cause an error
	}

	mockIntegration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "test-token",
		},
	}

	ctx := core.SyncContext{
		HTTP:          mockHTTP,
		Integration:   mockIntegration,
		Configuration: map[string]any{},
	}

	err := f.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing apps")
}

// Test Sync() - Default org slug when not provided
func Test__FlyIO__Sync__DefaultOrgSlug(t *testing.T) {
	f := &FlyIO{}

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
		},
	}

	ctx := core.SyncContext{
		HTTP:          mockHTTP,
		Integration:   mockIntegration,
		Configuration: map[string]any{
			// No orgSlug provided
		},
	}

	err := f.Sync(ctx)
	require.NoError(t, err)

	require.Len(t, mockHTTP.Requests, 1)
	req := mockHTTP.Requests[0]
	assert.Contains(t, req.URL.String(), "org_slug=personal")
}
