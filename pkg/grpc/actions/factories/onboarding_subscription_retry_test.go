package factories

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func TestOnboardingRetryChangesSubscriptionCredentials(t *testing.T) {
	for _, target := range []string{"hosted", "openai"} {
		t.Run(target, func(t *testing.T) {
			r := support.Setup(t)
			support.AssignOpenSystemTrial(t, r.Organization.ID)
			db := database.DB(t.Context())
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
			appID, lineID := createOnboardingResources(t, r, factory)
			component, source := "runnerCodex", "subscription"
			app := createLineAppWithRunner(t, r, factory.ID, component, source, "old-model")
			liveID := *app.LiveVersionID
			subscription := false
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
				ID: uuid.New(), OrganizationID: r.Organization.ID,
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
			assert.Equal(t, "integration", credentials["source"])
			integration, err := models.FindIntegrationInTransaction(db, r.Organization.ID, uuid.MustParse(agentID))
			require.NoError(t, err)
			assert.Equal(t, integration.InstallationName, credentials["integration"].(map[string]any)["name"])
			assert.NotEmpty(t, agent.Configuration["model"])
		})
	}
}

func TestHostedOnboardingClearsOldSubscriptionChoiceAndRejectsNewChoice(t *testing.T) {
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.DB(t.Context())
	upsertHostedOnboardingProvider(t, db)
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	subscription := true
	harness := models.FactoryOnboardingAgentHarnessCodex
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription, AgentHarness: &harness}))
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
	require.NoError(t, err)
	_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String(), CodexSubscription: &subscription})
	require.Error(t, err)
	response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String()})
	require.NoError(t, err)
	assert.False(t, response.Factory.Onboarding.CodexSubscription)
	assert.Equal(t, pb.FactoryOnboarding_AGENT_HARNESS_UNSPECIFIED, response.Factory.Onboarding.AgentHarness)
	kept, err := models.FindCodexConnection(db, r.Organization.ID, r.User)
	require.NoError(t, err)
	assert.Equal(t, connection.ID, kept.ID)
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription, AgentHarness: &harness}))
	require.NoError(t, db.Model(factory).Update("onboarding_completed_at", time.Now()).Error)
	response, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String()})
	require.NoError(t, err)
	assert.True(t, response.Factory.Onboarding.CodexSubscription)
	assert.Equal(t, pb.FactoryOnboarding_AGENT_HARNESS_CODEX, response.Factory.Onboarding.AgentHarness)
}

func TestOnboardingSubscriptionPersistsOwnerBeforeIntakeAndCompletion(t *testing.T) {
	for _, scenario := range []string{"new intake", "reused intake"} {
		t.Run(scenario, func(t *testing.T) {
			reuse := scenario == "reused intake"
			r := support.Setup(t)
			support.AssignOpenSystemTrial(t, r.Organization.ID)
			db := database.DB(t.Context())
			unsetOnboardingHostedDefault(t, db)
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
			appID, lineID := createOnboardingResources(t, r, factory)
			app := createLineAppWithRunner(t, r, factory.ID, "runnerClaudeCode", "integration", "old-model")
			connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
			require.NoError(t, err)
			require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
			require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{Model: "subscription-model"}}, time.Now()))
			subscription := true
			req := readyOnboardingRequest(factory.ID.String(), vcsID, "", appID, lineID)
			harness := pb.FactoryOnboarding_AGENT_HARNESS_CODEX
			req.AgentHarness = &harness
			req.CodexSubscription = &subscription
			var existingIntakeID string
			if reuse {
				require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
				agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "openai")
				apiKeyRequest := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
				apiKeyRequest.AgentHarness = &harness
				_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), apiKeyRequest)
				require.NoError(t, err)
				intake, err := CreateFactoryIntake(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.CreateFactoryIntakeRequest{FactoryId: factory.ID.String(), Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES, SkipInitialImport: true})
				require.NoError(t, err)
				require.Empty(t, intake.Intake.GetSettings().GetDefaultOwnerId())
				existingIntakeID = intake.Intake.GetId()
			}
			type connectionReadsKey struct{}
			const callback = "test:onboarding_connection_reads"
			require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
				if count, ok := tx.Statement.Context.Value(connectionReadsKey{}).(*atomic.Int64); ok && tx.Statement.Table == "factory_codex_connections" {
					count.Add(1)
				}
			}))
			t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callback)) })
			var connectionReads atomic.Int64
			ctx := context.WithValue(onboardingContext(r), connectionReadsKey{}, &connectionReads)
			response, err := UpdateFactoryOnboarding(ctx, onboardingDeps(r), r.Organization.ID.String(), req)
			require.NoError(t, err)
			assert.EqualValues(t, 1, connectionReads.Load(), "subscription validation must reuse the loaded connection")
			assert.True(t, response.Factory.Onboarding.CodexSubscription)
			assert.Equal(t, r.User.String(), response.Factory.Onboarding.CodexDefaultOwnerId)
			// Provisioning saves the branch and app references before it completes setup.
			branch := "main"
			factory, err = models.FindFactory(db, r.Organization.ID, factory.ID)
			require.NoError(t, err)
			require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{DefaultBranch: &branch}))
			response, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String(), DefaultBranch: &branch})
			require.NoError(t, err)
			assert.True(t, response.Factory.Onboarding.CodexSubscription)
			assert.Equal(t, r.User.String(), response.Factory.Onboarding.CodexDefaultOwnerId)
			if reuse {
				paused := true
				intake, err := UpdateFactoryIntake(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryIntakeRequest{FactoryId: factory.ID.String(), IntakeId: existingIntakeID, Paused: &paused})
				require.NoError(t, err)
				assert.Equal(t, r.User.String(), intake.Intake.GetSettings().GetDefaultOwnerId())
			} else {
				intake, err := CreateFactoryIntake(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.CreateFactoryIntakeRequest{FactoryId: factory.ID.String(), Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES, SkipInitialImport: true})
				require.NoError(t, err)
				assert.Equal(t, r.User.String(), intake.Intake.GetSettings().GetDefaultOwnerId())
			}
			complete := true
			response, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String(), Complete: &complete})
			require.NoError(t, err)
			require.NotNil(t, response.Factory.Onboarding.CompletedAt)
			assert.True(t, response.Factory.Onboarding.CodexSubscription)
			assert.Equal(t, pb.FactoryOnboarding_AGENT_HARNESS_CODEX, response.Factory.Onboarding.AgentHarness)
			assert.Empty(t, response.Factory.Onboarding.AgentIntegrationId)
			live, err := models.FindLiveCanvasVersionInTransaction(db, app.ID)
			require.NoError(t, err)
			agent := findIntakeNode(live.Nodes, "agent")
			require.NotNil(t, agent)
			assert.Equal(t, "runnerCodex", agent.ComponentName())
			assert.Equal(t, map[string]any{"source": "subscription"}, agent.Configuration["credentials"])
			assert.Empty(t, agent.Configuration["model"])
			_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String(), Complete: &complete})
			require.NoError(t, err)
			assertCanvasLiveVersion(t, db, app.ID, live.ID)
		})
	}
}

