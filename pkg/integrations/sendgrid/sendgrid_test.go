package sendgrid

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

func Test__SendGrid__Sync(t *testing.T) {
	app := &SendGrid{}

	t.Run("missing apiKey -> error", func(t *testing.T) {
		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
		})

		require.ErrorContains(t, err, "apiKey is required")
	})

	t.Run("missing fromEmail -> error", func(t *testing.T) {
		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{
				"apiKey": "sg-test",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
		})

		require.ErrorContains(t, err, "fromEmail is required")
	})

	t.Run("valid configuration -> sets ready state", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"email":"hello@example.com"}`)),
					Header:     http.Header{},
					Request:    &http.Request{},
					Status:     http.StatusText(http.StatusOK),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "sg-valid",
			},
		}

		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{
				"apiKey":    "sg-valid",
				"fromEmail": "sender@example.com",
			},
			Integration: integrationCtx,
			HTTP:        httpCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
	})

	t.Run("invalid apiKey -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"message":"invalid"}]}`)),
					Header:     http.Header{},
					Request:    &http.Request{},
					Status:     http.StatusText(http.StatusUnauthorized),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "sg-invalid",
			},
		}

		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{
				"apiKey":    "sg-invalid",
				"fromEmail": "sender@example.com",
			},
			Integration: integrationCtx,
			HTTP:        httpCtx,
		})

		require.ErrorContains(t, err, "failed to verify SendGrid credentials")
	})
}
