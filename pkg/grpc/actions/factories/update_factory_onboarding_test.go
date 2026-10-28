package factories

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	_ "github.com/superplanehq/superplane/pkg/registryimports"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__UpdateFactoryOnboarding(t *testing.T) {
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.DB(t.Context())

	t.Run("new factory is incomplete in create response", func(t *testing.T) {
		response, err := CreateFactory(context.Background(), r.Organization.ID.String(), &pb.CreateFactoryRequest{
			Name: support.RandomName("factory"),
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding)
		assert.Nil(t, response.Factory.Onboarding.CompletedAt)
		assert.Equal(t, pb.FactoryOnboarding_ISSUES_SOURCE_UNSPECIFIED, response.Factory.Onboarding.IssuesSource)
		assert.Equal(t, pb.FactoryOnboarding_AGENT_HARNESS_UNSPECIFIED, response.Factory.Onboarding.AgentHarness)
		assert.Empty(t, response.Factory.Onboarding.VcsProvider)
		assertGitHubVCSCapabilities(t, response.Factory.GetVcsCapabilities())
	})

	t.Run("partial update persists and returns merged config", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		vcsID := uuid.New().String()
		appRepo := "acme/api"
		issuesSource := pb.FactoryOnboarding_ISSUES_SOURCE_VCS

		response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:               factory.ID.String(),
			VcsIntegrationId: &vcsID,
			AppRepository:    &appRepo,
			IssuesSource:     &issuesSource,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding)
		assert.Nil(t, response.Factory.Onboarding.CompletedAt)
		assert.Equal(t, vcsID, response.Factory.Onboarding.VcsIntegrationId)
		assert.Equal(t, "acme/api", response.Factory.Onboarding.AppRepository)
		assert.Equal(t, pb.FactoryOnboarding_ISSUES_SOURCE_VCS, response.Factory.Onboarding.IssuesSource)

		backlogRepo := "acme/backlog"
		agentHarness := pb.FactoryOnboarding_AGENT_HARNESS_CLAUDE_CODE
		response, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:                factory.ID.String(),
			BacklogRepository: &backlogRepo,
			AgentHarness:      &agentHarness,
		})
		require.NoError(t, err)
		assert.Equal(t, vcsID, response.Factory.Onboarding.VcsIntegrationId)
		assert.Equal(t, "acme/api", response.Factory.Onboarding.AppRepository)
		assert.Equal(t, "acme/backlog", response.Factory.Onboarding.BacklogRepository)
		assert.Equal(t, pb.FactoryOnboarding_AGENT_HARNESS_CLAUDE_CODE, response.Factory.Onboarding.AgentHarness)
	})

	t.Run("bitbucket integration stores the provider and keeps automations off", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, models.ProviderBitbucket)

		response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:               factory.ID.String(),
			VcsIntegrationId: &vcsID,
		})
		require.NoError(t, err)
		assert.Equal(t, models.ProviderBitbucket, response.Factory.Onboarding.VcsProvider)
		assertBitbucketVCSCapabilities(t, response.Factory.GetVcsCapabilities())

		reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ProviderBitbucket, reloaded.OnboardingConfigValue().VCSProvider)
		assert.Equal(t, models.ProviderBitbucket, reloaded.OnboardingConfigValue().EffectiveVCSProvider())
		assert.Equal(t, models.ProviderGitHub, models.FactoryOnboardingConfig{}.EffectiveVCSProvider())
	})

	t.Run("complete succeeds when ready and is idempotent", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		appID, lineID := createOnboardingResources(t, r, factory)

		req := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
		complete := true
		req.Complete = &complete

		response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding.CompletedAt)
		firstCompletedAt := response.Factory.Onboarding.CompletedAt.AsTime()

		response, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:       factory.ID.String(),
			Complete: &complete,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding.CompletedAt)
		assert.Equal(t, firstCompletedAt.UnixMicro(), response.Factory.Onboarding.CompletedAt.AsTime().UnixMicro())
	})

	t.Run("complete without agent integration succeeds when hosted credit is empty", func(t *testing.T) {
		emptyOrg := support.CreateOrganization(t, r, r.User)
		support.AssignOpenSystemTrial(t, emptyOrg.ID)
		require.NoError(t, db.Where("organization_id = ?", emptyOrg.ID).Delete(&billingmodels.OrganizationLLMCreditGrant{}).Error)

		factory, err := models.CreateFactory(db, emptyOrg.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, emptyOrg.ID, "github")
		appID, lineID := createOnboardingResources(t, r, factory)

		req := readyOnboardingRequest(factory.ID.String(), vcsID, "", appID, lineID)
		req.AgentIntegrationId = nil
		complete := true
		req.Complete = &complete

		upsertHostedOnboardingProvider(t, db)
		emptyUser := support.CreateUser(t, r, emptyOrg.ID)
		response, err := UpdateFactoryOnboarding(authentication.SetUserIdInMetadata(context.Background(), emptyUser.ID.String()), onboardingDeps(r), emptyOrg.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding.CompletedAt)
		assert.Empty(t, response.Factory.Onboarding.AgentIntegrationId)
	})

	t.Run("complete without agent integration rejects when no hosted provider is offered", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		appID, lineID := createOnboardingResources(t, r, factory)

		req := readyOnboardingRequest(factory.ID.String(), vcsID, "", appID, lineID)
		req.AgentIntegrationId = nil
		complete := true
		req.Complete = &complete

		clearHostedLLMProviders(t, db)
		_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Contains(t, message, "SuperPlane-hosted models")
	})

	t.Run("complete succeeds with an OpenRouter agent integration", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "openrouter")
		appID, lineID := createOnboardingResources(t, r, factory)

		req := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
		complete := true
		req.Complete = &complete

		response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding.CompletedAt)
		assert.Equal(t, agentID, response.Factory.Onboarding.AgentIntegrationId)
	})

	t.Run("complete succeeds with an OpenAI agent integration and Codex harness", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "openai")
		appID, lineID := createOnboardingResources(t, r, factory)

		req := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
		codex := pb.FactoryOnboarding_AGENT_HARNESS_CODEX
		req.AgentHarness = &codex
		complete := true
		req.Complete = &complete

		response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Onboarding.CompletedAt)
		assert.Equal(t, agentID, response.Factory.Onboarding.AgentIntegrationId)
		assert.Equal(t, pb.FactoryOnboarding_AGENT_HARNESS_CODEX, response.Factory.Onboarding.AgentHarness)
	})

	t.Run("complete rejects an integration from another organization", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		otherOrg := support.CreateOrganization(t, r, r.User)
		vcsID := createReadyOnboardingIntegration(t, otherOrg.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		appID, lineID := createOnboardingResources(t, r, factory)
		complete := true

		request := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
		request.Complete = &complete
		_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), request)

		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("complete with missing fields -> invalid argument", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		complete := true
		appRepo := "acme/api"
		_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:            factory.ID.String(),
			AppRepository: &appRepo,
			Complete:      &complete,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("invalid uuid -> invalid argument", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		badID := "not-a-uuid"
		_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:               factory.ID.String(),
			VcsIntegrationId: &badID,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("wrong organization -> not found", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		otherOrg := support.CreateOrganization(t, r, r.User)
		appRepo := "acme/api"
		_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), otherOrg.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:            factory.ID.String(),
			AppRepository: &appRepo,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("not found -> error", func(t *testing.T) {
		appRepo := "acme/api"
		_, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:            "00000000-0000-0000-0000-000000000001",
			AppRepository: &appRepo,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})
}

func TestUpdateFactoryOnboardingCreatesDiscussionHandler(t *testing.T) {
	r := support.Setup(t)
	support.AssignOpenSystemTrial(t, r.Organization.ID)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationBYOK))
	db := database.DB(t.Context())

	for _, provider := range []string{"github", models.ProviderBitbucket} {
		t.Run(provider+" first completion creates one all-bots handler", func(t *testing.T) {
			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, provider)
			agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
			appID, lineID := createOnboardingResources(t, r, factory)
			req := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
			if provider == models.ProviderBitbucket {
				source := pb.FactoryOnboarding_ISSUES_SOURCE_SKIP
				req.IssuesSource = &source
			}
			complete := true
			req.Complete = &complete

			response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
			require.NoError(t, err)
			require.NotNil(t, response.Factory.Onboarding.CompletedAt)
			assertDiscussionHandler(t, db, factory, provider, 1)

			response, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
				Id:       factory.ID.String(),
				Complete: &complete,
			})
			require.NoError(t, err)
			require.NotNil(t, response.Factory.Onboarding.CompletedAt)
			assertDiscussionHandler(t, db, factory, provider, 1)
		})
	}

	t.Run("overlapping completions create one handler", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		appID, lineID := createOnboardingResources(t, r, factory)
		req := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
		complete := true
		req.Complete = &complete

		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(len(errs))
		for i := range errs {
			go func(i int) {
				defer wg.Done()
				_, errs[i] = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
			}(i)
		}
		wg.Wait()
		successes := 0
		for _, err := range errs {
			if err == nil {
				successes++
			}
		}
		require.Greater(t, successes, 0)
		assertDiscussionHandler(t, db, factory, "github", 1)
	})

	t.Run("an already complete workspace is unchanged", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		appID, lineID := createOnboardingResources(t, r, factory)
		appRepo := "acme/api"
		backlogRepo := "acme/backlog"
		issuesSource := models.FactoryOnboardingIssuesSourceSkip
		agentHarness := models.FactoryOnboardingAgentHarnessClaudeCode
		require.NoError(t, factory.CompleteOnboarding(db, models.FactoryOnboardingPatch{
			VCSIntegrationID:   &vcsID,
			AgentIntegrationID: &agentID,
			AppRepository:      &appRepo,
			BacklogRepository:  &backlogRepo,
			IssuesSource:       &issuesSource,
			AgentHarness:       &agentHarness,
			ProvisionedAppID:   &appID,
			ProvisionedLineID:  &lineID,
		}))
		complete := true

		_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{
			Id:       factory.ID.String(),
			Complete: &complete,
		})
		require.NoError(t, err)
		assert.Zero(t, discussionHandlerCount(t, db, factory.ID))
	})

	t.Run("completion does not request another connection", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		appID, lineID := createOnboardingResources(t, r, factory)
		req := readyOnboardingRequest(factory.ID.String(), vcsID, agentID, appID, lineID)
		complete := true
		req.Complete = &complete

		sqlDB, err := database.Conn().DB()
		require.NoError(t, err)
		previousOpen := sqlDB.Stats().MaxOpenConnections
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		t.Cleanup(func() {
			sqlDB.SetMaxOpenConns(previousOpen)
			sqlDB.SetMaxIdleConns(previousOpen)
		})

		ctx, cancel := context.WithTimeout(onboardingContext(r), 15*time.Second)
		defer cancel()
		_, err = UpdateFactoryOnboarding(ctx, onboardingDeps(r), r.Organization.ID.String(), req)
		require.NoError(t, err)
		assertDiscussionHandler(t, db, factory, "github", 1)
	})
}

