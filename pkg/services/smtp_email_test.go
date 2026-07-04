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

func TestSMTPEmailService_SendMagicCodeEmail(t *testing.T) {
	tmpDir := t.TempDir()
	writeMagicCodeTemplates(t, tmpDir)

	settings := &SMTPSettings{
		Host:      "smtp.example.com",
		Port:      587,
		Username:  "user",
		Password:  "pass",
		FromName:  "SuperPlane",
		FromEmail: "noreply@example.com",
		UseTLS:    true,
	}

	provider := &fakeSettingsProvider{settings: settings}
	service := NewSMTPEmailService(provider, tmpDir)

	fakeClient := &fakeSMTPClient{extensions: map[string]bool{"STARTTLS": true}}
	originalDial := smtpDial
	smtpDial = func(addr string) (smtpClient, error) {
		assert.Equal(t, "smtp.example.com:587", addr)
		return fakeClient, nil
	}
	t.Cleanup(func() {
		smtpDial = originalDial
	})

	err := service.SendMagicCodeEmail("user@example.com", "123456", "https://example.com/login?token=a&next=b")
	require.NoError(t, err)

	assert.Equal(t, "noreply@example.com", fakeClient.mailFrom)
	assert.Equal(t, []string{"user@example.com"}, fakeClient.rcpt)
	assert.True(t, fakeClient.startedTLS)
	assert.True(t, fakeClient.authCalled)
	assert.True(t, fakeClient.quitCalled)
	assert.True(t, fakeClient.closeCalled)

	message := fakeClient.message.String()
	assert.Contains(t, message, "From: SuperPlane <noreply@example.com>")
	assert.Contains(t, message, "Subject: Your SuperPlane sign-in code")
	assert.Contains(t, message, "To: user@example.com")
	assert.True(t, strings.Contains(message, "Code 123456"))
	assert.True(t, strings.Contains(message, "https://example.com/login?token=a&next=b"))
	assert.True(t, strings.Contains(message, "<p>Code 123456</p>"))
}

func TestRenderEmailTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	templateDir := filepath.Join(tmpDir, "email")
	require.NoError(t, os.MkdirAll(templateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "magic_code.txt"), []byte("Open {{.MagicLink}}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "magic_code.html"), []byte("<a href=\"{{.MagicLink}}\">Open</a>"), 0o644))

	data := MagicCodeTemplateData{MagicLink: "https://example.com/login?token=a&next=b"}

	text, err := renderEmailTemplate(tmpDir, "magic_code.txt", data)
	require.NoError(t, err)
	assert.Equal(t, "Open https://example.com/login?token=a&next=b", text)

	html, err := renderEmailTemplate(tmpDir, "magic_code.html", data)
	require.NoError(t, err)
	assert.Contains(t, html, "token=a&amp;next=b")
}

func TestBuildEmailService(t *testing.T) {
	assert.Nil(t, BuildEmailService(nil, EmailServiceConfig{}))

	smtpService := BuildEmailService(nil, EmailServiceConfig{
		TemplateDir:       "templates",
		OwnerSetupEnabled: true,
	})
	assert.IsType(t, &SMTPEmailService{}, smtpService)

	assert.Nil(t, BuildEmailService(nil, EmailServiceConfig{TemplateDir: "templates"}))

	resendService := BuildEmailService(nil, EmailServiceConfig{
		TemplateDir:  "templates",
		ResendAPIKey: "re_test",
		FromName:     "SuperPlane",
		FromEmail:    "noreply@example.com",
	})
	assert.IsType(t, &ResendEmailService{}, resendService)
}

func TestNoopEmailService(t *testing.T) {
	service := NewNoopEmailService()

	require.NoError(t, service.SendMagicCodeEmail("user@example.com", "123456", "https://example.com"))

	emails := service.SentMagicCodeEmails()
	require.Len(t, emails, 1)
	assert.Equal(t, SentMagicCodeEmail{
		ToEmail:   "user@example.com",
		Code:      "123456",
		MagicLink: "https://example.com",
	}, emails[0])

	emails[0].Code = "mutated"
	assert.Equal(t, "123456", service.SentMagicCodeEmails()[0].Code)

	service.Reset()
	assert.Empty(t, service.SentMagicCodeEmails())
}

func writeMagicCodeTemplates(t *testing.T, root string) {
	t.Helper()

	templateDir := filepath.Join(root, "email")
	require.NoError(t, os.MkdirAll(templateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "magic_code.txt"), []byte("Code {{.Code}}\n{{.MagicLink}}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "magic_code.html"), []byte("<p>Code {{.Code}}</p><a href=\"{{.MagicLink}}\">Open</a>"), 0o644))
}
