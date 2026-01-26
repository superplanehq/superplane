package smtp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SMTP__Sync(t *testing.T) {
	app := &SMTP{}

	t.Run("missing host -> error", func(t *testing.T) {
		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{
				"port":      "587",
				"fromEmail": "sender@example.com",
			},
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{},
			},
		})

		require.ErrorContains(t, err, "host is required")
	})

	t.Run("invalid port -> error", func(t *testing.T) {
		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{
				"host":      "smtp.example.com",
				"port":      "0",
				"fromEmail": "sender@example.com",
			},
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{},
			},
		})

		require.ErrorContains(t, err, "port must be a number between 1 and 65535")
	})

	t.Run("missing fromEmail -> error", func(t *testing.T) {
		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{
				"host": "smtp.example.com",
				"port": "587",
			},
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{},
			},
		})

		require.ErrorContains(t, err, "fromEmail is required")
	})

	t.Run("valid configuration with successful connection -> sets ready state", func(t *testing.T) {
		// Mock SMTP client that succeeds
		originalDial := smtpDial
		smtpDial = func(addr string) (smtpClient, error) {
			assert.Equal(t, "smtp.example.com:587", addr)
			return &fakeSMTPClient{}, nil
		}
		defer func() { smtpDial = originalDial }()

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"host":      "smtp.example.com",
				"port":      "587",
				"fromEmail": "sender@example.com",
				"useTLS":    "false",
			},
		}

		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{
				"host":      "smtp.example.com",
				"port":      "587",
				"fromEmail": "sender@example.com",
				"useTLS":    false,
			},
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
	})

	t.Run("SMTP connection failure -> returns error", func(t *testing.T) {
		originalDial := smtpDial
		smtpDial = func(addr string) (smtpClient, error) {
			return nil, fmt.Errorf("connection refused")
		}
		defer func() { smtpDial = originalDial }()

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"host":      "smtp.example.com",
				"port":      "587",
				"fromEmail": "sender@example.com",
				"useTLS":    "false",
			},
		}

		err := app.Sync(core.SyncContext{
			Configuration: map[string]any{
				"host":      "smtp.example.com",
				"port":      "587",
				"fromEmail": "sender@example.com",
				"useTLS":    false,
			},
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "SMTP connection test failed")
	})
}
