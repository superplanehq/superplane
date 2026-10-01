package organizations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func TestOrganizationHostedLLMModels(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	ctx := context.Background()
	orgID := r.Organization.ID.String()
	t.Cleanup(func() {
		_ = db.Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationHostedModelAllowlist{})
		_ = db.Where("provider IN ?", models.KnownHostedLLMProviders()).Delete(&models.HostedLLMProvider{})
	})
	for _, provider := range []models.HostedLLMProvider{
		{Provider: "anthropic", Enabled: true, APIKey: []byte("encrypted"), AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet"}},
		{Provider: "openrouter", Enabled: true, APIKey: []byte("encrypted"), AllowedModels: datatypes.JSONSlice[string]{"anthropic/claude-sonnet"}},
	} {
		_, err := models.UpsertHostedLLMProvider(db, provider)
		require.NoError(t, err)
	}
	list, err := ListOrganizationHostedLLMModels(ctx, orgID)
	require.NoError(t, err)
	require.Len(t, list.Candidates, 2)
	assert.Equal(t, list.Candidates, list.Selected)
	assert.Equal(t, "anthropic/claude-sonnet", list.Candidates[0].Label)
	assert.NotEqual(t, list.Candidates[0].Key, list.Candidates[1].Key)
	row, err := models.FindOrganizationHostedModelAllowlist(db, r.Organization.ID, "anthropic")
	require.NoError(t, err)
	assert.Nil(t, row, "reading must not save a selection")

	key := models.FormatSelectableLLMModelKey("hosted", "anthropic", "claude-sonnet")
	for _, invalid := range [][]string{
		{models.FormatSelectableLLMModelKey("hosted", "anthropic", "not-enabled")},
		{models.FormatSelectableLLMModelKey("byok", "anthropic", "claude-sonnet")},
		{key, key},
	} {
		_, err := UpdateOrganizationHostedLLMModels(ctx, orgID, &pb.UpdateOrganizationHostedLLMModelsRequest{AllowedModels: invalid})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	}
	saved, err := UpdateOrganizationHostedLLMModels(ctx, orgID, &pb.UpdateOrganizationHostedLLMModelsRequest{AllowedModels: []string{key}})
	require.NoError(t, err)
	require.Len(t, saved.Selected, 1)
	assert.Equal(t, key, saved.Selected[0].Key)
	for _, provider := range models.KnownHostedLLMProviders() {
		row, err := models.FindOrganizationHostedModelAllowlist(db, r.Organization.ID, provider)
		require.NoError(t, err)
		require.NotNil(t, row, "every provider must be saved, including unconfigured providers")
	}
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: "openai", Enabled: true, APIKey: []byte("encrypted"), AllowedModels: datatypes.JSONSlice[string]{"gpt-5"},
	})
	require.NoError(t, err)
	list, err = ListOrganizationHostedLLMModels(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, list.Candidates, 3)
	require.Len(t, list.Selected, 1)
	assert.Equal(t, key, list.Selected[0].Key)

	_, err = UpdateOrganizationHostedLLMModels(ctx, orgID, &pb.UpdateOrganizationHostedLLMModelsRequest{})
	require.NoError(t, err)
	list, err = ListOrganizationHostedLLMModels(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, list.Candidates, 3)
	assert.Empty(t, list.Selected)
}