func assertDiscussionHandler(t *testing.T, db *gorm.DB, factory *models.Factory, provider string, want int) {
	t.Helper()
	require.Equal(t, int64(want), discussionHandlerCount(t, db, factory.ID))
	if want == 0 {
		return
	}
	handlers, err := factory.ListPRFeedbackHandlers(db)
	require.NoError(t, err)
	require.Len(t, handlers, want)
	canvas, err := models.FindCanvasInTransaction(db, factory.OrganizationID, handlers[0].CanvasID)
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
	settings := prFeedbackSettingsFromGraph(resolvePRFeedbackGraph(spec), spec)
	assert.Equal(t, prFeedbackDefaultMention, settings.Mention)
	assert.True(t, settings.IgnoreBots)
	assert.True(t, settings.AddressAllBots)
	assert.Empty(t, settings.AllowedBots)
	prompt := discussionRunnerPrompt(t, version.Nodes)
	assert.Contains(t, prompt, "Read every bot comment, including comments with no mention.")
	assert.Contains(t, prompt, "Ignore comments SuperPlane already wrote.")
	assert.Contains(t, prompt, prFeedbackDefaultMention)
	component := "github.onPRComment"
	if provider == models.ProviderBitbucket {
		component = "bitbucket.onPullRequestComment"
	}
	trigger := findIntakeNode(version.Nodes, prFeedbackCommentTriggerNodeID)
	require.NotNil(t, trigger)
	assert.Equal(t, component, trigger.ComponentName())
	assert.Equal(t, true, trigger.Configuration["addressAllBots"])
}

