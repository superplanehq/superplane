package services

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/smtp"
	"strings"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type SMTPSettings struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromName  string
	FromEmail string
	UseTLS    bool
}

type EmailSettingsProvider interface {
	GetSMTPSettings(ctx context.Context) (*SMTPSettings, error)
}

type DatabaseEmailSettingsProvider struct {
	Encryptor crypto.Encryptor
}

func (p *DatabaseEmailSettingsProvider) GetSMTPSettings(ctx context.Context) (*SMTPSettings, error) {
	settings, err := models.FindEmailSettings(models.EmailProviderSMTP)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("smtp settings not configured")
		}
		return nil, err
	}

	password := ""
	if len(settings.SMTPPassword) > 0 {
		decrypted, err := p.Encryptor.Decrypt(ctx, settings.SMTPPassword, []byte("smtp_password"))
		if err != nil {
			return nil, err
		}
		password = string(decrypted)
	}

	return &SMTPSettings{
		Host:      settings.SMTPHost,
		Port:      settings.SMTPPort,
		Username:  settings.SMTPUsername,
		Password:  password,
		FromName:  settings.SMTPFromName,
		FromEmail: settings.SMTPFromEmail,
		UseTLS:    settings.SMTPUseTLS,
	}, nil
}

type SMTPEmailService struct {
	settingsProvider EmailSettingsProvider
	templateDir      string
}

func NewSMTPEmailService(settingsProvider EmailSettingsProvider, templateDir string) *SMTPEmailService {
	return &SMTPEmailService{
		settingsProvider: settingsProvider,
		templateDir:      templateDir,
	}
}

type smtpClient interface {
	Hello(localName string) error
	Extension(ext string) (bool, string)
	StartTLS(config *tls.Config) error
	Auth(auth smtp.Auth) error
	Mail(from string) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

type smtpDialer func(addr string) (smtpClient, error)

type smtpClientAdapter struct {
	client *smtp.Client
}

func (c *smtpClientAdapter) Hello(localName string) error {
	return c.client.Hello(localName)
}

func (c *smtpClientAdapter) Extension(ext string) (bool, string) {
	return c.client.Extension(ext)
}

func (c *smtpClientAdapter) StartTLS(config *tls.Config) error {
	return c.client.StartTLS(config)
}

func (c *smtpClientAdapter) Auth(auth smtp.Auth) error {
	return c.client.Auth(auth)
}

func (c *smtpClientAdapter) Mail(from string) error {
	return c.client.Mail(from)
}

func (c *smtpClientAdapter) Rcpt(to string) error {
	return c.client.Rcpt(to)
}

func (c *smtpClientAdapter) Data() (io.WriteCloser, error) {
	return c.client.Data()
}

func (c *smtpClientAdapter) Quit() error {
	return c.client.Quit()
}

func (c *smtpClientAdapter) Close() error {
	return c.client.Close()
}

var smtpDial smtpDialer = func(addr string) (smtpClient, error) {
	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}
	return &smtpClientAdapter{client: client}, nil
}

