package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListSelectableLLMModels(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	t.Cleanup(func() {
		_ = database.Conn().Where("provider <> ?", "").Delete(&models.HostedLLMProvider{})
		_ = database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	resp, err := ListSelectableLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{})
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

	resp, err = ListSelectableLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Models, 2)
	assert.Equal(t, "byok::anthropic::claude-sonnet-4-6", resp.Models[0].Key)
	assert.Equal(t, "Your keys", resp.Models[0].Source.Name)
	assert.Equal(t, "hosted::anthropic::claude-sonnet-4-6", resp.Models[1].Key)
	assert.Equal(t, "SuperPlane", resp.Models[1].Source.Name)
	assert.Equal(t, "anthropic/claude-sonnet-4-6", resp.Models[1].Label)

	_, err = ListSelectableLLMModels(context.Background(), "not-a-uuid", &pb.ListSelectableLLMModelsRequest{})
	require.Error(t, err)

	_, err = ListSelectableLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{
		FactoryId: "not-a-uuid",
	})
	require.Error(t, err)

	_, err = ListSelectableLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListSelectableLLMModelsRequest{
		FactoryId: uuid.NewString(),
	})
	require.Error(t, err)
}
