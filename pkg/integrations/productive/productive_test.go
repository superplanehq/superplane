package productive

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

func Test__Productive__Sync(t *testing.T) {
	p := &Productive{}

	t.Run("missing apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "",
				"organizationId": "org-1",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("missing organizationId -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "token-1",
				"organizationId": "",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "organizationId is required")
	})

	t.Run("valid credentials -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(webhookProbeValidationBody)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "token-1",
				"organizationId": "org-1",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "organization_memberships")
		assert.Equal(t, "token-1", httpContext.Requests[0].Header.Get(AuthTokenHeader))
		assert.Equal(t, "org-1", httpContext.Requests[0].Header.Get(OrganizationIDHeader))
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
		assert.Contains(t, httpContext.Requests[1].URL.Path, "/webhooks")
	})

	t.Run("webhook probe validation error -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(webhookProbeValidationBody)),
				},
			},
		}

		appCtx := authorizedIntegration()
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
		body, readErr := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, readErr)
		assert.JSONEq(t, `{"data":{"type":"webhooks","attributes":{}}}`, string(body))
	})

	t.Run("webhook probe 403 without a code -> read and write token required", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"status":"403","title":"Forbidden"}]}`)),
				},
			},
		}

		appCtx := authorizedIntegration()
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingWritePermission)
		assert.Contains(t, err.Error(), "read and write access")
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("webhook probe webhooks_limit_exceeded -> plan message", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusForbidden,
					Body: io.NopCloser(strings.NewReader(
						`{"errors":[{"status":"403","code":"webhooks_limit_exceeded","title":"Webhooks are not available on your plan"}]}`,
					)),
				},
			},
		}

		appCtx := authorizedIntegration()
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWebhooksLimitExceeded)
		assert.Contains(t, err.Error(), "does not offer webhooks on this plan")
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("webhook probe creates a webhook -> deletes it and is ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"id":"probe-1","type":"webhooks","attributes":{}}}`)),
				},
				jsonResponse(`{}`),
			},
		}

		appCtx := authorizedIntegration()
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[2].Method)
		assert.Contains(t, httpContext.Requests[2].URL.String(), "/webhooks/probe-1")
	})

	t.Run("webhook probe delete failure retains the webhook id", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"id":"probe-1","type":"webhooks","attributes":{}}}`)),
				},
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"title":"Server Error"}]}`)),
				},
			},
		}

		appCtx := authorizedIntegration()
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "could not delete webhook probe-1")
		assert.NotEqual(t, "ready", appCtx.State)
		assert.Equal(t, "probe-1", probeWebhookID(appCtx))
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[2].Method)
	})

	t.Run("retained probe webhook is deleted before another probe", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				jsonResponse(`{}`),
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(webhookProbeValidationBody)),
				},
			},
		}

		appCtx := authorizedIntegration()
		appCtx.Metadata = map[string]any{probeWebhookMetadataKey: "probe-1", "other": "kept"}
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		assert.Empty(t, probeWebhookID(appCtx))
		assert.Equal(t, "kept", appCtx.Metadata.(map[string]any)["other"])
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/webhooks/probe-1")
		assert.Equal(t, http.MethodPost, httpContext.Requests[2].Method)
	})

	t.Run("retained probe webhook delete failure does not create another", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"title":"Server Error"}]}`)),
				},
			},
		}

		appCtx := authorizedIntegration()
		appCtx.Metadata = map[string]any{probeWebhookMetadataKey: "probe-1"}
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "previous permission check")
		assert.NotEqual(t, "ready", appCtx.State)
		assert.Equal(t, "probe-1", probeWebhookID(appCtx))
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
		assert.NotEqual(t, http.MethodPost, httpContext.Requests[1].Method)
	})

	t.Run("webhook probe 422 for another attribute is not ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body: io.NopCloser(strings.NewReader(
						`{"errors":[{"status":"422","title":"Invalid Attribute","detail":"is invalid","source":{"pointer":"data/attributes/custom_headers"}}]}`,
					)),
				},
			},
		}

		appCtx := authorizedIntegration()
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "error checking webhook permission")
		assert.NotErrorIs(t, err, ErrMissingWritePermission)
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("webhook probe 400 is not ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusBadRequest,
					Body: io.NopCloser(strings.NewReader(
						`{"errors":[{"status":"400","title":"Unsupported Filter","detail":"filter is not supported on this endpoint"}]}`,
					)),
				},
			},
		}

		appCtx := authorizedIntegration()
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "error checking webhook permission")
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("webhook probe 401 is an authentication failure", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[]}`),
				{
					StatusCode: http.StatusUnauthorized,
					Body: io.NopCloser(strings.NewReader(
						`{"errors":[{"status":"401","title":"Unauthenticated","detail":"You are not authenticated"}]}`,
					)),
				},
			},
		}

		appCtx := authorizedIntegration()
		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrMissingWritePermission)
		assert.NotContains(t, err.Error(), "read and write access")
		assert.NotEqual(t, "ready", appCtx.State)
	})

	t.Run("invalid credentials -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"title":"Not authorized"}]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken":       "bad-token",
				"organizationId": "org-1",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid credentials")
		assert.NotEqual(t, "ready", appCtx.State)
	})
}

func Test__Productive__ListResources(t *testing.T) {
	p := &Productive{}

	t.Run("unknown type returns empty", func(t *testing.T) {
		resources, err := p.ListResources("unknown", core.ListResourcesContext{
			Integration: authorizedIntegration(),
			HTTP:        &contexts.HTTPContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("projects lists connected projects", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":[{"id":"1","type":"projects","attributes":{"name":"Payments"}}]}`),
			},
		}

		resources, err := p.ListResources(ResourceTypeProject, core.ListResourcesContext{
			Integration: authorizedIntegration(),
			HTTP:        httpContext,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "Payments", resources[0].Name)
		assert.Equal(t, ResourceTypeProject, resources[0].Type)
	})
}

func Test__Productive__Instructions(t *testing.T) {
	p := &Productive{}

	instructions := p.Instructions()

	assert.Contains(t, instructions, "Settings > API Integrations")
	assert.Contains(t, instructions, "read and write access")
	assert.Contains(t, instructions, "create webhooks")
	assert.NotContains(t, instructions, "profile")
}

func Test__Productive__Configuration(t *testing.T) {
	p := &Productive{}

	fields := p.Configuration()

	require.Len(t, fields, 3)
	assert.Equal(t, "apiToken", fields[0].Name)
	assert.Equal(t, "Personal access token from Productive Settings > API Integrations. SuperPlane needs read and write access to create webhooks.", fields[0].Description)
	assert.Equal(t, "The numeric organization id in your Productive URL", fields[1].Description)
}

// authorizedIntegration returns an integration context with valid Productive.io
// credentials configured, for tests that need a working client.
func authorizedIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken":       "token-1",
			"organizationId": "org-1",
		},
	}
}

const webhookProbeValidationBody = `{"errors":[
	{"status":"422","title":"Invalid Attribute","detail":"can't be blank","source":{"pointer":"data/attributes/event_id"}},
	{"status":"422","title":"Invalid Attribute","detail":"can't be blank","source":{"pointer":"data/attributes/type_id"}},
	{"status":"422","title":"Invalid Attribute","detail":"can't be blank","source":{"pointer":"data/attributes/target_url"}}
]}`

func probeWebhookID(integration *contexts.IntegrationContext) string {
	metadata, _ := integration.Metadata.(map[string]any)
	if metadata == nil {
		return ""
	}
	id, _ := metadata[probeWebhookMetadataKey].(string)
	return id
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
