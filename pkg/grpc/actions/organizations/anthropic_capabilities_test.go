package organizations

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func TestBYOKModelRefreshPublishesClaudeEffortsWithoutChangingSelection(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	integration := models.NewIntegration(uuid.New(), r.Organization.ID, "claude", "claude", map[string]any{})
	integration.State = models.IntegrationStateReady
	require.NoError(t, db.Create(integration).Error)
	_, err := models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.NewJSONSlice([]string{"claude-new-model"}))
	require.NoError(t, err)
	r.Registry.Integrations["claude"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		ListResources: func(string, core.ListResourcesContext) ([]core.IntegrationResource, error) {
			return []core.IntegrationResource{
				{ID: "claude-new-model", ThinkingLevels: []string{"max", "low", "minimal"}},
				{ID: "claude-other-model", ThinkingLevels: []string{"high"}},
			}, nil
		},
	})
	_, err = ListBYOKLLMModels(t.Context(), r.Registry, r.Organization.ID.String(), &pb.ListBYOKLLMModelsRequest{Provider: models.UsageProviderAnthropic})
	require.NoError(t, err)
	response, err := ListSelectableLLMModels(t.Context(), nil, r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{})
	require.NoError(t, err)
	byokModels := map[string][]string{}
	for _, model := range response.Models {
		if model.Source.Id == models.UsageFundingSourceBYOK && model.Provider.Id == models.UsageProviderAnthropic {
			byokModels[model.Model.Id] = model.ThinkingLevels
		}
	}
	assert.Equal(t, map[string][]string{"claude-new-model": {"low", "max"}}, byokModels)
}
