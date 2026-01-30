package rootly

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

func Test__Rootly__Sync(t *testing.T) {
	r := &Rootly{}

	t.Run("no API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "",
			},
		}

		err := r.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "API key is required")
	})

	t.Run("successful sync -> ready state and metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"data": [
								{
									"id": "svc-123",
									"type": "services",
									"attributes": {
										"name": "Production API",
										"slug": "production-api",
										"description": "Main production API"
									}
								}
							]
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := r.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.rootly.com/v1/services", httpContext.Requests[0].URL.String())

		metadata := integrationCtx.Metadata.(Metadata)
		assert.Len(t, metadata.Services, 1)
		assert.Equal(t, "svc-123", metadata.Services[0].ID)
		assert.Equal(t, "Production API", metadata.Services[0].Name)
	})

	t.Run("failed service list -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": "unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "invalid-api-key",
			},
		}

		err := r.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
	})
}

func Test__Rootly__CompareWebhookConfig(t *testing.T) {
	r := &Rootly{}

	testCases := []struct {
		name        string
		configA     any
		configB     any
		expectEqual bool
		expectError bool
	}{
		{
			name: "identical events",
			configA: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different events",
			configA: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.resolved"},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "superset of events (A contains all of B)",
			configA: WebhookConfiguration{
				Events: []string{"incident.created", "incident.updated", "incident.resolved"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "subset of events (A does not contain all of B)",
			configA: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.created", "incident.resolved"},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"events": []string{"incident.created", "incident.updated"},
			},
			configB: map[string]any{
				"events": []string{"incident.created"},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
			configB:     "invalid",
			expectEqual: false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal, err := r.CompareWebhookConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err, "expected error, but got none")
			} else {
				require.NoError(t, err, "did not expect, but got an error")
			}

			assert.Equal(t, tc.expectEqual, equal, "expected config comparison result to match")
		})
	}
}

func Test__verifyWebhookSignature(t *testing.T) {
	t.Run("missing signature -> error", func(t *testing.T) {
		err := verifyWebhookSignature("", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "missing signature")
	})

	t.Run("invalid signature format -> error", func(t *testing.T) {
		err := verifyWebhookSignature("invalid", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "invalid signature format")
	})

	t.Run("missing timestamp -> error", func(t *testing.T) {
		err := verifyWebhookSignature("v1=abc123", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "invalid signature format")
	})

	t.Run("missing signature value -> error", func(t *testing.T) {
		err := verifyWebhookSignature("t=1234567890", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "invalid signature format")
	})

	t.Run("signature mismatch -> error", func(t *testing.T) {
		err := verifyWebhookSignature("t=1234567890,v1=invalid", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "signature mismatch")
	})

	t.Run("valid signature -> no error", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created"}}`)
		secret := []byte("test-secret")
		timestamp := "1234567890"

		// Compute the expected signature
		payload := append([]byte(timestamp), body...)
		expectedSig := computeHMACSHA256(secret, payload)

		signature := "t=" + timestamp + ",v1=" + expectedSig
		err := verifyWebhookSignature(signature, body, secret)
		require.NoError(t, err)
	})

	t.Run("valid signature with spaces -> no error", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created"}}`)
		secret := []byte("test-secret")
		timestamp := "1234567890"

		payload := append([]byte(timestamp), body...)
		expectedSig := computeHMACSHA256(secret, payload)

		// Format with spaces after commas (as Rootly might send)
		signature := "t=" + timestamp + ", v1=" + expectedSig
		err := verifyWebhookSignature(signature, body, secret)
		require.NoError(t, err)
	})
}