func discussionHandlerCount(t *testing.T, db *gorm.DB, factoryID uuid.UUID) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&models.FactoryPRFeedbackHandler{}).
		Where("factory_id = ? AND source = ?", factoryID, models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion).
		Count(&count).Error)
	return count
}

func discussionRunnerPrompt(t *testing.T, nodes []models.Node) string {
	t.Helper()
	node := findIntakeNode(nodes, prFeedbackRunnerNodeID)
	require.NotNil(t, node)
	steps, ok := node.Configuration["steps"].([]any)
	require.True(t, ok)
	for _, item := range steps {
		step, ok := item.(map[string]any)
		if !ok || step["name"] != "Address PR feedback" {
			continue
		}
		prompt, _ := step["prompt"].(string)
		return prompt
	}
	t.Fatal("missing discussion prompt")
	return ""
}

func onboardingContext(r *support.ResourceRegistry) context.Context {
	return authentication.SetUserIdInMetadata(context.Background(), r.User.String())
}

func onboardingDeps(r *support.ResourceRegistry) IntakeDependencies {
	return IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}
}

func readyOnboardingRequest(factoryID, vcsID, agentID, appID, lineID string) *pb.UpdateFactoryOnboardingRequest {
	appRepo := "acme/api"
	backlogRepo := "acme/backlog"
	issuesSource := pb.FactoryOnboarding_ISSUES_SOURCE_SKIP
	agentHarness := pb.FactoryOnboarding_AGENT_HARNESS_CLAUDE_CODE

	return &pb.UpdateFactoryOnboardingRequest{
		Id:                 factoryID,
		VcsIntegrationId:   &vcsID,
		AgentIntegrationId: &agentID,
		AppRepository:      &appRepo,
		BacklogRepository:  &backlogRepo,
		IssuesSource:       &issuesSource,
		AgentHarness:       &agentHarness,
		ProvisionedAppId:   &appID,
		ProvisionedLineId:  &lineID,
	}
}

