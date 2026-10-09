package githubapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	privateKeyAssociatedData    = "installation_github_app_private_key"
	webhookSecretAssociatedData = "installation_github_app_webhook_secret"
	clientSecretAssociatedData  = "installation_github_app_client_secret"
)

var (
	ErrAlreadyConfigured = errors.New("GitHub App is already configured")
	ErrGitHubAppMissing  = errors.New("installation GitHub App is not configured")
)

const (
	LoginClientMissing = "missing"
	LoginClientNeeded  = "needs_client"
	LoginClientReady   = "ready"
)

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
	clientSecret := ""
	if len(record.EncryptedClientSecret) > 0 {
		plain, err := encryptor.Decrypt(ctx, record.EncryptedClientSecret, []byte(clientSecretAssociatedData))
		if err != nil {
			return config.GitHubHostedAppConfig{}, fmt.Errorf("decrypt GitHub App client secret: %w", err)
		}
		clientSecret = string(plain)
	}

	cfg := config.GitHubHostedAppConfig{
		ID:            record.GitHubAppID,
		Slug:          record.Slug,
		PrivateKey:    string(privateKey),
		WebhookSecret: string(webhookSecret),
		ClientID:      record.ClientID,
		ClientSecret:  clientSecret,
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
	encryptedClientSecret := []byte{}
	if cfg.ClientSecret != "" {
		encryptedClientSecret, err = encryptor.Encrypt(ctx, []byte(cfg.ClientSecret), []byte(clientSecretAssociatedData))
		if err != nil {
			return fmt.Errorf("encrypt GitHub App client secret: %w", err)
		}
	}
	err = models.SaveInstallationGitHubApp(tx, cfg.ID, cfg.Slug, cfg.ClientID, privateKey, webhookSecret, encryptedClientSecret)
	if !errors.Is(err, models.ErrInstallationGitHubAppExists) {
		return err
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return ErrAlreadyConfigured
	}
	err = models.ReplaceInstallationGitHubAppOAuth(tx, cfg.ID, cfg.ClientID, encryptedClientSecret)
	if errors.Is(err, models.ErrInstallationGitHubAppExists) {
		return ErrAlreadyConfigured
	}
	return err
}

// LoginClientStatus reports whether sign-in can start, or the saved app still
// needs its login client. Creating another GitHub App does not add that client.
func LoginClientStatus(tx *gorm.DB) (string, string, error) {
	if strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")) != "" && strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")) != "" {
		return LoginClientReady, "", nil
	}
	if env := config.LoadGitHubHostedAppConfig(); env.Enabled() {
		return LoginClientReady, env.Slug, nil
	}
	if tx == nil {
		return LoginClientMissing, "", nil
	}

	record, err := models.FindInstallationGitHubApp(tx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return LoginClientMissing, "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("load installation GitHub App: %w", err)
	}
	if record.ClientID != "" && len(record.EncryptedClientSecret) > 0 {
		return LoginClientReady, record.Slug, nil
	}
	return LoginClientNeeded, record.Slug, nil
}

// SaveLoginClient stores the login client on the GitHub App that already
// exists. It does not replace that app.
func SaveLoginClient(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, clientID, clientSecret string) error {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if clientID == "" || clientSecret == "" {
		return errors.New("client id and client secret are required")
	}
	if config.LoadGitHubHostedAppConfig().Enabled() {
		return ErrAlreadyConfigured
	}

	record, err := models.FindInstallationGitHubApp(tx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrGitHubAppMissing
	}
	if err != nil {
		return fmt.Errorf("load installation GitHub App: %w", err)
	}
	if record.ClientID != "" {
		return ErrAlreadyConfigured
	}

	encryptedClientSecret, err := encryptor.Encrypt(ctx, []byte(clientSecret), []byte(clientSecretAssociatedData))
	if err != nil {
		return fmt.Errorf("encrypt GitHub App client secret: %w", err)
	}
	return models.ReplaceInstallationGitHubAppOAuth(tx, record.GitHubAppID, clientID, encryptedClientSecret)
}

// UserConnectReady reports whether GitHub account connection can start.
// Cloud sets GITHUB_CLIENT_ID. Self-host uses the client stored with the app.
func UserConnectReady(ctx context.Context) bool {
	if strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")) != "" && strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")) != "" {
		return true
	}
	cfg, err := ResolveProcess(ctx)
	return err == nil && cfg.ClientID != "" && cfg.ClientSecret != ""
}
