package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
)

func Test__ListSelectableLLMModels(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		_ = database.Conn().Where("provider <> ?", "").Delete(&models.HostedLLMProvider{})
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	resp, err := ListSelectableLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Models)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)
	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
	})
	require.NoError(t, err)

	resp, err = ListSelectableLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Models, 2)
	assert.Equal(t, "byok::anthropic::claude-sonnet-4-6", resp.Models[0].Key)
	assert.Equal(t, "Your keys", resp.Models[0].Source.Name)
	assert.Equal(t, "hosted::anthropic::claude-sonnet-4-6", resp.Models[1].Key)
	assert.Equal(t, "SuperPlane", resp.Models[1].Source.Name)
	assert.Equal(t, "anthropic/claude-sonnet-4-6", resp.Models[1].Label)

	_, err = ListSelectableLLMModels(context.Background(), r.Registry, "not-a-uuid", &pb.ListSelectableLLMModelsRequest{})
	require.Error(t, err)

	_, err = ListSelectableLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{
		FactoryId: "not-a-uuid",
	})
	require.Error(t, err)

	_, err = ListSelectableLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{
		FactoryId: uuid.NewString(),
	})
	require.Error(t, err)
}

func Test__ListSelectableLLMModels__EnablesConnectedKeyModelsByDefault(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "openai", support.RandomName("openai"), map[string]any{})
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)
	r.Registry.Integrations["openai"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		ListResources: func(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
			return []core.IntegrationResource{{ID: "gpt-5"}, {ID: "gpt-4.1"}}, nil
		},
	})

	resp, err := ListSelectableLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{})
	require.NoError(t, err)
	keys := make([]string, 0, len(resp.Models))
	for _, model := range resp.Models {
		keys = append(keys, model.Key)
	}
	assert.ElementsMatch(t, []string{"byok::openai::gpt-5", "byok::openai::gpt-4.1"}, keys)

	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{})
	require.NoError(t, err)
	resp, err = ListSelectableLLMModels(context.Background(), r.Registry, r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Models)
}
