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

	t.Run("missing token -> no error", func(t *testing.T) {
		ctx := core.SyncContext{
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{},
		}
		err := s.Sync(ctx)
		assert.NoError(t, err)
	})

	t.Run("invalid token (API error) -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader("{}"))},
			},
		}
		integration := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				"sentryPublicAccessToken": {Name: "sentryPublicAccessToken", Value: []byte("bad")},
			},
		}
		ctx := core.SyncContext{
			HTTP:        httpCtx,
			Integration: integration,
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
			Secrets: map[string]core.IntegrationSecret{
				"sentryPublicAccessToken": {Name: "sentryPublicAccessToken", Value: []byte("good-token")},
			},
		}
		ctx := core.SyncContext{
			HTTP:        httpCtx,
			Integration: integration,
		}
		err := s.Sync(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "ready", integration.State)
	})
}

func Test__SentryWebhookHandler__CompareConfig(t *testing.T) {
	h := &SentryWebhookHandler{}

	t.Run("identical events -> equal", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{"created", "resolved"}}
		b := WebhookConfiguration{Events: []string{"created", "resolved"}}
		equal, err := h.CompareConfig(a, b)
		assert.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different events -> still equal (webhook can be reused)", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{"created"}}
		b := WebhookConfiguration{Events: []string{"resolved"}}
		equal, err := h.CompareConfig(a, b)
		assert.NoError(t, err)
		assert.True(t, equal)
	})
}

func Test__SentryWebhookHandler__Merge(t *testing.T) {
	h := &SentryWebhookHandler{}

	t.Run("requested is subset -> unchanged", func(t *testing.T) {
		current := WebhookConfiguration{Events: []string{"created", "resolved"}}
		requested := WebhookConfiguration{Events: []string{"created"}}

		mergedAny, changed, err := h.Merge(current, requested)
		assert.NoError(t, err)
		assert.False(t, changed)

		merged, ok := mergedAny.(WebhookConfiguration)
		assert.True(t, ok)
		assert.ElementsMatch(t, []string{"created", "resolved"}, merged.Events)
	})

	t.Run("requested adds new events -> changed + union", func(t *testing.T) {
		current := WebhookConfiguration{Events: []string{"created"}}
		requested := WebhookConfiguration{Events: []string{"resolved", "created"}}

		mergedAny, changed, err := h.Merge(current, requested)
		assert.NoError(t, err)
		assert.True(t, changed)

		merged, ok := mergedAny.(WebhookConfiguration)
		assert.True(t, ok)
		assert.ElementsMatch(t, []string{"created", "resolved"}, merged.Events)
	})
}
