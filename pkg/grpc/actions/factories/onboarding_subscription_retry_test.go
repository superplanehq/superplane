package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func TestOnboardingRetryChangesSubscriptionCredentials(t *testing.T) {
	for _, target := range []string{"subscription", "subscription-from-key", "hosted", "openai"} {
		t.Run(target, func(t *testing.T) {
			r := support.Setup(t)
			support.AssignOpenSystemTrial(t, r.Organization.ID)
			db := database.DB(t.Context())
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
			appID, lineID := createOnboardingResources(t, r, factory)
			component, source := "runnerCodex", "subscription"
			if target == "subscription" {
				component, source = models.SuperPlaneRunnerComponent, "hosted"
			}
			if target == "subscription-from-key" {
				component, source = "runnerClaudeCode", "integration"
			}
			app := createLineAppWithRunner(t, r, factory.ID, component, source, "old-model")
			liveID := *app.LiveVersionID
			subscription := target == "subscription" || target == "subscription-from-key"
			harness := pb.FactoryOnboarding_AGENT_HARNESS_CODEX
			agentID := ""
			if target == "hosted" {
				upsertHostedOnboardingProvider(t, db)
				harness = pb.FactoryOnboarding_AGENT_HARNESS_SUPERPLANE
			}
			if target == "openai" {
				agentID = createReadyOnboardingIntegration(t, r.Organization.ID, "openai")
			}
			req := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
			req.AgentHarness = &harness
			req.CodexSubscription = &subscription
			// Save the new choice separately, as the UI does before retrying completion.
			_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
			require.NoError(t, err)
			now := time.Now()
			require.NoError(t, db.Create(&models.FactoryCodexConnection{
				ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID,
				OwnerUserID: r.User, Status: models.CodexConnectionStatusReady,
				CreatedAt: now, UpdatedAt: now,
			}).Error)
			complete := true
			response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
				Id: factory.ID.String(), Complete: &complete,
			})
			require.NoError(t, err)
			require.NotNil(t, response.Factory.Onboarding.CompletedAt)
			assertCanvasLiveVersionChanged(t, db, app.ID, liveID)
			live, err := models.FindLiveCanvasVersionInTransaction(db, app.ID)
			require.NoError(t, err)
			// Repeated completion must not create another version or rewrite models.
			_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
				Id: factory.ID.String(), Complete: &complete,
			})
			require.NoError(t, err)
			assertCanvasLiveVersion(t, db, app.ID, live.ID)
			agent := findIntakeNode(live.Nodes, "agent")
			require.NotNil(t, agent)
			if target == "hosted" {
				assertHostedAgentNode(t, db, app.ID)
				return
			}
			assert.Equal(t, "runnerCodex", agent.ComponentName())
			credentials := agent.Configuration["credentials"].(map[string]any)
			if subscription {
				assert.Equal(t, map[string]any{"source": "subscription"}, credentials)
				assert.Empty(t, agent.Configuration["model"])
				return
			}
			assert.Equal(t, "integration", credentials["source"])
			integration, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(agentID))
			require.NoError(t, err)
			assert.Equal(t, integration.InstallationName, credentials["integration"].(map[string]any)["name"])
			assert.NotEmpty(t, agent.Configuration["model"])
		})
	}
}