func unsetOnboardingHostedDefault(t *testing.T, db *gorm.DB) {
	t.Helper()
	settings, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.Background()).Model(settings).Updates(map[string]any{
			"default_hosted_provider": settings.DefaultHostedProvider, "default_hosted_model": settings.DefaultHostedModel,
		}).Error)
	})
	empty := *settings
	empty.DefaultHostedProvider, empty.DefaultHostedModel = nil, nil
	_, err = models.UpdateInstallationLLMSettings(db, empty)
	require.NoError(t, err)
}

func TestOnboardingSubscriptionRequiresEligibleConnection(t *testing.T) {
	for _, state := range []string{"missing", "pending", "disconnected", "reconnect_required", "empty-catalog", "reserved", "revoked", "no-credentials", "not-member"} {
		t.Run(state, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			unsetOnboardingHostedDefault(t, db)
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			if state != "missing" {
				connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
				require.NoError(t, err)
				require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
				if state != "empty-catalog" {
					require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{Model: "subscription-model"}}, time.Now()))
				}
				if state == "reserved" {
					fleet := models.DefaultInstallationRunnerFleets(models.DefaultRunnerVersion)[0]
					require.NoError(t, fleet.Create(db))
					task := &models.RunnerTask{ID: uuid.New(), OrganizationID: r.Organization.ID, FleetID: fleet.ID, Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateRunning, PayloadCiphertext: []byte("payload")}
					require.NoError(t, db.Create(task).Error)
					require.NoError(t, db.Create(&models.FactoryCodexLease{ID: uuid.New(), ConnectionID: connection.ID, TaskID: task.ID, Generation: 1, Purpose: models.CodexLeasePurposeLogin, SecretHash: uuid.NewString()}).Error)
				}
				switch state {
				case "pending", "disconnected", "reconnect_required":
					require.NoError(t, db.Model(connection).Update("status", state).Error)
				case "revoked":
					require.NoError(t, db.Model(connection).Update("revocation_pending_at", time.Now()).Error)
				case "no-credentials":
					require.NoError(t, db.Model(connection).Update("credential_version", 0).Error)
				case "not-member":
					require.NoError(t, db.Model(&models.User{}).Where("id = ?", r.User).Update("type", models.UserTypeAPIKey).Error)
				}
			}
			subscription := true
			_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String(), CodexSubscription: &subscription})
			require.Error(t, err)
			switch state {
			case "reserved":
				assert.Equal(t, codes.Aborted, grpcerrors.Code(err))
			case "not-member":
				assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
			default:
				requireCodexConnectionRequired(t, err)
			}
			saved, err := models.FindFactory(db, r.Organization.ID, factory.ID)
			require.NoError(t, err)
			assert.False(t, saved.OnboardingConfigValue().CodexSubscription)
			assert.Empty(t, saved.OnboardingConfigValue().CodexDefaultOwnerID)
		})
	}
}
