package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	_ "github.com/superplanehq/superplane/pkg/integrations/claude"
	_ "github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func TestListIntegrationsScopesWorkspaceFailures(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	ctx := context.Background()
	factory, err := models.CreateFactory(db, r.Organization.ID, "Current", "", "CUR")
	require.NoError(t, err)
	other, err := models.CreateFactory(db, r.Organization.ID, "Other", "", "OTH")
	require.NoError(t, err)
	empty, err := models.CreateFactory(db, r.Organization.ID, "Empty", "", "EMP")
	require.NoError(t, err)

	connection := func(provider, name string) *models.Integration {
		integration := models.NewIntegration(uuid.New(), r.Organization.ID, provider, name, map[string]any{})
		integration.State = models.IntegrationStateError
		require.NoError(t, db.Create(integration).Error)
		return integration
	}
	github := connection("github", "Current GitHub")
	agent := connection("github", "Configured agent")
	jira := connection("jira", "Intake Jira")
	unusedGitHub := connection("github", "Old GitHub")
	unusedJira := connection("jira", "Old Jira")
	deleted := connection("github", "Deleted app")
	appAgent := connection("claude", "App agent")
	appEnvironment := connection("sentry", "App environment")
	config := models.FactoryOnboardingConfig{VCSIntegrationID: github.ID.String(), AgentIntegrationID: agent.ID.String(), AppRepository: "acme/api"}
	require.NoError(t, db.Model(factory).Update("onboarding_config", datatypes.NewJSONType(config)).Error)
	require.NoError(t, db.Model(other).Update("onboarding_config", datatypes.NewJSONType(config)).Error)

	hook := func(integration *models.Integration, configuration map[string]any) *models.Webhook {
		webhook := &models.Webhook{
			AppInstallationID: &integration.ID, State: models.WebhookStateFailed, Secret: []byte("test-secret"),
			Configuration: datatypes.NewJSONType[any](configuration),
			Metadata:      datatypes.NewJSONType[any](map[string]any{}),
		}
		require.NoError(t, db.Create(webhook).Error)
		return webhook
	}
	shared := hook(github, map[string]any{"repository": "acme/api", "eventType": "issues"})
	monitor := hook(github, map[string]any{"repository": "acme/api", "factoryMergeability": true})
	retainedMonitor := hook(github, map[string]any{"repository": "acme/retained", "factoryMergeability": true})
	contextMonitor := hook(github, map[string]any{"repository": "ACME/context", "factoryMergeability": true})
	jiraHook := hook(jira, map[string]any{})
	// Neither provider nor repository overlap proves that a registration is used.
	unrelated := hook(github, map[string]any{"repository": "acme/api", "eventType": "issues"})
	oldMonitor := hook(github, map[string]any{"repository": "acme/old", "factoryMergeability": true})
	oldHook := hook(unusedGitHub, map[string]any{"repository": "{{ root().data.repository.name }}"})
	wrongConnection := hook(unusedGitHub, map[string]any{"repository": "acme/api", "factoryMergeability": true})
	unusedJiraHook := hook(unusedJira, map[string]any{})

	bind := func(owner *models.Factory, integration *models.Integration, webhook *models.Webhook) *models.Canvas {
		canvas := support.CreateFactoryCanvas(t, r, owner.ID, "app")
		node := &models.CanvasNode{WorkflowID: canvas.ID, NodeID: "trigger", AppInstallationID: &integration.ID, State: models.CanvasNodeStateError}
		if webhook != nil {
			node.WebhookID = &webhook.ID
		}
		require.NoError(t, db.Create(node).Error)
		return canvas
	}
	bind(factory, github, shared)
	bind(other, github, shared)
	bind(factory, jira, jiraHook)
	app := bind(factory, github, nil)
	require.NoError(t, db.Model(&models.CanvasNode{}).Where("workflow_id = ?", app.ID).Updates(map[string]any{
		"type": models.NodeTypeComponent,
		"ref":  datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}}),
		"configuration": datatypes.NewJSONType(map[string]any{
			"credentials": map[string]any{"source": "integration", "integration": map[string]any{"name": appAgent.InstallationName}},
			"environmentFrom": []any{
				map[string]any{"source": "integration", "integration": map[string]any{"name": appEnvironment.InstallationName}},
				map[string]any{"source": "secret", "integration": map[string]any{"name": unusedGitHub.InstallationName}},
			},
			"prompt": map[string]any{"integration": map[string]any{"name": unusedJira.InstallationName}},
		}),
	}).Error)
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/context", Access: models.FactoryContextRepoAccessWrite,
	})
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Existing task", "", nil, nil, nil)
	require.NoError(t, err)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderGitHub, Repository: "acme/retained", Number: 1,
		URL: "https://github.com/acme/retained/pull/1", State: models.FactoryPullRequestStateDraft,
	})
	require.NoError(t, err)
	bind(other, unusedJira, unusedJiraHook)
	deletedCanvas := bind(factory, deleted, nil)
	require.NoError(t, db.Delete(deletedCanvas).Error)
	deletedNodeCanvas := bind(factory, unusedGitHub, oldHook)
	require.NoError(t, db.Where("workflow_id = ?", deletedNodeCanvas.ID).Delete(&models.CanvasNode{}).Error)

	for _, tc := range []struct {
		name         string
		ref          string
		integrations []uuid.UUID
		webhooks     []uuid.UUID
	}{
		{"current workspace", factory.ID.String(), []uuid.UUID{github.ID, agent.ID, jira.ID, appAgent.ID, appEnvironment.ID}, []uuid.UUID{shared.ID, monitor.ID, contextMonitor.ID, retainedMonitor.ID, jiraHook.ID}},
		{"shared workspace", other.RouteSegment(), []uuid.UUID{github.ID, agent.ID, unusedJira.ID}, []uuid.UUID{shared.ID, monitor.ID, unusedJiraHook.ID}},
		{"empty workspace", empty.ID.String(), nil, nil},
		{"organization settings", "", []uuid.UUID{github.ID, agent.ID, jira.ID, unusedGitHub.ID, unusedJira.ID, deleted.ID, appAgent.ID, appEnvironment.ID}, []uuid.UUID{shared.ID, monitor.ID, contextMonitor.ID, retainedMonitor.ID, jiraHook.ID, unrelated.ID, oldMonitor.ID, oldHook.ID, wrongConnection.ID, unusedJiraHook.ID}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := ListIntegrations(ctx, r.Registry, r.Organization.ID.String(), tc.ref)
			require.NoError(t, err)
			var integrationIDs, webhookIDs []uuid.UUID
			for _, integration := range response.Integrations {
				integrationIDs = append(integrationIDs, uuid.MustParse(integration.Metadata.Id))
				for _, failure := range integration.Status.WebhookFailures {
					webhookIDs = append(webhookIDs, uuid.MustParse(failure.Id))
				}
			}
			require.ElementsMatch(t, tc.integrations, integrationIDs)
			require.ElementsMatch(t, tc.webhooks, webhookIDs)
		})
	}

	foreign, err := models.CreateOrganization("foreign-"+uuid.NewString(), "")
	require.NoError(t, err)
	_, err = ListIntegrations(ctx, r.Registry, foreign.ID.String(), factory.ID.String())
	code, _, _ := grpcerrors.HandlerStatus(err)
	require.Equal(t, codes.NotFound, code)
}
