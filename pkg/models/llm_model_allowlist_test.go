package models_test

import (
	"github.com/superplanehq/superplane/pkg/features"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ResolveSelectableLLMModels(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.Conn()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = database.Conn().Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
		_ = database.Conn().Where("factory_id = ?", factory.ID).Delete(&models.FactoryLLMModelAllowlist{})
	})

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)

	hosted, err := models.ResolveSelectableLLMModels(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted)
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-sonnet-4-6", "claude-opus-4-6"}, hosted)

	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
		"ignored-model",
	})
	require.Error(t, err)

	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, datatypes.JSONSlice[string]{
		"claude-sonnet-4-6",
	})
	require.NoError(t, err)

	hosted, err = models.ResolveSelectableLLMModels(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted)
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-sonnet-4-6"}, hosted)

	allowed, err := models.ModelIsSelectable(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, "claude-opus-4-6")
	require.NoError(t, err)
	assert.False(t, allowed)

	_, err = support.SaveProviderModelCatalog(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{
		"gpt-4.1",
		"gpt-4o",
	})
	require.NoError(t, err)

	byok, err := models.ResolveSelectableLLMModels(db, r.Organization.ID, nil, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-4.1", "gpt-4o"}, byok)

	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, nil)
	require.NoError(t, err)

	byok, err = models.ResolveSelectableLLMModels(db, r.Organization.ID, &factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-4.1", "gpt-4o"}, byok)

	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
	require.NoError(t, err)
	used := 0.0
	duration := int64(300)
	reset := time.Now().Add(time.Hour).Unix()
	limits := []models.CodexUsageLimit{{LimitID: "codex", Primary: &models.CodexUsageWindow{UsedPercent: &used, WindowDurationMins: &duration, ResetsAt: &reset}}}
	require.NoError(t, connection.StoreUsage(db, limits, []models.CodexModel{{Model: "gpt-5.4"}}, time.Now()))
	require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
	byok, err = models.ResolveSelectableLLMModels(db, r.Organization.ID, &factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-4.1", "gpt-4o"}, byok)
	allowedSubscriptionModel, err := models.RunnerAcceptsModel(db, r.Organization.ID, &factory.ID, "runnerCodex", map[string]any{"credentials": map[string]any{"source": "integration"}}, "gpt-5.4")
	require.NoError(t, err)
	assert.True(t, allowedSubscriptionModel, "API-key execution is independent of personal catalogs")
	allowedSubscriptionModel, err = models.RunnerAcceptsModel(db, r.Organization.ID, &factory.ID, "runnerCodex", map[string]any{"credentials": map[string]any{"source": "subscription"}}, "gpt-5.4", r.User)
	require.NoError(t, err)
	assert.True(t, allowedSubscriptionModel)
	byok, err = models.ResolveSelectableLLMModels(db, r.Organization.ID, &factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK)
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-4.1", "gpt-4o"}, byok)

	allowed, err = models.ModelIsSelectable(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, "claude-opus-4-6")
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = models.ModelIsSelectable(db, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceHosted, "")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func Test__FactoryLLMModelPolicy__InheritanceAndExplicitEmpty(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	parent := []string{"native-default", "native-model"}
	for _, scenario := range []struct {
		name            string
		selected        []string
		inherit         *bool
		want            []string
		wantInheritance bool
	}{
		{name: "legacy empty inherits", want: parent, wantInheritance: true},
		{name: "legacy selection restricts", selected: []string{"native-model"}, want: []string{"native-model"}},
		{name: "explicit empty disables", inherit: boolPointer(false), want: []string{}},
		{name: "explicit selection restricts", inherit: boolPointer(false), selected: []string{"native-model"}, want: []string{"native-model"}},
		{name: "explicit inheritance restores catalog", inherit: boolPointer(true), want: parent, wantInheritance: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			saved, err := models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceSubscription, datatypes.JSONSlice[string](scenario.selected), scenario.inherit)
			require.NoError(t, err)
			assert.Equal(t, scenario.wantInheritance, saved.InheritsParent())
			assert.Equal(t, scenario.inherit, saved.InheritParent)
			resolved, err := models.ApplyFactoryLLMModelPolicy(db, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceSubscription, parent)
			require.NoError(t, err)
			assert.Equal(t, scenario.want, resolved)
			other, err := models.ApplyFactoryLLMModelPolicy(db, otherFactory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceSubscription, parent)
			require.NoError(t, err)
			assert.Equal(t, parent, other)
			byok, err := models.ApplyFactoryLLMModelPolicy(db, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceBYOK, parent)
			require.NoError(t, err)
			assert.Equal(t, parent, byok)
		})
	}

	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceSubscription, datatypes.JSONSlice[string]{"native-model"}, boolPointer(true))
	require.ErrorIs(t, err, models.ErrInheritedModelSelection)
	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderAnthropic, models.UsageFundingSourceSubscription, nil, boolPointer(false))
	require.Error(t, err)
}

