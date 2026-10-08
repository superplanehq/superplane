package githubapp

import (
	"context"
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	privateKeyAssociatedData    = "installation_github_app_private_key"
	webhookSecretAssociatedData = "installation_github_app_webhook_secret"
)

var ErrAlreadyConfigured = errors.New("GitHub App is already configured")

// Resolve returns the public GitHub App for this request. Process environment
// wins so Cloud keeps env-only credentials. Self-host stores the app on the
// installation after the first-run create step.
func Resolve(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor) (config.GitHubHostedAppConfig, error) {
	env := config.LoadGitHubHostedAppConfig()
	if env.Enabled() {
		return env, nil
	}
	if tx == nil || encryptor == nil {
		return config.GitHubHostedAppConfig{}, nil
	}

	record, err := models.FindInstallationGitHubApp(tx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return config.GitHubHostedAppConfig{}, nil
	}
	if err != nil {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("load installation GitHub App: %w", err)
	}

	privateKey, err := encryptor.Decrypt(ctx, record.EncryptedPrivateKey, []byte(privateKeyAssociatedData))
	if err != nil {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("decrypt GitHub App private key: %w", err)
	}
	webhookSecret, err := encryptor.Decrypt(ctx, record.EncryptedWebhookSecret, []byte(webhookSecretAssociatedData))
	if err != nil {
		return config.GitHubHostedAppConfig{}, fmt.Errorf("decrypt GitHub App webhook secret: %w", err)
	}

	cfg := config.GitHubHostedAppConfig{
		ID:            record.GitHubAppID,
		Slug:          record.Slug,
		PrivateKey:    string(privateKey),
		WebhookSecret: string(webhookSecret),
	}
	if !cfg.Enabled() {
		return config.GitHubHostedAppConfig{}, nil
	}
	return cfg, nil
}

// ResolveProcess reads the public GitHub App using the process encryptor.
func ResolveProcess(ctx context.Context) (config.GitHubHostedAppConfig, error) {
	env := config.LoadGitHubHostedAppConfig()
	if env.Enabled() {
		return env, nil
	}
	encryptor, err := crypto.FromEnv()
	if err != nil {
		return config.GitHubHostedAppConfig{}, nil
	}
	return Resolve(ctx, database.DB(ctx), encryptor)
}

func Enabled(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor) bool {
	cfg, err := Resolve(ctx, tx, encryptor)
	return err == nil && cfg.Enabled()
}

func ProcessEnabled(ctx context.Context) bool {
	cfg, err := ResolveProcess(ctx)
	return err == nil && cfg.Enabled()
}

func Save(
	ctx context.Context,
	tx *gorm.DB,
	encryptor crypto.Encryptor,
	cfg config.GitHubHostedAppConfig,
) error {
	if !cfg.Enabled() {
		return errors.New("GitHub App credentials are incomplete")
	}
	privateKey, err := encryptor.Encrypt(ctx, []byte(cfg.PrivateKey), []byte(privateKeyAssociatedData))
	if err != nil {
		return fmt.Errorf("encrypt GitHub App private key: %w", err)
	}
	webhookSecret, err := encryptor.Encrypt(ctx, []byte(cfg.WebhookSecret), []byte(webhookSecretAssociatedData))
	if err != nil {
		return fmt.Errorf("encrypt GitHub App webhook secret: %w", err)
	}
	err = models.SaveInstallationGitHubApp(tx, cfg.ID, cfg.Slug, privateKey, webhookSecret)
	if errors.Is(err, models.ErrInstallationGitHubAppExists) {
		return ErrAlreadyConfigured
	}
	return err
}
