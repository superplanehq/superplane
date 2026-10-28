package factories

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// A retry can save a new credential choice after the setup apps already exist.
// Repair subscription transitions from the actual live nodes, not the saved
// choice, which the UI updates before it requests completion.
func synchronizeOnboardingSubscriptionApps(ctx context.Context, tx *gorm.DB, deps IntakeDependencies, factory *models.Factory, config models.FactoryOnboardingConfig) error {
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return err
	}
	var rewrite *workspaceAgentRewrite
	for i := range canvases {
		canvas := &canvases[i]
		if canvas.LiveVersionID == nil {
			continue
		}
		live, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
		if err != nil {
			return err
		}
		needsRewrite := false
		for _, node := range live.Nodes {
			if !isWorkspaceAgentComponent(node.ComponentName()) {
				continue
			}
			credentials, _ := node.Configuration["credentials"].(map[string]any)
			subscription := node.ComponentName() == "runnerCodex" && credentials["source"] == "subscription"
			if subscription != config.CodexSubscription {
				needsRewrite = true
				break
			}
		}
		if !needsRewrite {
			continue
		}
		if rewrite == nil {
			resolved, err := onboardingAgentRewrite(tx, deps, factory, config)
			if err != nil {
				return err
			}
			rewrite = &resolved
		}
		if _, err := rewriteCanvasAgents(tx, canvas, versionOwnerID(ctx), *rewrite); err != nil {
			return err
		}
	}
	return nil
}

func onboardingAgentRewrite(tx *gorm.DB, deps IntakeDependencies, factory *models.Factory, config models.FactoryOnboardingConfig) (workspaceAgentRewrite, error) {
	if config.CodexSubscription {
		return workspaceAgentRewrite{Component: "runnerCodex", CredentialsSource: "subscription"}, nil
	}
	if config.AgentIntegrationID == "" {
		return workspaceAgentRewriteFor(modelSourceHosted, "", nil)
	}
	integration, err := findReadyOnboardingIntegration(tx, factory.OrganizationID, config.AgentIntegrationID)
	if err != nil {
		return workspaceAgentRewrite{}, err
	}
	provider := map[string]string{
		"claude": modelSourceAnthropic, "openai": modelSourceOpenAI,
		"openrouter": modelSourceOpenRouter, models.CustomLLMAppName: modelSourceCustom,
	}[integration.AppName]
	var modelIDs []string
	if provider == modelSourceCustom {
		modelIDs, err = organizations.ListConnectedBYOKModelIDs(tx, deps.Registry, integration)
		if err != nil {
			return workspaceAgentRewrite{}, err
		}
	}
	return workspaceAgentRewriteFor(provider, integration.InstallationName, modelIDs)
}