func (s *SMTPEmailService) SendMagicCodeEmail(toEmail, code, magicLink string) error {
	settings, err := s.settingsProvider.GetSMTPSettings(context.Background())
	if err != nil {
		return err
	}

	templateData := MagicCodeTemplateData{Code: code, MagicLink: magicLink}

	plainTextContent, err := s.renderTemplate("magic_code.txt", templateData)
	if err != nil {
		return fmt.Errorf("failed to render magic code plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("magic_code.html", templateData)
	if err != nil {
		return fmt.Errorf("failed to render magic code HTML template: %w", err)
	}

	subject := "Your SuperPlane sign-in code"
	return s.sendEmail(settings, outgoingMail{
		to:       []string{toEmail},
		subject:  subject,
		textBody: plainTextContent,
		htmlBody: htmlContent,
	})
}

func (s *SMTPEmailService) SendWorkOrderNotificationEmail(
	toEmail, subject string,
	data WorkOrderNotificationTemplateData,
) error {
	settings, err := s.settingsProvider.GetSMTPSettings(context.Background())
	if err != nil {
		return err
	}

	plainTextContent, err := s.renderTemplate("work_order_notification.txt", data)
	if err != nil {
		return fmt.Errorf("failed to render work order notification plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("work_order_notification.html", data)
	if err != nil {
		return fmt.Errorf("failed to render work order notification HTML template: %w", err)
	}

	return s.sendEmail(settings, outgoingMail{
		to:       []string{toEmail},
		subject:  subject,
		textBody: plainTextContent,
		htmlBody: htmlContent,
	})
}

func (s *SMTPEmailService) SendSupportFeedbackEmail(toEmail string, feedback SupportFeedback) error {
	settings, err := s.settingsProvider.GetSMTPSettings(context.Background())
	if err != nil {
		return err
	}

	data := feedback.TemplateData()
	plainTextContent, err := s.renderTemplate("support_feedback.txt", data)
	if err != nil {
		return fmt.Errorf("failed to render support feedback plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("support_feedback.html", data)
	if err != nil {
		return fmt.Errorf("failed to render support feedback HTML template: %w", err)
	}

	mail := outgoingMail{
		to:       []string{toEmail},
		subject:  feedback.EmailSubject(),
		textBody: plainTextContent,
		htmlBody: htmlContent,
		replyTo:  strings.TrimSpace(feedback.UserEmail),
	}
	if feedback.Attachment != nil {
		mail.attachments = []SupportFeedbackAttachment{*feedback.Attachment}
	}
	return s.sendEmail(settings, mail)
}

func (s *SMTPEmailService) renderTemplate(templateName string, data any) (string, error) {
	return renderEmailTemplate(s.templateDir, templateName, data)
}

type outgoingMail struct {
	to          []string
	bcc         []string
	subject     string
	textBody    string
	htmlBody    string
	replyTo     string
	attachments []SupportFeedbackAttachment
}

func (s *SMTPEmailService) sendEmail(settings *SMTPSettings, mail outgoingMail) error {
	from := formatFrom(settings.FromName, settings.FromEmail)
	if settings.Host == "" || settings.Port == 0 || settings.FromEmail == "" {
		return fmt.Errorf("smtp settings are incomplete")
	}

	recipients := append([]string{}, mail.to...)
	recipients = append(recipients, mail.bcc...)
	if len(recipients) == 0 {
		return nil
	}

	message, err := buildMultipartEmail(from, mail)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", settings.Host, settings.Port)
	conn, err := smtpDial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.Hello(settings.Host); err != nil {
		return err
	}

	if settings.UseTLS {
		if ok, _ := conn.Extension("STARTTLS"); !ok {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
		tlsConfig := &tls.Config{
			ServerName: settings.Host,
			MinVersion: tls.VersionTLS12,
		}
		if err := conn.StartTLS(tlsConfig); err != nil {
			return err
		}
	}

	if settings.Username != "" {
		auth := smtp.PlainAuth("", settings.Username, settings.Password, settings.Host)
		if err := conn.Auth(auth); err != nil {
			return err
		}
	}

	if err := conn.Mail(settings.FromEmail); err != nil {
		return err
	}

	for _, recipient := range recipients {
		if err := conn.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := conn.Data()
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	return conn.Quit()
}

func buildMultipartEmail(from string, mail outgoingMail) (string, error) {
	alternativeBoundary, err := randomBoundary()
	if err != nil {
		return "", err
	}

	alternative := alternativePart(alternativeBoundary, mail.textBody, mail.htmlBody)

	headers := []string{
		fmt.Sprintf("From: %s", sanitizeSMTPHeaderValue(from)),
		fmt.Sprintf("Subject: %s", sanitizeSMTPHeaderValue(mail.subject)),
		"MIME-Version: 1.0",
	}
	if len(mail.to) > 0 {
		headers = append(headers, fmt.Sprintf("To: %s", sanitizeSMTPHeaderValue(strings.Join(mail.to, ", "))))
	}
	if mail.replyTo != "" {
		headers = append(headers, fmt.Sprintf("Reply-To: %s", sanitizeSMTPHeaderValue(mail.replyTo)))
	}

	if len(mail.attachments) == 0 {
		headers = append(headers, fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"", alternativeBoundary))
		return strings.Join(headers, "\r\n") + "\r\n\r\n" + alternative, nil
	}

	mixedBoundary, err := randomBoundary()
	if err != nil {
		return "", err
	}
	headers = append(headers, fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"", mixedBoundary))

	var b strings.Builder
	b.WriteString(strings.Join(headers, "\r\n"))
	b.WriteString("\r\n\r\n")
	fmt.Fprintf(&b, "--%s\r\n", mixedBoundary)
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", alternativeBoundary)
	b.WriteString(alternative)
	for _, attachment := range mail.attachments {
		fmt.Fprintf(&b, "--%s\r\n", mixedBoundary)
		b.WriteString(attachmentPart(attachment))
	}
	fmt.Fprintf(&b, "--%s--\r\n", mixedBoundary)
	return b.String(), nil
}

func alternativePart(boundary, textBody, htmlBody string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
	b.WriteString(textBody)
	b.WriteString("\r\n\r\n")
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
	b.WriteString(htmlBody)
	b.WriteString("\r\n\r\n")
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.String()
}

func attachmentPart(attachment SupportFeedbackAttachment) string {
	contentType := attachment.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": attachment.Filename})

	var b strings.Builder
	fmt.Fprintf(&b, "Content-Type: %s; name=\"%s\"\r\n", contentType, sanitizeSMTPHeaderValue(attachment.Filename))
	fmt.Fprintf(&b, "Content-Disposition: %s\r\n", disposition)
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Content)))
	base64.StdEncoding.Encode(encoded, attachment.Content)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		b.Write(encoded[i:end])
		b.WriteString("\r\n")
	}
	b.WriteString("\r\n")
	return b.String()
}

func formatFrom(name, email string) string {
	if name == "" {
		return email
	}

	return fmt.Sprintf("%s <%s>", name, email)
}

// sanitizeSMTPHeaderValue strips CR/LF so caller-influenced values cannot inject SMTP headers.
func sanitizeSMTPHeaderValue(value string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}

func randomBoundary() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
