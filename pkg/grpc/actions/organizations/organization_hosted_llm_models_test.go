package organizations

import (
	"context"
	"errors"
	"slices"
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
	"gorm.io/gorm"
)

func TestOrganizationHostedLLMModels(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	ctx := context.Background()
	orgID := r.Organization.ID.String()
	providers := []string{
		models.UsageProviderAnthropic,
		models.UsageProviderOpenRouter,
		models.UsageProviderOpenAI,
	}
	previous := map[string]*models.HostedLLMProvider{}
	for _, provider := range providers {
		previous[provider] = snapshotHostedLLMProvider(t, db, provider)
	}
	t.Cleanup(func() {
		_ = db.Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationHostedModelAllowlist{})
		for _, provider := range providers {
			restoreHostedLLMProvider(db, provider, previous[provider])
		}
	})
	for _, provider := range []models.HostedLLMProvider{
		{Provider: models.UsageProviderAnthropic, Enabled: true, APIKey: []byte("encrypted"), AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet"}},
		{Provider: models.UsageProviderOpenRouter, Enabled: true, APIKey: []byte("encrypted"), AllowedModels: datatypes.JSONSlice[string]{"anthropic/claude-sonnet"}},
	} {
		_, err := models.UpsertHostedLLMProvider(db, provider)
		require.NoError(t, err)
	}

	anthropicKey := models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderAnthropic, "claude-sonnet")
	openRouterKey := models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderOpenRouter, "anthropic/claude-sonnet")
	list, err := ListOrganizationHostedLLMModels(ctx, orgID)
	require.NoError(t, err)
	assert.ElementsMatch(t, list.Candidates, list.Selected)
	assertHostedCandidate(t, list.Candidates, anthropicKey, "anthropic/claude-sonnet", models.UsageProviderAnthropic)
	assertHostedCandidate(t, list.Candidates, openRouterKey, "anthropic/claude-sonnet", models.UsageProviderOpenRouter)
	row, err := models.FindOrganizationHostedModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic)
	require.NoError(t, err)
	assert.Nil(t, row, "reading must not save a selection")

	for _, invalid := range [][]string{
		{models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderAnthropic, "not-enabled")},
		{models.FormatSelectableLLMModelKey("byok", models.UsageProviderAnthropic, "claude-sonnet")},
		{anthropicKey, anthropicKey},
	} {
		_, err := UpdateOrganizationHostedLLMModels(ctx, orgID, &pb.UpdateOrganizationHostedLLMModelsRequest{AllowedModels: invalid})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	}
	saved, err := UpdateOrganizationHostedLLMModels(ctx, orgID, &pb.UpdateOrganizationHostedLLMModelsRequest{AllowedModels: []string{anthropicKey}})
	require.NoError(t, err)
	require.Len(t, saved.Selected, 1)
	assert.Equal(t, anthropicKey, saved.Selected[0].Key)
	for _, provider := range models.KnownHostedLLMProviders() {
		row, err := models.FindOrganizationHostedModelAllowlist(db, r.Organization.ID, provider)
		require.NoError(t, err)
		require.NotNil(t, row, "every provider must be saved, including unconfigured providers")
	}

	const probe = "organization-selection-probe"
	require.NoError(t, addHostedLLMModel(db, models.UsageProviderOpenAI, probe))
	probeKey := models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderOpenAI, probe)
	list, err = ListOrganizationHostedLLMModels(ctx, orgID)
	require.NoError(t, err)
	assertHostedCandidate(t, list.Candidates, probeKey, "openai/"+probe, models.UsageProviderOpenAI)
	require.Len(t, list.Selected, 1)
	assert.Equal(t, anthropicKey, list.Selected[0].Key)

	_, err = UpdateOrganizationHostedLLMModels(ctx, orgID, &pb.UpdateOrganizationHostedLLMModelsRequest{})
	require.NoError(t, err)
	list, err = ListOrganizationHostedLLMModels(ctx, orgID)
	require.NoError(t, err)
	assertHostedCandidate(t, list.Candidates, probeKey, "openai/"+probe, models.UsageProviderOpenAI)
	assert.Empty(t, list.Selected)
}

func snapshotHostedLLMProvider(t *testing.T, db *gorm.DB, provider string) *models.HostedLLMProvider {
	t.Helper()
	row, err := models.FindHostedLLMProvider(db, provider)
	if errors.Is(err, models.ErrHostedLLMProviderNotFound) {
		return nil
	}
	require.NoError(t, err)
	copy := *row
	copy.AllowedModels = append(datatypes.JSONSlice[string]{}, row.AllowedModels...)
	return &copy
}

func restoreHostedLLMProvider(db *gorm.DB, provider string, row *models.HostedLLMProvider) {
	if row == nil {
		_ = db.Where("provider = ?", provider).Delete(&models.HostedLLMProvider{}).Error
		return
	}
	_, _ = models.UpsertHostedLLMProvider(db, *row)
}

func addHostedLLMModel(db *gorm.DB, provider, model string) error {
	row, err := models.FindHostedLLMProvider(db, provider)
	if errors.Is(err, models.ErrHostedLLMProviderNotFound) {
		_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
			Provider:      provider,
			Enabled:       true,
			APIKey:        []byte("encrypted"),
			AllowedModels: datatypes.JSONSlice[string]{model},
		})
		return err
	}
	if err != nil {
		return err
	}
	allowed := append(datatypes.JSONSlice[string]{}, row.AllowedModels...)
	if !slices.Contains(allowed, model) {
		allowed = append(allowed, model)
	}
	apiKey := row.APIKey
	if len(apiKey) == 0 {
		apiKey = []byte("encrypted")
	}
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		ID:            row.ID,
		Provider:      provider,
		Enabled:       true,
		APIKey:        apiKey,
		ManagementKey: row.ManagementKey,
		BaseURL:       row.BaseURL,
		AllowedModels: allowed,
		CreatedAt:     row.CreatedAt,
	})
	return err
}

func assertHostedCandidate(t *testing.T, candidates []*pb.OrganizationHostedLLMModel, key, label, provider string) {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.Key != key {
			continue
		}
		assert.Equal(t, label, candidate.Label)
		assert.Equal(t, provider, candidate.Provider)
		return
	}
	t.Fatalf("candidate %s not found", key)
}
