package factories

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/automations"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestCanvasAutomationRetainsHostedProviderForDuplicateModelID(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := seedHandOffFactory(t, r, db)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &vcsID}))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryAsCode))
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	upsertHostedOnboardingProvider(t, db)
	_, err := models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider: models.UsageProviderOpenAI, Enabled: true, APIKey: []byte("test-hosted-key"),
		AllowedModels: datatypes.JSONSlice[string]{"sonnet"},
	})
	require.NoError(t, err)

	const source = `apiVersion: v2
kind: Automation
name: Inspect task
agent: {model: sonnet}
triggers: [{use: manual}]
steps: [{name: Inspect, prompt: Inspect the changes.}]
`
	ctx := onboardingContext(r)
	backend := CanvasAutomationBackend{Deps: onboardingDeps(r)}
	service := automations.Service{Backend: backend}
	placement := automations.Placement{OrganizationID: r.Organization.ID, FactoryID: factory.ID, Stage: models.CanvasColumnKeyBacklog}
	_, err = service.Create(ctx, placement, r.User, source)
	require.ErrorContains(t, err, "provider::model")
	qualified := strings.Replace(source, "model: sonnet", "model: openai::sonnet", 1)
	created, err := service.Create(ctx, placement, r.User, qualified)
	require.NoError(t, err)
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, source)
	require.NoError(t, err)
	live, err := models.FindLiveCanvasVersionInTransaction(db, created.ID)
	require.NoError(t, err)
	require.Equal(t, "hosted::openai::sonnet", intakeAgentFromCanvasNodes(live.Nodes).Model)
	legacy, err := backend.FindLegacy(ctx, db, r.Organization.ID, created.ID)
	require.NoError(t, err)
	exported, err := backend.ImportLegacy(ctx, db, legacy)
	require.NoError(t, err)
	doc, err := automationspec.Parse(exported)
	require.NoError(t, err)
	require.Equal(t, "openai::sonnet", doc.Agent.Model)

	// A policy change can remove the saved route while another provider still offers the ID.
	inherit := false
	_, err = models.UpsertFactoryLLMModelAllowlist(db, r.Organization.ID, factory.ID, models.UsageProviderOpenAI, models.UsageFundingSourceHosted, nil, &inherit)
	require.NoError(t, err)
	_, err = service.Update(ctx, r.Organization.ID, created.ID, r.User, source)
	require.ErrorContains(t, err, "agent.model")
	other, err := service.Create(ctx, placement, r.User, strings.Replace(source, "Inspect task", "Review task", 1))
	require.NoError(t, err)
	live, err = models.FindLiveCanvasVersionInTransaction(db, other.ID)
	require.NoError(t, err)
	require.Equal(t, "hosted::anthropic::sonnet", intakeAgentFromCanvasNodes(live.Nodes).Model)
}
