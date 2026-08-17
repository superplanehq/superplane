package services

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"path/filepath"
	"strings"
	texttemplate "text/template"

	"github.com/resend/resend-go/v3"
	log "github.com/sirupsen/logrus"
)

type EmailService interface {
	SendMagicCodeEmail(toEmail, code, magicLink string) error
	SendWorkOrderNotificationEmail(toEmail, subject string, data WorkOrderNotificationTemplateData) error
}

type MagicCodeTemplateData struct {
	Code      string
	MagicLink string
}

// WorkOrderNotificationTemplateData feeds the shared
// work_order_notification templates. One template renders every
// notification type; the consumer builds the summary and detail lines.
type WorkOrderNotificationTemplateData struct {
	// Summary is the one-line event description, for example
	// "Ana Souza commented on SP-42".
	Summary        string
	WorkOrderKey   string
	WorkOrderTitle string
	// Detail is optional supporting text (comment excerpt, state change).
	Detail        string
	WorkOrderLink string
}

type ResendEmailService struct {
	apiKey      string
	fromName    string
	fromEmail   string
	templateDir string
	client      *resend.Client
}

func NewResendEmailService(apiKey, fromName, fromEmail, templateDir string) *ResendEmailService {
	return &ResendEmailService{
		apiKey:      apiKey,
		fromName:    fromName,
		fromEmail:   fromEmail,
		templateDir: templateDir,
		client:      resend.NewClient(apiKey),
	}
}

func (s *ResendEmailService) SendMagicCodeEmail(toEmail, code, magicLink string) error {
	templateData := MagicCodeTemplateData{Code: code, MagicLink: magicLink}

	plainTextContent, err := s.renderTemplate("magic_code.txt", templateData)
	if err != nil {
		log.Errorf("Error rendering magic code plain text template: %v", err)
		return fmt.Errorf("failed to render magic code plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("magic_code.html", templateData)
	if err != nil {
		log.Errorf("Error rendering magic code HTML template: %v", err)
		return fmt.Errorf("failed to render magic code HTML template: %w", err)
	}

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{toEmail},
		Subject: "Your SuperPlane sign-in code",
		Text:    plainTextContent,
		Html:    htmlContent,
	}

	response, err := s.client.Emails.Send(params)
	if err != nil {
		log.Errorf("Error sending magic code email to %s: %v", toEmail, err)
		return err
	}

	log.Infof("Magic code email sent successfully to %s (ID: %s)", toEmail, response.Id)
	return nil
}

func (s *ResendEmailService) SendWorkOrderNotificationEmail(
	toEmail, subject string,
	data WorkOrderNotificationTemplateData,
) error {
	plainTextContent, err := s.renderTemplate("work_order_notification.txt", data)
	if err != nil {
		log.Errorf("Error rendering work order notification plain text template: %v", err)
		return fmt.Errorf("failed to render work order notification plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("work_order_notification.html", data)
	if err != nil {
		log.Errorf("Error rendering work order notification HTML template: %v", err)
		return fmt.Errorf("failed to render work order notification HTML template: %w", err)
	}

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{toEmail},
		Subject: subject,
		Text:    plainTextContent,
		Html:    htmlContent,
	}

	response, err := s.client.Emails.Send(params)
	if err != nil {
		log.Errorf("Error sending work order notification email to %s: %v", toEmail, err)
		return err
	}

	log.Infof("Work order notification email sent successfully to %s (ID: %s)", toEmail, response.Id)
	return nil
}

func (s *ResendEmailService) renderTemplate(templateName string, data any) (string, error) {
	return renderEmailTemplate(s.templateDir, templateName, data)
}

// renderEmailTemplate renders an email template from templateDir. It uses
// text/template for .txt files to avoid HTML-escaping URLs (html/template
// converts & to &amp; which breaks plain-text links), and html/template
// for everything else.
func renderEmailTemplate(templateDir, templateName string, data any) (string, error) {
	templatePath := filepath.Join(templateDir, "email", templateName)

	var buf bytes.Buffer

	if strings.HasSuffix(templateName, ".txt") {
		tmpl, err := texttemplate.ParseFiles(templatePath)
		if err != nil {
			return "", fmt.Errorf("failed to parse template %s: %w", templatePath, err)
		}
		if err = tmpl.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
		}
	} else {
		tmpl, err := htmltemplate.ParseFiles(templatePath)
		if err != nil {
			return "", fmt.Errorf("failed to parse template %s: %w", templatePath, err)
		}
		if err = tmpl.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("failed to execute template %s: %w", templatePath, err)
		}
	}

	return buf.String(), nil
}
