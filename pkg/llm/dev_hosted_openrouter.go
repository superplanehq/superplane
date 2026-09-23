package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SeedDevHostedOpenRouterFromEnv writes the SuperPlane-hosted OpenRouter
// provider when the local development seed is complete. It is a no-op when
// the seed is unset. A half-filled flag returns an error.
func SeedDevHostedOpenRouterFromEnv(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor) error {
	cfg, err := config.LoadDevHostedOpenRouterConfig()
	if err != nil {
		return err
	}
	if !cfg.Enabled() {
		return nil
	}
	return SeedDevHostedOpenRouter(ctx, tx, encryptor, cfg)
}

// SeedDevHostedOpenRouter replaces the OpenRouter hosted-provider row and
// optionally sets the installation default model.
func SeedDevHostedOpenRouter(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, cfg config.DevHostedOpenRouterConfig) error {
	if !cfg.Enabled() {
		return nil
	}
	if err := ValidateBaseURL(cfg.BaseURL); err != nil {
		return err
	}

	return tx.Transaction(func(inner *gorm.DB) error {
		apiKey, err := EncryptAPIKey(ctx, encryptor, models.UsageProviderOpenRouter, cfg.APIKey)
		if err != nil {
			return err
		}
		managementKey, err := EncryptManagementKey(ctx, encryptor, models.UsageProviderOpenRouter, cfg.ManagementKey)
		if err != nil {
			return err
		}

		_, err = models.UpsertHostedLLMProvider(inner, models.HostedLLMProvider{
			Provider:      models.UsageProviderOpenRouter,
			Enabled:       true,
			APIKey:        apiKey,
			ManagementKey: managementKey,
			BaseURL:       cfg.BaseURL,
			AllowedModels: datatypes.JSONSlice[string](cfg.Models),
		})
		if err != nil {
			return err
		}

		return syncDevDefaultHostedModel(inner, cfg)
	})
}

func syncDevDefaultHostedModel(tx *gorm.DB, cfg config.DevHostedOpenRouterConfig) error {
	settings, err := models.GetInstallationLLMSettings(tx)
	if err != nil {
		return err
	}

	current := models.InstallationDefaultHostedLLMModel(settings)
	nextModel := cfg.DefaultModel
	if nextModel == "" {
		if current.IsSet() {
			err = models.AssertDefaultHostedLLMModelAllowed(tx, current)
			if err == nil {
				return nil
			}
			if !errors.Is(err, models.ErrDefaultHostedModelNotOnAllowlist) {
				return err
			}
		}
		nextModel = cfg.Models[0]
	}

	next := *settings
	provider := models.UsageProviderOpenRouter
	next.DefaultHostedProvider = &provider
	next.DefaultHostedModel = &nextModel
	_, err = models.UpdateInstallationLLMSettings(tx, next)
	if err != nil {
		return fmt.Errorf("set development hosted default model: %w", err)
	}
	return nil
}