func createOnboardingResources(
	t *testing.T,
	r *support.ResourceRegistry,
	factory *models.Factory,
) (string, string) {
	t.Helper()
	app, _ := support.CreateCanvas(t, factory.OrganizationID, r.User, nil, nil)
	require.NoError(t, database.DB(t.Context()).Model(app).Update("factory_id", factory.ID).Error)
	line, err := factory.CreateLine(database.DB(t.Context()), support.RandomName("line"), []models.FactoryLineStep{
		{
			Type:       models.FactoryLineStepTypeRunApp,
			AppID:      app.ID,
			Entrypoint: "work-order-dispatch",
		},
	})
	require.NoError(t, err)
	return app.ID.String(), line.ID.String()
}

func assertGitHubVCSCapabilities(t *testing.T, capabilities *pb.FactoryVCSCapabilities) {
	t.Helper()
	require.NotNil(t, capabilities)
	assert.True(t, capabilities.PrClosure)
	assert.True(t, capabilities.PrFeedback)
	assert.True(t, capabilities.PrChecks)
	assert.True(t, capabilities.MergeConfidence)
	assert.True(t, capabilities.Velocity)
	assert.True(t, capabilities.BoardMerge)
	assert.True(t, capabilities.BoardClose)
	assert.True(t, capabilities.VcsIssueIntake)
}

func assertBitbucketVCSCapabilities(t *testing.T, capabilities *pb.FactoryVCSCapabilities) {
	t.Helper()
	require.NotNil(t, capabilities)
	assert.True(t, capabilities.PrClosure)
	assert.True(t, capabilities.PrFeedback)
	assert.True(t, capabilities.PrChecks)
	assert.True(t, capabilities.MergeConfidence)
	assert.True(t, capabilities.Velocity)
	assert.True(t, capabilities.BoardMerge)
	assert.True(t, capabilities.BoardClose)
	assert.False(t, capabilities.VcsIssueIntake)
}

