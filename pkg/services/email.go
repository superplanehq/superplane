package services

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	log "github.com/sirupsen/logrus"
)

type EmailService interface {
	SendInvitationEmail(toEmail, toName, organizationName, invitationLink string) error
}

type SendGridEmailService struct {
	apiKey      string
	fromName    string
	fromEmail   string
	templateDir string
}

func NewSendGridEmailService(apiKey, fromName, fromEmail, templateDir string) *SendGridEmailService {
	return &SendGridEmailService{
		apiKey:      apiKey,
		fromName:    fromName,
		fromEmail:   fromEmail,
		templateDir: templateDir,
	}
}

type InvitationTemplateData struct {
	ToName           string
	OrganizationName string
	InvitationLink   string
}

func (s *SendGridEmailService) SendInvitationEmail(toEmail, toName, organizationName, invitationLink string) error {
	from := mail.NewEmail(s.fromName, s.fromEmail)
	to := mail.NewEmail(toName, toEmail)

	subject := fmt.Sprintf("You're invited to join %s on Superplane", organizationName)

	templateData := InvitationTemplateData{
		ToName:           toName,
		OrganizationName: organizationName,
		InvitationLink:   invitationLink,
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

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	client := sendgrid.NewSendClient(s.apiKey)
	response, err := client.Send(message)
	if err != nil {
		log.Errorf("Error sending invitation email to %s: %v", toEmail, err)
		return err
	}

	if response.StatusCode >= 300 {
		log.Errorf("SendGrid API returned error status %d for email to %s", response.StatusCode, toEmail)
		return fmt.Errorf("failed to send email: status code %d", response.StatusCode)
	}

	log.Infof("Invitation email sent successfully to %s", toEmail)
	return nil
}

func (s *SendGridEmailService) renderTemplate(templateName string, data InvitationTemplateData) (string, error) {
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
