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

func TestBuildMultipartEmail_StripsCRLFFromHeaders(t *testing.T) {
	msg, err := buildMultipartEmail(
		"Sender\r\nBcc: evil@external.com <sender@example.com>",
		[]string{"to@example.com\r\nCc: attacker@external.com"},
		nil,
		"Urgent\r\nFrom: ceo@victim.com\r\nBcc: attacker@external.com",
		"plain body",
		"<p>html body</p>",
	)
	require.NoError(t, err)

	assert.Contains(t, msg, "Subject: UrgentFrom: ceo@victim.comBcc: attacker@external.com")
	assert.Contains(t, msg, "From: SenderBcc: evil@external.com <sender@example.com>")
	assert.Contains(t, msg, "To: to@example.comCc: attacker@external.com")
	assert.NotContains(t, msg, "\r\nFrom: ceo@victim.com")
	assert.NotContains(t, msg, "\r\nBcc: attacker@external.com")
	assert.NotContains(t, msg, "\r\nCc: attacker@external.com")
	assert.Equal(t, 1, countHeaderLines(msg, "From:"))
	assert.Equal(t, 1, countHeaderLines(msg, "Subject:"))
	assert.Equal(t, 1, countHeaderLines(msg, "To:"))
}

func countHeaderLines(message, headerPrefix string) int {
	headerEnd := strings.Index(message, "\r\n\r\n")
	if headerEnd < 0 {
		return 0
	}

	count := 0
	for _, line := range strings.Split(message[:headerEnd], "\r\n") {
		if strings.HasPrefix(line, headerPrefix) {
			count++
		}
	}
	return count
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

func TestSMTPEmailService_SendWorkOrderNotificationEmail(t *testing.T) {
	tmpDir := t.TempDir()
	writeWorkOrderNotificationTemplates(t, tmpDir)

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

	err := service.SendWorkOrderNotificationEmail("user@example.com", "[SP-42] New comment", WorkOrderNotificationTemplateData{
		Summary:        "Ana commented on SP-42.",
		WorkOrderKey:   "SP-42",
		WorkOrderTitle: "Fix login",
		Detail:         "Looks good",
		WorkOrderLink:  "https://app.superplane.com/org/workspaces/sp/work-order/42",
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"user@example.com"}, fakeClient.rcpt)
	message := fakeClient.message.String()
	assert.Contains(t, message, "Subject: [SP-42] New comment")
	assert.Contains(t, message, "Ana commented on SP-42.")
	assert.Contains(t, message, "Looks good")
}

func TestResendEmailService_SendWorkOrderNotificationEmail_MissingTemplates(t *testing.T) {
	t.Run("missing plain text template", func(t *testing.T) {
		service := NewResendEmailService("re_test", "SuperPlane", "noreply@example.com", t.TempDir())
		err := service.SendWorkOrderNotificationEmail("user@example.com", "[SP-1] Update", WorkOrderNotificationTemplateData{
			Summary: "A work order changed.",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "plain text template")
	})

	t.Run("missing html template", func(t *testing.T) {
		tmpDir := t.TempDir()
		templateDir := filepath.Join(tmpDir, "email")
		require.NoError(t, os.MkdirAll(templateDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(templateDir, "work_order_notification.txt"), []byte("{{.Summary}}"), 0o644))

		service := NewResendEmailService("re_test", "SuperPlane", "noreply@example.com", tmpDir)
		err := service.SendWorkOrderNotificationEmail("user@example.com", "[SP-1] Update", WorkOrderNotificationTemplateData{
			Summary: "A work order changed.",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "HTML template")
	})
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

	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "work_order_notification.txt"), []byte("{{.Summary}} {{.WorkOrderLink}}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "work_order_notification.html"), []byte("<a href=\"{{.WorkOrderLink}}\">{{.Summary}}</a>"), 0o644))

	workOrder := WorkOrderNotificationTemplateData{
		Summary:       "Ana commented",
		WorkOrderLink: "https://app.superplane.com/org/workspaces/sp/work-order/42?tab=activity",
	}
	text, err = renderEmailTemplate(tmpDir, "work_order_notification.txt", workOrder)
	require.NoError(t, err)
	assert.Contains(t, text, "tab=activity")

	html, err = renderEmailTemplate(tmpDir, "work_order_notification.html", workOrder)
	require.NoError(t, err)
	assert.Contains(t, html, "tab=activity")
}

