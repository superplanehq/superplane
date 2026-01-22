package smtp

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SendEmail__Setup(t *testing.T) {
	component := &SendEmail{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration:   "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing to -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"subject": "Test Subject",
				"body":    "Test body",
			},
		})

		require.ErrorContains(t, err, "to is required")
	})

	t.Run("missing subject -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"to":   "test@example.com",
				"body": "Test body",
			},
		})

		require.ErrorContains(t, err, "subject is required")
	})

	t.Run("missing body -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"to":      "test@example.com",
				"subject": "Test Subject",
			},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("invalid email in to -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        &contexts.MetadataContext{},
			Configuration: map[string]any{
				"to":      "not-an-email",
				"subject": "Test Subject",
				"body":    "Test body",
			},
		})

		require.ErrorContains(t, err, "invalid 'to' email addresses")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        metadata,
			Configuration: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test Subject",
				"body":    "Hello, this is a test.",
			},
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(SendEmailMetadata)
		require.True(t, ok)
		assert.Equal(t, []string{"recipient@example.com"}, stored.To)
		assert.Equal(t, "Test Subject", stored.Subject)
	})

	t.Run("valid configuration with multiple recipients -> stores all", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: &contexts.AppInstallationContext{},
			Metadata:        metadata,
			Configuration: map[string]any{
				"to":      "a@example.com, b@example.com, c@example.com",
				"subject": "Test Subject",
				"body":    "<p>Hello</p>",
				"isHTML":  true,
			},
		})

		require.NoError(t, err)
		stored := metadata.Metadata.(SendEmailMetadata)
		assert.Equal(t, []string{"a@example.com", "b@example.com", "c@example.com"}, stored.To)
	})
}

func Test__SendEmail__Execute(t *testing.T) {
	component := &SendEmail{}

	t.Run("missing to -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			AppInstallation: &contexts.AppInstallationContext{
				Configuration: map[string]any{
					"host":      "smtp.example.com",
					"port":      "587",
					"fromEmail": "sender@example.com",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration: map[string]any{
				"subject": "Test",
				"body":    "Body",
			},
		})

		require.ErrorContains(t, err, "to is required")
	})

	t.Run("valid configuration -> sends email and emits result", func(t *testing.T) {
		// Mock SMTP client
		sentData := &strings.Builder{}
		mockClient := &fakeSMTPClient{
			dataWriter: sentData,
		}

		originalDial := smtpDial
		smtpDial = func(addr string) (smtpClient, error) {
			assert.Equal(t, "smtp.example.com:587", addr)
			return mockClient, nil
		}
		defer func() { smtpDial = originalDial }()

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"host":      "smtp.example.com",
				"port":      "587",
				"fromEmail": "sender@example.com",
				"fromName":  "Test Sender",
				"useTLS":    "false",
			},
		}

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  execState,
			Configuration: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test Subject",
				"body":    "Hello, this is a test message.",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "smtp.email.sent", execState.Type)
		require.Len(t, execState.Payloads, 1)

		// Verify SMTP commands were called correctly
		assert.Equal(t, "sender@example.com", mockClient.mailFrom)
		assert.Equal(t, []string{"recipient@example.com"}, mockClient.rcptTo)
		assert.True(t, mockClient.quitCalled)

		// Verify email content
		emailContent := sentData.String()
		assert.Contains(t, emailContent, "From: Test Sender <sender@example.com>")
		assert.Contains(t, emailContent, "To: recipient@example.com")
		assert.Contains(t, emailContent, "Subject: Test Subject")
		assert.Contains(t, emailContent, "Hello, this is a test message.")
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

		err := component.Execute(core.ExecutionContext{
			AppInstallation: appCtx,
			ExecutionState:  &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test",
				"body":    "Body",
			},
		})

		require.ErrorContains(t, err, "connection refused")
	})
}

// fakeSMTPClient is a mock implementation for testing
type fakeSMTPClient struct {
	dataWriter *strings.Builder
	mailFrom   string
	rcptTo     []string
	quitCalled bool
	authCalled bool
}

func (c *fakeSMTPClient) Hello(localName string) error {
	return nil
}

func (c *fakeSMTPClient) Extension(ext string) (bool, string) {
	if ext == "STARTTLS" {
		return true, ""
	}
	return false, ""
}

func (c *fakeSMTPClient) StartTLS(config *tls.Config) error {
	return nil
}

func (c *fakeSMTPClient) Auth(auth smtp.Auth) error {
	c.authCalled = true
	return nil
}

func (c *fakeSMTPClient) Mail(from string) error {
	c.mailFrom = from
	return nil
}

func (c *fakeSMTPClient) Rcpt(to string) error {
	c.rcptTo = append(c.rcptTo, to)
	return nil
}

func (c *fakeSMTPClient) Data() (io.WriteCloser, error) {
	return &nopWriteCloser{c.dataWriter}, nil
}

func (c *fakeSMTPClient) Quit() error {
	c.quitCalled = true
	return nil
}

func (c *fakeSMTPClient) Close() error {
	return nil
}

type nopWriteCloser struct {
	w io.Writer
}

func (n *nopWriteCloser) Write(p []byte) (int, error) {
	return n.w.Write(p)
}

func (n *nopWriteCloser) Close() error {
	return nil
}
