package contexts

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// HostedLLMContext resolves installation-hosted provider credentials.
// Credit checks use a committed connection so remaining credit includes
// billed spend from other runs.
type HostedLLMContext struct {
	tx             *gorm.DB
	encryptor      crypto.Encryptor
	organizationID uuid.UUID
	factoryID      *uuid.UUID
}

func NewHostedLLMContext(tx *gorm.DB, encryptor crypto.Encryptor, organizationID uuid.UUID, factoryID *uuid.UUID) *HostedLLMContext {
	return &HostedLLMContext{
		tx:             tx,
		encryptor:      encryptor,
		organizationID: organizationID,
		factoryID:      factoryID,
	}
}

func (c *HostedLLMContext) Resolve(provider string) (core.HostedLLMAccess, error) {
	row, err := models.RequireEnabledHostedLLMProvider(c.tx, provider)
	if err != nil {
		return core.HostedLLMAccess{}, err
	}

	apiKey, err := llm.DecryptAPIKey(context.Background(), c.encryptor, row.Provider, row.APIKey)
	if err != nil {
		return core.HostedLLMAccess{}, err
	}

	allowed, err := models.ResolveSelectableLLMModels(c.tx, c.organizationID, c.factoryID, provider, models.UsageFundingSourceHosted)
	if err != nil {
		return core.HostedLLMAccess{}, err
	}

	return core.HostedLLMAccess{
		APIKey:        apiKey,
		BaseURL:       row.BaseURL,
		AllowedModels: allowed,
	}, nil
}

func (c *HostedLLMContext) AssertModelSelectable(provider, fundingSource, model string) error {
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required")
	}
	allowed, err := models.ModelIsSelectable(c.tx, c.organizationID, c.factoryID, provider, fundingSource, model)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("model %s is not on the selected-model list", model)
	}
	return nil
}

func (c *HostedLLMContext) AssertCreditAvailable() error {
	return models.AssertHostedRunAllowed(database.Conn(), c.organizationID, c.factoryID)
}

func (c *HostedLLMContext) DefaultModel() (core.DefaultHostedLLMModel, error) {
	defaultModel, err := models.GetInstallationDefaultHostedLLMModel(c.tx)
	if err != nil {
		return core.DefaultHostedLLMModel{}, err
	}
	return core.DefaultHostedLLMModel{
		Provider: defaultModel.Provider,
		Model:    defaultModel.Model,
	}, nil
}
