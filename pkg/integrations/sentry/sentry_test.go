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
}
