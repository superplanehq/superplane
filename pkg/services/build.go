package services

import (
	"os"

	"github.com/superplanehq/superplane/pkg/crypto"
)

// BuildEmailService creates an EmailService based on environment configuration.
// Returns nil if required configuration is missing.
func BuildEmailService(encryptor crypto.Encryptor, templateDir string) EmailService {
	if templateDir == "" {
		return nil
	}

	if os.Getenv("OWNER_SETUP_ENABLED") == "yes" {
		settingsProvider := &DatabaseEmailSettingsProvider{Encryptor: encryptor}
		return NewSMTPEmailService(settingsProvider, templateDir)
	}

	resendAPIKey := os.Getenv("RESEND_API_KEY")
	fromName := os.Getenv("EMAIL_FROM_NAME")
	fromEmail := os.Getenv("EMAIL_FROM_ADDRESS")
	if resendAPIKey == "" || fromName == "" || fromEmail == "" {
		return nil
	}

	return NewResendEmailService(resendAPIKey, fromName, fromEmail, templateDir)
}
