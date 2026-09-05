package models_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__RewriteHostedProviderRunnerToSuperPlane(t *testing.T) {
	t.Parallel()

	t.Run("rewrites hosted Claude nodes", func(t *testing.T) {
		node := models.Node{
			ID:  "agent-1",
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
			Configuration: map[string]any{
				"machineType": "e1-large-amd64",
				"model":       "sonnet",
				"credentials": map[string]any{"source": "hosted"},
				"steps":       []any{map[string]any{"name": "Prompt", "type": "prompt"}},
			},
		}

		assert.True(t, models.RewriteHostedProviderRunnerToSuperPlane(&node))
		assert.Equal(t, models.SuperPlaneRunnerComponent, node.ComponentName())
		assert.NotContains(t, node.Configuration, "credentials")
		assert.Equal(t, "anthropic::sonnet", node.Configuration["model"])
		assert.Equal(t, "e1-large-amd64", node.Configuration["machineType"])
	})

	t.Run("rewrites hosted OpenRouter nodes and drops maxTurns", func(t *testing.T) {
		node := models.Node{
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerOpenRouter"}},
			Configuration: map[string]any{
				"model":       "anthropic/claude-sonnet-4-6",
				"maxTurns":    64,
				"credentials": map[string]any{"source": "hosted"},
			},
		}

		assert.True(t, models.RewriteHostedProviderRunnerToSuperPlane(&node))
		assert.Equal(t, models.SuperPlaneRunnerComponent, node.ComponentName())
		assert.NotContains(t, node.Configuration, "maxTurns")
		assert.Equal(t, "openrouter::anthropic/claude-sonnet-4-6", node.Configuration["model"])
	})

	t.Run("leaves integration nodes unchanged", func(t *testing.T) {
		node := models.Node{
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerCodex"}},
			Configuration: map[string]any{
				"model": "gpt-5",
				"credentials": map[string]any{
					"source":      "integration",
					"integration": map[string]any{"name": "openai"},
				},
			},
		}

		assert.False(t, models.RewriteHostedProviderRunnerToSuperPlane(&node))
		assert.Equal(t, "runnerCodex", node.ComponentName())
		assert.Equal(t, "gpt-5", node.Configuration["model"])
	})

	t.Run("rewrites a slice of hosted nodes", func(t *testing.T) {
		nodes := []models.Node{
			{
				Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
				Configuration: map[string]any{
					"credentials": map[string]any{"source": "hosted"},
				},
			},
			{
				Ref: models.NodeRef{Component: &models.ComponentRef{Name: "runnerCodex"}},
				Configuration: map[string]any{
					"credentials": map[string]any{
						"source":      "integration",
						"integration": map[string]any{"name": "openai"},
					},
				},
			},
		}
		models.RewriteHostedProviderRunnerNodes(nodes)
		assert.Equal(t, models.SuperPlaneRunnerComponent, nodes[0].ComponentName())
		assert.Equal(t, "runnerCodex", nodes[1].ComponentName())
	})

	t.Run("ignores a nil node", func(t *testing.T) {
		assert.False(t, models.RewriteHostedProviderRunnerToSuperPlane(nil))
	})
}

func Test__NormalizeDefaultHostedLLMModel(t *testing.T) {
	t.Parallel()

	empty, err := models.NormalizeDefaultHostedLLMModel("", "")
	require.NoError(t, err)
	assert.False(t, empty.IsSet())

	_, err = models.NormalizeDefaultHostedLLMModel("anthropic", "")
	assert.ErrorIs(t, err, models.ErrDefaultHostedModelIncomplete)

	got, err := models.NormalizeDefaultHostedLLMModel(" Anthropic ", " claude-sonnet-4-6 ")
	require.NoError(t, err)
	assert.Equal(t, models.DefaultHostedLLMModel{Provider: "anthropic", Model: "claude-sonnet-4-6"}, got)
}

