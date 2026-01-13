package services

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/resend/resend-go/v3"
	log "github.com/sirupsen/logrus"
)

type EmailService interface {
	SendInvitationEmail(toEmail, organizationName, invitationLink, inviterEmail string) error
	SendNotificationEmail(bccEmails []string, title, body, url, urlLabel string) error
}

type InvitationTemplateData struct {
	ToEmail          string
	OrganizationName string
	InvitationLink   string
	InviterEmail     string
}

type NotificationTemplateData struct {
	Title    string
	Body     string
	URL      string
	URLLabel string
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

func (s *ResendEmailService) SendInvitationEmail(toEmail, organizationName, invitationLink, inviterEmail string) error {
	templateData := InvitationTemplateData{
		ToEmail:          toEmail,
		OrganizationName: organizationName,
		InvitationLink:   invitationLink,
		InviterEmail:     inviterEmail,
	}

	plainTextContent, err := s.renderTemplate("invitation.txt", templateData)
	if err != nil {
		log.Errorf("Error rendering plain text template: %v", err)
		return fmt.Errorf("failed to render plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("invitation.html", templateData)
	if err != nil {
		log.Errorf("Error rendering HTML template: %v", err)
		return fmt.Errorf("failed to render HTML template: %w", err)
	}

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{toEmail},
		Subject: "You have been invited to join an organization on Superplane",
		Text:    plainTextContent,
		Html:    htmlContent,
	}

	response, err := s.client.Emails.Send(params)
	if err != nil {
		log.Errorf("Error sending invitation email to %s: %v", toEmail, err)
		return err
	}

	log.Infof("Invitation email sent successfully to %s (ID: %s)", toEmail, response.Id)
	return nil
}

func (s *ResendEmailService) SendNotificationEmail(bccEmails []string, title, body, url, urlLabel string) error {
	if len(bccEmails) == 0 {
		return nil
	}

	if title == "" {
		title = "Superplane Notification"
	}

	templateData := NotificationTemplateData{
		Title:    title,
		Body:     body,
		URL:      url,
		URLLabel: urlLabel,
	}

	plainTextContent, err := s.renderTemplate("notification.txt", templateData)
	if err != nil {
		log.Errorf("Error rendering notification plain text template: %v", err)
		return fmt.Errorf("failed to render notification plain text template: %w", err)
	}

	htmlContent, err := s.renderTemplate("notification.html", templateData)
	if err != nil {
		log.Errorf("Error rendering notification HTML template: %v", err)
		return fmt.Errorf("failed to render notification HTML template: %w", err)
	}

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{s.fromEmail},
		Bcc:     bccEmails,
		Subject: title,
		Text:    plainTextContent,
		Html:    htmlContent,
	}

	response, err := s.client.Emails.Send(params)
	if err != nil {
		log.Errorf("Error sending notification email: %v", err)
		return err
	}

	log.Infof("Notification email sent successfully (ID: %s)", response.Id)
	return nil
}

func (s *ResendEmailService) renderTemplate(templateName string, data any) (string, error) {
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