func Test__CompactModelIDs(t *testing.T) {
	assert.Equal(t, []string{"gpt-4o", "gpt-4.1"}, models.CompactModelIDs([]string{" gpt-4o ", "", "gpt-4o", "gpt-4.1"}))
}

func Test__IntersectModelIDs(t *testing.T) {
	assert.Equal(t, []string{"b"}, models.IntersectModelIDs([]string{"a", "b", "c"}, []string{"gone", "b", "b", ""}))
}

func Test__BYOKIntegrationAppName(t *testing.T) {
	name, err := models.BYOKIntegrationAppName(models.UsageProviderAnthropic)
	require.NoError(t, err)
	assert.Equal(t, "claude", name)

	name, err = models.BYOKIntegrationAppName(models.UsageProviderCustom)
	require.NoError(t, err)
	assert.Equal(t, models.CustomLLMAppName, name)
}

func TestLegacyOrganizationSelectionsDoNotRestrictWorkspaceModels(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	// Old rows can remain visible during a coordinated upgrade. They are neither
	// provider discovery data nor workspace execution policy.
	for _, table := range []string{"organization_byok_model_allowlists", "organization_hosted_model_allowlists"} {
		require.NoError(t, tx.Exec("CREATE TEMP TABLE "+table+" (organization_id uuid, provider text, allowed_models jsonb) ON COMMIT DROP").Error)
	}
	_, err = models.UpsertHostedLLMProvider(tx, models.HostedLLMProvider{Provider: models.UsageProviderAnthropic, Enabled: true, APIKey: []byte("key"), AllowedModels: []string{"claude-sonnet-4-6", "claude-opus-4-6"}})
	require.NoError(t, err)
	_, err = support.SaveProviderModelCatalog(tx, r.Organization.ID, models.UsageProviderAnthropic, []string{"claude-sonnet-4-6", "claude-opus-4-6"})
	require.NoError(t, err)
	for _, oldSelection := range []string{`[]`, `["claude-sonnet-4-6"]`} {
		for _, table := range []string{"organization_byok_model_allowlists", "organization_hosted_model_allowlists"} {
			require.NoError(t, tx.Exec("DELETE FROM "+table).Error)
			require.NoError(t, tx.Exec("INSERT INTO "+table+" VALUES (?, ?, ?::jsonb)", r.Organization.ID, models.UsageProviderAnthropic, oldSelection).Error)
		}
		for _, funding := range []string{models.UsageFundingSourceHosted, models.UsageFundingSourceBYOK} {
			catalog, err := models.ResolveSelectableLLMModels(tx, r.Organization.ID, &factory.ID, models.UsageProviderAnthropic, funding)
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{"claude-sonnet-4-6", "claude-opus-4-6"}, catalog)
		}
		model, err := models.ResolveBYOKRunnerModel(tx, r.Organization.ID, &factory.ID, "runnerClaudeCode", map[string]any{"model": "claude-opus-4-6", "credentials": map[string]any{"source": "integration"}})
		require.NoError(t, err)
		assert.Equal(t, "claude-opus-4-6", model)
	}
}
