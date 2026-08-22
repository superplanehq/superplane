package contexts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/llm"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type HostedLLMContext struct {
	tx             *gorm.DB
	encryptor      crypto.Encryptor
	organizationID uuid.UUID
}

func NewHostedLLMContext(tx *gorm.DB, encryptor crypto.Encryptor, organizationID uuid.UUID) *HostedLLMContext {
	return &HostedLLMContext{
		tx:             tx,
		encryptor:      encryptor,
		organizationID: organizationID,
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

	return core.HostedLLMAccess{
		APIKey:        apiKey,
		BaseURL:       row.BaseURL,
		AllowedModels: append([]string{}, row.AllowedModels...),
	}, nil
}

func (c *HostedLLMContext) AssertCreditAvailable() error {
	if c.organizationID == uuid.Nil {
		return fmt.Errorf("organization is required for hosted LLM credit")
	}
	return models.AssertHostedCreditAvailable(c.tx, c.organizationID)
}
