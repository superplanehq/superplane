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

func Test__ListHostedLLMModels(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	require.NoError(t, db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{}).Error)
	t.Cleanup(func() {
		_ = database.Conn().Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	resp, err := ListHostedLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListHostedLLMModelsRequest{
		Provider: "anthropic",
	})
	require.NoError(t, err)
	assert.False(t, resp.Enabled)
	assert.Empty(t, resp.Models)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)

	resp, err = ListHostedLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListHostedLLMModelsRequest{
		Provider: "anthropic",
	})
	require.NoError(t, err)
	assert.True(t, resp.Enabled)
	require.Len(t, resp.Models, 2)
	assert.Equal(t, "claude-sonnet-4-6", resp.Models[0].Id)

	require.NoError(t, db.Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{}).Error)
	t.Cleanup(func() {
		_ = database.Conn().Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
	})

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       false,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"openai/gpt-4.1", "moonshotai/kimi-k2.6"},
	})
	require.NoError(t, err)

	resp, err = ListHostedLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListHostedLLMModelsRequest{
		Provider: "openrouter",
	})
	require.NoError(t, err)
	assert.True(t, resp.Enabled)
	require.Len(t, resp.Models, 2)
	assert.Equal(t, "openai/gpt-4.1", resp.Models[0].Id)

	_, err = ListHostedLLMModels(context.Background(), "not-a-uuid", &pb.ListHostedLLMModelsRequest{Provider: "anthropic"})
	require.Error(t, err)

	_, err = ListHostedLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListHostedLLMModelsRequest{
		Provider:  "anthropic",
		FactoryId: "not-a-uuid",
	})
	require.Error(t, err)

	_, err = ListHostedLLMModels(context.Background(), r.Organization.ID.String(), &pb.ListHostedLLMModelsRequest{
		Provider:  "anthropic",
		FactoryId: uuid.NewString(),
	})
	require.Error(t, err)
}