func Test__ParseHostedLLMModelKey(t *testing.T) {
	t.Parallel()

	got, err := models.ParseHostedLLMModelKey("openrouter::anthropic/claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Equal(t, models.DefaultHostedLLMModel{Provider: "openrouter", Model: "anthropic/claude-sonnet-4-6"}, got)

	got, err = models.ParseHostedLLMModelKey("hosted::anthropic::claude-sonnet-4-6")
	require.NoError(t, err)
	assert.Equal(t, models.DefaultHostedLLMModel{Provider: "anthropic", Model: "claude-sonnet-4-6"}, got)

	_, err = models.ParseHostedLLMModelKey("claude-sonnet-4-6")
	assert.ErrorIs(t, err, models.ErrDefaultHostedModelIncomplete)

	assert.Equal(t, "anthropic::claude-sonnet-4-6", models.FormatHostedLLMModelKey("anthropic", "claude-sonnet-4-6"))
}

func Test__InstallationDefaultHostedLLMModel(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	_ = r

	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	t.Run("rejects a default that is not on an offered allowlist", func(t *testing.T) {
		_, err := models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
			WelcomeGrantCents:     models.DefaultWelcomeGrantCents,
			MarkupBPS:             models.DefaultMarkupBPS,
			WarningThresholdBPS:   models.DefaultWarningThresholdBPS,
			DefaultHostedProvider: stringPtr("anthropic"),
			DefaultHostedModel:    stringPtr("claude-sonnet-4-6"),
		})
		assert.ErrorIs(t, err, models.ErrDefaultHostedModelNotOnAllowlist)
	})

	t.Run("saves a default that is on an offered allowlist", func(t *testing.T) {
		_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
			Provider:      models.UsageProviderAnthropic,
			Enabled:       true,
			APIKey:        []byte("test-hosted-key"),
			AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
		})
		require.NoError(t, err)

		saved, err := models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
			WelcomeGrantCents:     models.DefaultWelcomeGrantCents,
			MarkupBPS:             models.DefaultMarkupBPS,
			WarningThresholdBPS:   models.DefaultWarningThresholdBPS,
			DefaultHostedProvider: stringPtr("anthropic"),
			DefaultHostedModel:    stringPtr("claude-sonnet-4-6"),
		})
		require.NoError(t, err)
		assert.Equal(t, "anthropic", stringValue(saved.DefaultHostedProvider))
		assert.Equal(t, "claude-sonnet-4-6", stringValue(saved.DefaultHostedModel))
	})

	t.Run("rejects removing the default from an allowlist when other models remain", func(t *testing.T) {
		_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
			Provider:      models.UsageProviderAnthropic,
			Enabled:       true,
			APIKey:        []byte("test-hosted-key"),
			AllowedModels: datatypes.JSONSlice[string]{"claude-opus-4-6"},
		})
		require.NoError(t, err)
		err = models.SyncDefaultHostedLLMModelAfterProviderChange(db)
		assert.ErrorIs(t, err, models.ErrDefaultHostedModelMustBeReplaced)
	})
}

func Test__SuperPlaneRunnerReadinessMessage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, models.SuperPlaneRunnerNoCreditMessage, models.SuperPlaneRunnerReadinessMessage(models.ErrSuperPlaneRunnerNoCredit))
	assert.Equal(t, models.SuperPlaneRunnerNoFactoryBudgetMessage, models.SuperPlaneRunnerReadinessMessage(models.ErrSuperPlaneRunnerNoFactoryBudget))
	assert.Equal(t, models.SuperPlaneRunnerNoFactoryBudgetMessage, models.SuperPlaneRunnerReadinessMessage(models.ErrFactoryHostedBudgetEmpty))
	assert.Equal(t, models.SuperPlaneRunnerNoModelMessage, models.SuperPlaneRunnerReadinessMessage(models.ErrSuperPlaneRunnerNoModel))
	assert.Equal(t, models.SuperPlaneRunnerModelNotAllowedMessage, models.SuperPlaneRunnerReadinessMessage(models.ErrSuperPlaneRunnerModelNotAllowed))
	assert.Equal(t, "organization is required for hosted LLM credit", models.SuperPlaneRunnerReadinessMessage(fmt.Errorf("organization is required for hosted LLM credit")))
}

func Test__SuperPlaneRunnerReadinessError(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	err := models.SuperPlaneRunnerReadinessError(db, r.Organization.ID, nil)
	assert.ErrorIs(t, err, models.ErrSuperPlaneRunnerNoModel)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("test-hosted-key"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)
	_, err = models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
		WelcomeGrantCents:     models.DefaultWelcomeGrantCents,
		MarkupBPS:             models.DefaultMarkupBPS,
		WarningThresholdBPS:   models.DefaultWarningThresholdBPS,
		DefaultHostedProvider: stringPtr("anthropic"),
		DefaultHostedModel:    stringPtr("claude-sonnet-4-6"),
	})
	require.NoError(t, err)

	require.NoError(t, models.SuperPlaneRunnerReadinessError(db, r.Organization.ID, nil))
}

func Test__AnnotateSuperPlaneRunnerNodes(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("stamps a missing instance model", func(t *testing.T) {
		nodes := []models.Node{{
			ID:  "agent-1",
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: models.SuperPlaneRunnerComponent}},
		}}
		require.NoError(t, models.AnnotateSuperPlaneRunnerNodes(db, r.Organization.ID, nil, nodes))
		require.NotNil(t, nodes[0].ErrorMessage)
		assert.Equal(t, models.SuperPlaneRunnerNoModelMessage, *nodes[0].ErrorMessage)
	})

	t.Run("returns unexpected errors instead of painting them on the node", func(t *testing.T) {
		nodes := []models.Node{{
			Ref: models.NodeRef{Component: &models.ComponentRef{Name: models.SuperPlaneRunnerComponent}},
		}}
		err := models.AnnotateSuperPlaneRunnerNodes(db, uuid.Nil, nil, nodes)
		require.Error(t, err)
		assert.Nil(t, nodes[0].ErrorMessage)
	})
}

func stringPtr(value string) *string {
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
