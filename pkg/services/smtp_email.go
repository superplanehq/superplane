package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"path/filepath"
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

func (s *SMTPEmailService) SendInvitationEmail(toEmail, organizationName, invitationLink, inviterEmail string) error {
	settings, err := s.settingsProvider.GetSMTPSettings(context.Background())
	if err != nil {
		return err
	}

	templateData := InvitationTemplateData{
		ToEmail:          toEmail,
		OrganizationName: organizationName,
		InvitationLink:   invitationLink,
		InviterEmail:     inviterEmail,
	}

	plainTextContent, err := s.renderTemplate("invitation.txt", templateData)
	if err != nil {
		return fmt.Errorf("failed to render invitation plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("invitation.html", templateData)
	if err != nil {
		return fmt.Errorf("failed to render invitation HTML template: %w", err)
	}

	subject := "You have been invited to join an organization on SuperPlane"
	return s.sendEmail(settings, []string{toEmail}, nil, subject, plainTextContent, htmlContent)
}

func (s *SMTPEmailService) SendNotificationEmail(bccEmails []string, title, body, url, urlLabel string) error {
	if len(bccEmails) == 0 {
		return nil
	}

	settings, err := s.settingsProvider.GetSMTPSettings(context.Background())
	if err != nil {
		return err
	}

	if title == "" {
		title = "SuperPlane Notification"
	}

	templateData := NotificationTemplateData{
		Title:    title,
		Body:     body,
		URL:      url,
		URLLabel: urlLabel,
	}

	plainTextContent, err := s.renderTemplate("notification.txt", templateData)
	if err != nil {
		return fmt.Errorf("failed to render notification plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("notification.html", templateData)
	if err != nil {
		return fmt.Errorf("failed to render notification HTML template: %w", err)
	}

	return s.sendEmail(settings, nil, bccEmails, title, plainTextContent, htmlContent)
}

func (s *SMTPEmailService) renderTemplate(templateName string, data any) (string, error) {
	templatePath := filepath.Join(s.templateDir, "email", templateName)
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return buf.String(), nil
}

func (s *SMTPEmailService) sendEmail(settings *SMTPSettings, to []string, bcc []string, subject, textBody, htmlBody string) error {
	from := formatFrom(settings.FromName, settings.FromEmail)
	if settings.Host == "" || settings.Port == 0 || settings.FromEmail == "" {
		return fmt.Errorf("smtp settings are incomplete")
	}

	recipients := append([]string{}, to...)
	recipients = append(recipients, bcc...)
	if len(recipients) == 0 {
		return nil
	}

	message, err := buildMultipartEmail(from, to, bcc, subject, textBody, htmlBody)
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

func buildMultipartEmail(from string, to []string, bcc []string, subject, textBody, htmlBody string) (string, error) {
	boundary, err := randomBoundary()
	if err != nil {
		return "", err
	}

	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"", boundary),
	}

	if len(to) > 0 {
		headers = append(headers, fmt.Sprintf("To: %s", strings.Join(to, ", ")))
	}

	message := strings.Join(headers, "\r\n") + "\r\n\r\n"
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n"
	message += textBody + "\r\n\r\n"
	message += fmt.Sprintf("--%s\r\n", boundary)
	message += "Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"
	message += htmlBody + "\r\n\r\n"
	message += fmt.Sprintf("--%s--\r\n", boundary)

	return message, nil
}

func formatFrom(name, email string) string {
	if name == "" {
		return email
	}

	return fmt.Sprintf("%s <%s>", name, email)
}

func randomBoundary() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
