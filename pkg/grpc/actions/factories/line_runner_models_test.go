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

	ids, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-opus-4-6", "claude-sonnet-4-6"}, ids)
	assert.NotContains(t, ids, "gpt-5")
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

	ids, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-5", "gpt-5-mini"}, ids)
	assert.NotContains(t, ids, "sonnet")
	assert.NotContains(t, ids, "opus")
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

	ids, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"anthropic/claude-sonnet-4-6", "openai/gpt-5"}, ids)
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

	ids, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-sonnet-4-6", "gpt-5"}, ids)
}

func Test__ListLineRunnerModels__IncludesCanvasModelsWhenAllowlistEmpty(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	planning := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "integration", "opus")
	implement := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "integration", "sonnet")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: planning.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: implement.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	ids, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Equal(t, []string{"opus", "sonnet"}, ids)
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

	ids, err := listLineRunnerModels(db, r.Organization.ID, factoryModel.ID, "ship")
	require.NoError(t, err)
	assert.Empty(t, ids)
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
