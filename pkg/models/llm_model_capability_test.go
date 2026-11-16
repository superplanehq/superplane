package models_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestCapabilityPublicationsRollBackAllBatches(t *testing.T) {
	for _, publish := range []struct {
		name  string
		write func(*gorm.DB, string, string, []models.LLMModelCapability) error
	}{{"replace", models.ReplaceLLMModelCapabilities}, {"upsert", models.UpsertLLMModelCapabilities}} {
		t.Run(publish.name, func(t *testing.T) {
			support.Setup(t)
			db := database.DB(t.Context())
			provider, endpoint := models.UsageProviderAnthropic, "https://provider.example/v1"
			original := []models.LLMModelCapability{{Model: "cached", ThinkingLevels: datatypes.JSONSlice[string]{"high"}, RefreshedAt: time.Now()}}
			require.NoError(t, publish.write(db, provider, endpoint, original))
			before, err := models.ListLLMModelCapabilities(db, provider)
			require.NoError(t, err)
			// Reject a row in the second insert batch after the first batch succeeds.
			constraint := "test_capability_publication"
			require.NoError(t, db.Exec("ALTER TABLE llm_model_capabilities ADD CONSTRAINT "+constraint+" CHECK (model <> 'rejected')").Error)
			t.Cleanup(func() {
				require.NoError(t, db.WithContext(context.Background()).Exec("ALTER TABLE llm_model_capabilities DROP CONSTRAINT "+constraint).Error)
			})
			next := make([]models.LLMModelCapability, 251)
			for i := range next {
				next[i] = models.LLMModelCapability{Model: fmt.Sprintf("new-%03d", i), RefreshedAt: time.Now()}
			}
			next[250].Model = "rejected"
			require.Error(t, publish.write(db, provider, endpoint, next))
			after, err := models.ListLLMModelCapabilities(db, provider)
			require.NoError(t, err)
			assert.ElementsMatch(t, before, after)
		})
	}
}
