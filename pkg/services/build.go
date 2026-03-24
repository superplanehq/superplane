package services

import (
	"github.com/superplanehq/superplane/pkg/crypto"
)

type EmailServiceConfig struct {
	TemplateDir       string
	OwnerSetupEnabled bool
	ResendAPIKey      string
	FromName          string
	FromEmail         string
}

// BuildEmailService creates an EmailService based on the provided configuration.
// Returns nil if required configuration is missing.
func BuildEmailService(encryptor crypto.Encryptor, cfg EmailServiceConfig) EmailService {
	if cfg.TemplateDir == "" {
		return nil
	}

	if cfg.OwnerSetupEnabled {
		settingsProvider := &DatabaseEmailSettingsProvider{Encryptor: encryptor}
		return NewSMTPEmailService(settingsProvider, cfg.TemplateDir)
	}

	if cfg.ResendAPIKey == "" || cfg.FromName == "" || cfg.FromEmail == "" {
		return nil
	}

	return NewResendEmailService(cfg.ResendAPIKey, cfg.FromName, cfg.FromEmail, cfg.TemplateDir)
}
