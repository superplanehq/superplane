package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListLineRunnerModels__ClaudeLineUsesAnthropicAllowlist(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6", "claude-opus-4-6")
	seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-opus-4-6", "claude-sonnet-4-6"}, ids)
	assert.NotContains(t, ids, "gpt-5")
	assert.True(t, usesHostedFunding)
}

func Test__ListLineRunnerModels__CodexLineOmitsClaudeAliases(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "sonnet", "opus")
	seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5", "gpt-5-mini")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerCodex, "hosted", "gpt-5")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-5", "gpt-5-mini"}, ids)
	assert.NotContains(t, ids, "sonnet")
	assert.NotContains(t, ids, "opus")
	assert.True(t, usesHostedFunding)
}

func Test__ListLineRunnerModels__OpenRouterLineUsesOpenRouterIds(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderOpenRouter, "anthropic/claude-sonnet-4-6", "openai/gpt-5")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerOpenRouter, "hosted", "anthropic/claude-sonnet-4-6")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"anthropic/claude-sonnet-4-6", "openai/gpt-5"}, ids)
	assert.True(t, usesHostedFunding)
}

func Test__ListLineRunnerModels__CustomProviderUsesItsAllowlistOnOpenCode(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenRouter, datatypes.JSONSlice[string]{"openrouter-only"})
	require.NoError(t, err)
	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderCustom, datatypes.JSONSlice[string]{"zeta-model", "alpha-model"})
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOKCustomProvider))

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerOpenRouter, "integration", "zeta-model")
	live, err := models.FindLiveCanvasVersionInTransaction(db, app.ID)
	require.NoError(t, err)
	for i := range live.Nodes {
		if live.Nodes[i].ID != "agent" {
			continue
		}
		live.Nodes[i].Configuration["llmProvider"] = models.UsageProviderCustom
	}
	require.NoError(t, db.Save(live).Error)
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha-model", "zeta-model"}, ids)
	assert.NotContains(t, ids, "openrouter-only")
	assert.False(t, usesHostedFunding)
}

func Test__ListLineRunnerModels__MixedLineUnionsProviders(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6")
	seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5")

	claudeApp := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	codexApp := createLineAppWithRunner(t, r, factoryModel.ID, runnerCodex, "hosted", "gpt-5")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: claudeApp.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: codexApp.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-sonnet-4-6", "gpt-5"}, ids)
	assert.True(t, usesHostedFunding)
}

func Test__ListLineRunnerModels__MixedFundingLineUsesHostedFunding(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6")
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))

	hostedApp := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	byokApp := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "integration", "claude-opus-4-6")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: hostedApp.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: byokApp.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	_, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.True(t, usesHostedFunding)
}

func Test__ListLineRunnerModels__BYOKLineSkipsHostedFunding(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "integration", "claude-sonnet-4-6")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	_, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.False(t, usesHostedFunding)
}

func Test__ListLineRunnerModels__StoredBYOKModelsRespectAllowlists(t *testing.T) {
	for _, tc := range []struct {
		name               string
		organizationModels []string
		factoryModels      []string
		want               []string
	}{
		{name: "missing organization allowlist"},
		{name: "empty organization allowlist", organizationModels: []string{}},
		{
			name:               "organization excludes stored model",
			organizationModels: []string{"claude-sonnet-4-6"},
			want:               []string{"claude-sonnet-4-6"},
		},
		{
			name:               "factory excludes stored model",
			organizationModels: []string{"claude-opus-4-6", "claude-sonnet-4-6"},
			factoryModels:      []string{"claude-sonnet-4-6"},
			want:               []string{"claude-sonnet-4-6"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
			factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			if tc.organizationModels != nil {
				_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, tc.organizationModels)
				require.NoError(t, err)
			}
			if tc.factoryModels != nil {
				_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factoryModel.ID, models.UsageProviderAnthropic, models.UsageFundingSourceBYOK, tc.factoryModels)
				require.NoError(t, err)
			}

			planning := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "integration", "claude-opus-4-6")
			implement := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "integration", "claude-sonnet-4-6")
			_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
				{Type: models.FactoryLineStepTypeRunApp, AppID: planning.ID, Entrypoint: "start"},
				{Type: models.FactoryLineStepTypeRunApp, AppID: implement.ID, Entrypoint: "start"},
			})
			require.NoError(t, err)

			ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
			require.NoError(t, err)
			assert.Equal(t, tc.want, ids)
			assert.False(t, usesHostedFunding)
		})
	}
}