func TestWorkOrderNotificationTemplates_KanbanCard(t *testing.T) {
	templateRoot := filepath.Join("..", "..", "templates")
	data := WorkOrderNotificationTemplateData{
		Summary:          "Ana Souza commented on SP-52.",
		WorkOrderKey:     "SP-52",
		WorkOrderTitle:   "Metrics on list of lines",
		Detail:           "Looks good",
		DetailCtaLabel:   "Review PR #52",
		DetailCtaURL:     "https://github.com/example/repo/pull/52",
		WorkOrderLink:    "https://app.superplane.com/org/workspaces/sp/work-order/52",
		StatusLabel:      "Waiting",
		StatusFg:         "#b45309",
		StatusBg:         "#fffbeb",
		StatusBorder:     "#fde68a",
		StatusDot:        "#f59e0b",
		LineStepLabel:    "Bugs · CI Loop",
		UpdatedLabel:     "8h ago",
		AssigneeInitials: "AS",
		AssigneeOverflow: "+1",
	}

	html, err := renderEmailTemplate(templateRoot, "work_order_notification.html", data)
	require.NoError(t, err)
	assert.Contains(t, html, "Waiting")
	assert.Contains(t, html, "SP-52")
	assert.Contains(t, html, "Metrics on list of lines")
	assert.Contains(t, html, "Bugs · CI Loop")
	assert.Contains(t, html, "8h ago")
	assert.Contains(t, html, "AS")
	assert.Contains(t, html, "&#43;1")
	assert.Contains(t, html, "#fffbeb")
	assert.Contains(t, html, "https://app.superplane.com/org/workspaces/sp/work-order/52")
	assert.Contains(t, html, "href=\"https://github.com/example/repo/pull/52\"")
	assert.Contains(t, html, "Review PR #52")

	text, err := renderEmailTemplate(templateRoot, "work_order_notification.txt", data)
	require.NoError(t, err)
	assert.Contains(t, text, "Waiting")
	assert.Contains(t, text, "SP-52")
	assert.Contains(t, text, "Bugs · CI Loop")
	assert.Contains(t, text, "8h ago")
	assert.Contains(t, text, "Review PR #52: https://github.com/example/repo/pull/52")
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
	require.NoError(t, service.SendWorkOrderNotificationEmail("owner@example.com", "[SP-1] New comment", WorkOrderNotificationTemplateData{
		Summary: "A comment was added.",
	}))

	emails := service.SentMagicCodeEmails()
	require.Len(t, emails, 1)
	assert.Equal(t, SentMagicCodeEmail{
		ToEmail:   "user@example.com",
		Code:      "123456",
		MagicLink: "https://example.com",
	}, emails[0])

	notifications := service.SentWorkOrderNotificationEmails()
	require.Len(t, notifications, 1)
	assert.Equal(t, "owner@example.com", notifications[0].ToEmail)
	assert.Equal(t, "[SP-1] New comment", notifications[0].Subject)

	emails[0].Code = "mutated"
	assert.Equal(t, "123456", service.SentMagicCodeEmails()[0].Code)

	service.Reset()
	assert.Empty(t, service.SentMagicCodeEmails())
	assert.Empty(t, service.SentWorkOrderNotificationEmails())
}

func writeMagicCodeTemplates(t *testing.T, root string) {
	t.Helper()

	templateDir := filepath.Join(root, "email")
	require.NoError(t, os.MkdirAll(templateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "magic_code.txt"), []byte("Code {{.Code}}\n{{.MagicLink}}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "magic_code.html"), []byte("<p>Code {{.Code}}</p><a href=\"{{.MagicLink}}\">Open</a>"), 0o644))
}

func writeWorkOrderNotificationTemplates(t *testing.T, root string) {
	t.Helper()

	templateDir := filepath.Join(root, "email")
	require.NoError(t, os.MkdirAll(templateDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "work_order_notification.txt"), []byte("{{.Summary}}\n{{.Detail}}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "work_order_notification.html"), []byte("<p>{{.Summary}}</p><p>{{.Detail}}</p>"), 0o644))
}
