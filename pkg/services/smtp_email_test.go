package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSettingsProvider struct {
	settings *SMTPSettings
	err      error
}

func (p *fakeSettingsProvider) GetSMTPSettings(ctx context.Context) (*SMTPSettings, error) {
	return p.settings, p.err
}

type bufferCloser struct {
	buf *bytes.Buffer
}

func (b *bufferCloser) Write(p []byte) (int, error) {
	return b.buf.Write(p)
}

func (b *bufferCloser) Close() error {
	return nil
}

type fakeSMTPClient struct {
	rcpt        []string
	mailFrom    string
	message     bytes.Buffer
	extensions  map[string]bool
	startedTLS  bool
	authCalled  bool
	quitCalled  bool
	closeCalled bool
}

func (c *fakeSMTPClient) Hello(localName string) error {
	return nil
}

func (c *fakeSMTPClient) Extension(ext string) (bool, string) {
	return c.extensions[ext], ""
}

func (c *fakeSMTPClient) StartTLS(_ *tls.Config) error {
	c.startedTLS = true
	return nil
}

func (c *fakeSMTPClient) Auth(_ smtp.Auth) error {
	c.authCalled = true
	return nil
}

func (c *fakeSMTPClient) Mail(from string) error {
	c.mailFrom = from
	return nil
}

func (c *fakeSMTPClient) Rcpt(to string) error {
	c.rcpt = append(c.rcpt, to)
	return nil
}

func (c *fakeSMTPClient) Data() (io.WriteCloser, error) {
	return &bufferCloser{buf: &c.message}, nil
}

func (c *fakeSMTPClient) Quit() error {
	c.quitCalled = true
	return nil
}

func (c *fakeSMTPClient) Close() error {
	c.closeCalled = true
	return nil
}

func TestBuildMultipartEmail(t *testing.T) {
	msg, err := buildMultipartEmail(
		"Sender <sender@example.com>",
		[]string{"to@example.com"},
		[]string{"bcc@example.com"},
		"Subject line",
		"plain body",
		"<p>html body</p>",
	)
	require.NoError(t, err)

	assert.Contains(t, msg, "Subject: Subject line")
	assert.Contains(t, msg, "To: to@example.com")
	assert.NotContains(t, msg, "Bcc:")
	assert.Contains(t, msg, "Content-Type: multipart/alternative; boundary=")
	assert.Contains(t, msg, "Content-Type: text/plain")
	assert.Contains(t, msg, "plain body")
	assert.Contains(t, msg, "Content-Type: text/html")
	assert.Contains(t, msg, "<p>html body</p>")
}

func TestSMTPEmailService_SendInvitationEmail(t *testing.T) {
	tmpDir := t.TempDir()
	templateDir := filepath.Join(tmpDir, "email")
	require.NoError(t, os.MkdirAll(templateDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "invitation.txt"), []byte("Invite {{.ToEmail}}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "invitation.html"), []byte("<p>Invite {{.ToEmail}}</p>"), 0o644))

	settings := &SMTPSettings{
		Host:      "smtp.example.com",
		Port:      587,
		Username:  "user",
		Password:  "pass",
		FromName:  "Superplane",
		FromEmail: "noreply@example.com",
		UseTLS:    true,
	}

	provider := &fakeSettingsProvider{settings: settings}
	service := NewSMTPEmailService(provider, tmpDir)

	fakeClient := &fakeSMTPClient{extensions: map[string]bool{"STARTTLS": true}}
	originalDial := smtpDial
	smtpDial = func(addr string) (smtpClient, error) {
		return fakeClient, nil
	}
	t.Cleanup(func() {
		smtpDial = originalDial
	})

	err := service.SendInvitationEmail("user@example.com", "Org", "https://example.com", "inviter@example.com")
	require.NoError(t, err)

	assert.Equal(t, "noreply@example.com", fakeClient.mailFrom)
	assert.Equal(t, []string{"user@example.com"}, fakeClient.rcpt)
	assert.True(t, fakeClient.startedTLS)
	assert.True(t, fakeClient.authCalled)
	assert.True(t, fakeClient.quitCalled)
	assert.True(t, fakeClient.closeCalled)

	message := fakeClient.message.String()
	assert.Contains(t, message, "Subject: You have been invited to join an organization on Superplane")
	assert.Contains(t, message, "To: user@example.com")
	assert.True(t, strings.Contains(message, "Invite user@example.com"))
}