func createReadyOnboardingIntegration(t *testing.T, organizationID uuid.UUID, appName string) string {
	t.Helper()
	integration, err := models.CreateIntegration(
		uuid.New(),
		organizationID,
		appName,
		support.RandomName(appName),
		map[string]any{},
	)
	require.NoError(t, err)
	require.NoError(t, database.DB(t.Context()).Model(integration).Update("state", models.IntegrationStateReady).Error)
	return integration.ID.String()
}

func upsertHostedOnboardingProvider(t *testing.T, db *gorm.DB) {
	t.Helper()
	settings, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.DB(context.Background()).Model(&models.InstallationLLMSettings{}).Where("id = ?", settings.ID).Updates(map[string]any{
			"default_hosted_provider": settings.DefaultHostedProvider,
			"default_hosted_model":    settings.DefaultHostedModel,
		}).Error)
	})
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("test-hosted-key"),
		AllowedModels: datatypes.JSONSlice[string]{"sonnet"},
	})
	require.NoError(t, err)
	provider := models.UsageProviderAnthropic
	model := "sonnet"
	hostedSettings := *settings
	hostedSettings.DefaultHostedProvider = &provider
	hostedSettings.DefaultHostedModel = &model
	_, err = models.UpdateInstallationLLMSettings(db, hostedSettings)
	require.NoError(t, err)
}

func clearHostedLLMProviders(t *testing.T, db *gorm.DB) {
	t.Helper()
	existing, err := models.ListHostedLLMProviders(db)
	require.NoError(t, err)
	require.NoError(t, db.Where("provider <> ?", "").Delete(&models.HostedLLMProvider{}).Error)
	t.Cleanup(func() {
		for _, provider := range existing {
			_, upsertErr := models.UpsertHostedLLMProvider(db, provider)
			require.NoError(t, upsertErr)
		}
	})
}

func TestFactoryOnboardingRejectsBYOKAfterRevocation(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	upsertHostedOnboardingProvider(t, db)
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String(), AgentIntegrationId: &integrationID})
	require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	// Store an old preference directly to represent a previously allowed setup.
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AgentIntegrationID: &integrationID}))
	complete := true
	_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), &pb.UpdateFactoryOnboardingRequest{Id: factory.ID.String(), Complete: &complete})
	require.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
}

func TestOnboardingRequiresBillingBeforeCompletion(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(billing.BillingEnabledEnv, "yes")
	db := database.DB(t.Context())
	upsertHostedOnboardingProvider(t, db)
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	vcsID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
	appID, lineID := createOnboardingResources(t, r, factory)
	req := readyOnboardingRequest(factory.ID.String(), vcsID, "", appID, lineID)
	req.AgentIntegrationId = nil
	complete := true
	req.Complete = &complete

	_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err), "welcome credit must not bypass plan selection")
	reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	require.False(t, reloaded.IsOnboardingComplete())

	plan := billingmodels.BillingPlan{Key: "onboarding_" + uuid.NewString()[:8], Name: "Card required", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never"}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	_, err = billingmodels.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	plan.RequirePaymentMethod = true
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	hasCard := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/customers/external/"+r.Organization.ID.String()+"/payment-methods", request.URL.Path)
		if !hasCard {
			_, _ = w.Write([]byte(`{"items":[],"pagination":{"max_page":1}}`))
			return
		}
		_, _ = fmt.Fprintf(w, `{"items":[{"id":"card","type":"card","method_metadata":{"exp_month":1,"exp_year":%d}}],"pagination":{"max_page":1}}`, time.Now().Year()+1)
	}))
	defer server.Close()
	t.Setenv("POLAR_ACCESS_TOKEN", "test")
	t.Setenv("POLAR_API_BASE_URL", server.URL)
	_, err = UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
	require.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err), "an active plan must not bypass its card requirement")
	reloaded, err = models.FindFactory(db, r.Organization.ID, factory.ID)
	require.NoError(t, err)
	require.False(t, reloaded.IsOnboardingComplete())
	hasCard = true
	response, err := UpdateFactoryOnboarding(onboardingContext(r), onboardingDeps(r), r.Organization.ID.String(), req)
	require.NoError(t, err)
	require.NotNil(t, response.Factory.Onboarding.CompletedAt)
}
