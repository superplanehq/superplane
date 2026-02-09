package sentry

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__looksLikeClientSecret(t *testing.T) {
	assert.True(t, looksLikeClientSecret("f3233f8fad765a684521a617144e8f15f5b34c1bea18c0354034ce1bfc4414cb"))
	assert.True(t, looksLikeClientSecret(strings.Repeat("a", 64)))
	assert.False(t, looksLikeClientSecret("sntryu_not_a_real_token"))
	assert.False(t, looksLikeClientSecret("sntrys_xxx"))
	assert.False(t, looksLikeClientSecret("short"))
	assert.False(t, looksLikeClientSecret(strings.Repeat("a", 63)))
	assert.False(t, looksLikeClientSecret(strings.Repeat("z", 64))) // z is not hex
}

func Test__Sentry__Name(t *testing.T) {
	s := &Sentry{}
	assert.Equal(t, "sentry", s.Name())
}

func Test__Sentry__Label(t *testing.T) {
	s := &Sentry{}
	assert.Equal(t, "Sentry", s.Label())
}

func Test__Sentry__Components(t *testing.T) {
	s := &Sentry{}
	components := s.Components()
	assert.Len(t, components, 1)
	assert.Equal(t, "sentry.updateIssue", components[0].Name())
}

func Test__Sentry__Triggers(t *testing.T) {
	s := &Sentry{}
	triggers := s.Triggers()
	assert.Len(t, triggers, 1)
	assert.Equal(t, "sentry.onIssueEvent", triggers[0].Name())
}

func Test__Sentry__Sync(t *testing.T) {
	s := &Sentry{}

	t.Run("missing authToken -> error", func(t *testing.T) {
		ctx := core.SyncContext{
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{},
		}
		err := s.Sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authToken")
	})

	t.Run("invalid token (API error) -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("{}"))},
			},
		}
		integration := &contexts.IntegrationContext{
			Configuration: map[string]any{"authToken": "bad", "baseURL": "https://sentry.io"},
		}
		ctx := core.SyncContext{
			Configuration: map[string]any{"authToken": "bad", "baseURL": "https://sentry.io"},
			HTTP:          httpCtx,
			Integration:   integration,
		}
		err := s.Sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid Sentry token")
	})

	t.Run("valid token -> Ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("[]"))},
			},
		}
		integration := &contexts.IntegrationContext{
			Configuration: map[string]any{"authToken": "good-token", "baseURL": "https://sentry.io"},
		}
		ctx := core.SyncContext{
			Configuration: map[string]any{"authToken": "good-token", "baseURL": "https://sentry.io"},
			HTTP:          httpCtx,
			Integration:   integration,
		}
		err := s.Sync(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "ready", integration.State)
	})

	t.Run("401 with value that looks like Client Secret -> error hints Client Secret", func(t *testing.T) {
		clientSecretHex := "f3233f8fad765a684521a617144e8f15f5b34c1bea18c0354034ce1bfc4414cb"
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"detail":"..."}`))},
			},
		}
		integration := &contexts.IntegrationContext{
			Configuration: map[string]any{"authToken": clientSecretHex, "baseURL": "https://sentry.io"},
		}
		ctx := core.SyncContext{
			Configuration: map[string]any{"authToken": clientSecretHex, "baseURL": "https://sentry.io"},
			HTTP:          httpCtx,
			Integration:   integration,
		}
		err := s.Sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Client Secret")
		assert.Contains(t, err.Error(), "Webhook secret")
	})
}

func Test__Sentry__CompareWebhookConfig(t *testing.T) {
	s := &Sentry{}

	t.Run("identical events without secrets -> equal", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{"created", "resolved"}}
		b := WebhookConfiguration{Events: []string{"created", "resolved"}}
		equal, err := s.CompareWebhookConfig(a, b)
		assert.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("identical events with secrets -> equal (secrets ignored)", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{"created", "resolved"}, WebhookSecret: "secret1"}
		b := WebhookConfiguration{Events: []string{"created", "resolved"}, WebhookSecret: "secret2"}
		equal, err := s.CompareWebhookConfig(a, b)
		assert.NoError(t, err)
		assert.True(t, equal, "webhook secrets should be ignored during comparison")
	})

	t.Run("identical events, one with secret -> equal (secret ignored)", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{"created", "resolved"}, WebhookSecret: "secret"}
		b := WebhookConfiguration{Events: []string{"created", "resolved"}}
		equal, err := s.CompareWebhookConfig(a, b)
		assert.NoError(t, err)
		assert.True(t, equal, "webhook secret should be ignored during comparison")
	})

	t.Run("different events -> not equal", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{"created"}}
		b := WebhookConfiguration{Events: []string{"resolved"}}
		equal, err := s.CompareWebhookConfig(a, b)
		assert.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("different event counts -> not equal", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{"created", "resolved"}}
		b := WebhookConfiguration{Events: []string{"created"}}
		equal, err := s.CompareWebhookConfig(a, b)
		assert.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("same events different order -> equal", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{"created", "resolved", "assigned"}}
		b := WebhookConfiguration{Events: []string{"resolved", "assigned", "created"}}
		equal, err := s.CompareWebhookConfig(a, b)
		assert.NoError(t, err)
		assert.True(t, equal, "event order should not matter")
	})
}