func Test__ListLineRunnerModels__DropsStoredBYOKModelWithoutAccess(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	upsertHostedOnboardingProvider(t, db)
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderAnthropic, datatypes.JSONSlice[string]{"opus"})
	require.NoError(t, err)

	planning := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "integration", "opus")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: planning.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Empty(t, ids)
	assert.False(t, usesHostedFunding)
}

func Test__ListLineRunnerModels__SuperPlaneLineUsesHostedSelectableKeys(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6")
	seedHostedModels(t, db, models.UsageProviderOpenRouter, "x-ai/grok-4.6")

	app := createLineAppWithRunner(t, r, factoryModel.ID, models.SuperPlaneRunnerComponent, "hosted", "")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{
		models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderAnthropic, "claude-sonnet-4-6"),
		models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderOpenRouter, "x-ai/grok-4.6"),
	}, ids)
	assert.True(t, usesHostedFunding)
}

func Test__SerializeLineRunnerModels__NamesHostedSelectableKeys(t *testing.T) {
	modelsProto := serializeLineRunnerModels([]string{
		models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderOpenRouter, "x-ai/grok-4.6"),
		"opus",
	})
	require.Len(t, modelsProto, 2)
	assert.Equal(t, "hosted::openrouter::x-ai/grok-4.6", modelsProto[0].GetId())
	assert.Equal(t, "x-ai/grok-4.6", modelsProto[0].GetName())
	assert.Equal(t, "claude-opus-5-5", modelsProto[1].GetId())
	assert.Equal(t, "claude-opus-5-5", modelsProto[1].GetName())
}

func TestLineRunnerModelsKeepSubscriptionAndAPIKeyCatalogsSeparate(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, []string{"gpt-4o"})
	require.NoError(t, err)
	connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, factory.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{Model: "gpt-5.4"}}, time.Now()))
	require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
	for _, source := range []string{"subscription", "integration", "secret"} {
		app := createLineAppWithRunner(t, r, factory.ID, runnerCodex, source, "")
		_, err = factory.CreateLine(db, source, []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"}})
		require.NoError(t, err)
		ids, hosted, err := listLineRunnerModels(db, r.Organization.ID, factory.ID, source, r.User)
		require.NoError(t, err)
		assert.False(t, hosted)
		if source == "subscription" {
			assert.Equal(t, []string{"gpt-5.4"}, ids)
			continue
		}
		assert.Equal(t, []string{"gpt-4o"}, ids)
	}
}

func Test__ListLineRunnerModels__NoRunnerNodesReturnsEmpty(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6")

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "plain", "start")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	ids, usesHostedFunding, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Empty(t, ids)
	assert.False(t, usesHostedFunding)
}

func seedHostedModels(t *testing.T, db *gorm.DB, provider string, modelsIDs ...string) {
	t.Helper()
	var previous models.HostedLLMProvider
	hadPrevious := db.Where("provider = ?", provider).First(&previous).Error == nil
	t.Cleanup(func() {
		if hadPrevious {
			_, _ = models.UpsertHostedLLMProvider(database.Conn(), previous)
			return
		}
		_ = database.Conn().Where("provider = ?", provider).Delete(&models.HostedLLMProvider{})
	})
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      provider,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string](modelsIDs),
	})
	require.NoError(t, err)
}

func createLineAppWithRunner(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
	component string,
	credentialSource string,
	model string,
) *models.Canvas {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           support.RandomName("app"),
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     "start",
			Name:       "start",
			Type:       models.NodeTypeTrigger,
			State:      models.CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "onRun"},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     "agent",
			Name:       "agent",
			Type:       models.NodeTypeComponent,
			State:      models.CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: component},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &r.User,
			Nodes: datatypes.NewJSONSlice([]models.Node{
				{
					ID:   "start",
					Name: "start",
					Type: models.NodeTypeTrigger,
					Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}},
				},
				{
					ID:   "agent",
					Name: "agent",
					Type: models.NodeTypeComponent,
					Ref:  models.NodeRef{Component: &models.ComponentRef{Name: component}},
					Configuration: map[string]any{
						"model": model,
						"credentials": map[string]any{
							"source": credentialSource,
						},
					},
				},
			}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}).Error
	}))
	return canvas
}
