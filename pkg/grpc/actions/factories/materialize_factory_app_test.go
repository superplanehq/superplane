package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__MaterializeFactoryAppDefaults(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		GitProvider:    r.GitProvider,
		WebhookBaseURL: "http://localhost:8000",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	newFactoryWithAppRepo := func(t *testing.T, repository string) *models.Factory {
		t.Helper()
		factory := newFactory(t)
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AppRepository: &repository,
		}))
		return factory
	}

	t.Run("the Backlog automation resets to its generated graph", func(t *testing.T) {
		factoryModel := newFactory(t)
		_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		backlog := liveBacklogCanvas(t, factoryModel)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     backlog.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, "backlog", response.GetTemplateId())

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		assert.Equal(t, backlog.ID.String(), defaults.Metadata.ID)
		assert.Equal(t, backlog.Name, defaults.Metadata.Name)
	})

	t.Run("a newly created app materializes its install template", func(t *testing.T) {
		factoryModel := newFactory(t)
		canvas := support.CreateFactoryCanvas(t, r, factoryModel.ID, support.RandomName("Plan"))

		response, err := MaterializeFactoryAppTemplate(ctx, orgID, &pb.MaterializeFactoryAppTemplateRequest{
			FactoryId:  factoryModel.ID.String(),
			TemplateId: "line-planning",
			AppId:      canvas.ID.String(),
			InstallParams: map[string]string{
				"appRepository": "acme/app",
				"defaultBranch": "main",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "line-planning", response.GetTemplateId())
		assert.NotEmpty(t, response.GetCanvasYaml())
		assert.NotEmpty(t, response.GetConsoleYaml())
	})

	t.Run("an app from another factory reports not found", func(t *testing.T) {
		factoryModel := newFactory(t)
		other := newFactory(t)
		canvas := support.CreateFactoryCanvas(t, r, other.ID, support.RandomName("Plan"))

		_, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     canvas.ID.String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("an intake resets to its generated graph", func(t *testing.T) {
		factoryModel := newFactory(t)
		intake, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     intake.GetIntake().GetCanvasId(),
		})
		require.NoError(t, err)
		assert.Equal(t, "intake:"+models.FactoryIntakeSourceGitHubIssues, response.GetTemplateId())

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		assert.Equal(t, intake.GetIntake().GetCanvasId(), defaults.Metadata.ID)
		assertNoRunnerNode(t, defaults)
	})

	t.Run("Plan with Claude BYOK resets to SuperPlane when the instance default is set", func(t *testing.T) {
		factoryModel := newFactory(t)
		canvas := createClaudePlanningCanvas(t, r, factoryModel.ID)
		enableInstanceSuperPlaneDefault(t)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     canvas.ID.String(),
		})
		require.NoError(t, err)
		assert.Equal(t, "line-planning", response.GetTemplateId())

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		assertSuperPlaneRunnerNode(t, findYAMLNode(t, defaults, "planner-agent-no-issue"))
	})

	t.Run("Plan keeps the Claude agent when the instance default is unset", func(t *testing.T) {
		factoryModel := newFactory(t)
		canvas := createClaudePlanningCanvas(t, r, factoryModel.ID)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     canvas.ID.String(),
		})
		require.NoError(t, err)

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		agent := findYAMLNode(t, defaults, "planner-agent-no-issue")
		assert.Equal(t, "runnerClaudeCode", agent.Component)
		assert.Equal(t, "opus", agent.Configuration["model"])
		assert.Equal(t, map[string]any{
			"source":      "integration",
			"integration": map[string]any{"name": "claude"},
		}, agent.Configuration["credentials"])
	})

	t.Run("Backlog with Claude resets to SuperPlane when the instance default is set", func(t *testing.T) {
		factoryModel := newFactory(t)
		_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)
		backlog := liveBacklogCanvas(t, factoryModel)
		enableInstanceSuperPlaneDefault(t)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     backlog.ID.String(),
		})
		require.NoError(t, err)

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		assertSuperPlaneRunnerNode(t, findYAMLNode(t, defaults, intakeAnalysisNodeID))
	})

	t.Run("Backlog keeps the Claude agent when the instance default is unset", func(t *testing.T) {
		factoryModel := newFactory(t)
		_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		})
		require.NoError(t, err)
		backlog := liveBacklogCanvas(t, factoryModel)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     backlog.ID.String(),
		})
		require.NoError(t, err)

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		analysis := findYAMLNode(t, defaults, intakeAnalysisNodeID)
		assert.Equal(t, "runnerClaudeCode", analysis.Component)
		assert.Equal(t, "opus", analysis.Configuration["model"])
	})

	t.Run("a discussion PR feedback handler resets to its generated graph", func(t *testing.T) {
		factoryModel := newFactoryWithAppRepo(t, "acme/ship")
		handler, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factoryModel.ID.String(),
			Settings: &pb.FactoryPRFeedbackHandler_Settings{
				Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{
					Mention: "@shipbot",
				},
			},
		})
		require.NoError(t, err)
		enableInstanceSuperPlaneDefault(t)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     handler.GetHandler().GetCanvasId(),
		})
		require.NoError(t, err)
		assert.Equal(t, prFeedbackDiscussionTemplateID, response.GetTemplateId())
		assert.Empty(t, response.GetConsoleYaml())

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		trigger := findYAMLNode(t, defaults, prFeedbackCommentTriggerNodeID)
		assert.Equal(t, "acme/ship", trigger.Configuration["repository"])
		assert.Equal(t, "@shipbot", trigger.Configuration["contentFilter"])
		assert.Equal(t, map[string]any{
			"id":      prFeedbackDiscussionTemplateID,
			"version": float64(factoryTemplateVersion),
		}, trigger.Metadata[factoryTemplateMetadataKey])
		assertSuperPlaneRunnerNode(t, findYAMLNode(t, defaults, prFeedbackRunnerNodeID))
	})

	t.Run("a checks PR feedback handler resets to its generated graph", func(t *testing.T) {
		factoryModel := newFactoryWithAppRepo(t, "acme/ship")
		handler, err := CreateFactoryPRFeedbackHandler(ctx, deps, orgID, &pb.CreateFactoryPRFeedbackHandlerRequest{
			FactoryId: factoryModel.ID.String(),
			Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
		})
		require.NoError(t, err)
		enableInstanceSuperPlaneDefault(t)

		response, err := MaterializeFactoryAppDefaults(ctx, orgID, &pb.MaterializeFactoryAppDefaultsRequest{
			FactoryId: factoryModel.ID.String(),
			AppId:     handler.GetHandler().GetCanvasId(),
		})
		require.NoError(t, err)
		assert.Equal(t, prFeedbackChecksTemplateID, response.GetTemplateId())
		assert.Empty(t, response.GetConsoleYaml())

		defaults, err := yaml.CanvasFromYAML([]byte(response.GetCanvasYaml()))
		require.NoError(t, err)
		trigger := findYAMLNode(t, defaults, prFeedbackPullRequestTriggerNodeID)
		assert.Equal(t, "acme/ship", trigger.Configuration["repository"])
		assert.NotNil(t, findYAMLNode(t, defaults, prFeedbackWaitChecksNodeID))
		assert.Equal(t, map[string]any{
			"id":      prFeedbackChecksTemplateID,
			"version": float64(factoryTemplateVersion),
		}, trigger.Metadata[factoryTemplateMetadataKey])
		assertSuperPlaneRunnerNode(t, findYAMLNode(t, defaults, prFeedbackRunnerNodeID))
	})
}

func createClaudePlanningCanvas(t *testing.T, r *support.ResourceRegistry, factoryID uuid.UUID) *models.Canvas {
	t.Helper()
	canvas := support.CreateFactoryCanvas(t, r, factoryID, "Plan")
	require.NoError(t, database.DB(t.Context()).Model(&models.CanvasVersion{}).
		Where("id = ?", *canvas.LiveVersionID).
		Update("nodes", datatypes.NewJSONSlice([]models.Node{
			{
				ID:   "onrun-create-plan",
				Name: "On Run",
				Type: models.NodeTypeTrigger,
				Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}},
			},
			{
				ID:   "planner-agent-no-issue",
				Name: "Planner",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}},
				Configuration: map[string]any{
					"model": "opus",
					"credentials": map[string]any{
						"source":      "integration",
						"integration": map[string]any{"name": "claude"},
					},
				},
			},
		})).Error)
	return canvas
}

func enableInstanceSuperPlaneDefault(t *testing.T) {
	t.Helper()
	db := database.DB(t.Context())
	previous, err := models.GetInstallationLLMSettings(db)
	require.NoError(t, err)

	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"x-ai/grok-4.6"},
	})
	require.NoError(t, err)

	provider := models.UsageProviderOpenRouter
	model := "x-ai/grok-4.6"
	_, err = models.UpdateInstallationLLMSettings(db, models.InstallationLLMSettings{
		WelcomeGrantCents:     previous.WelcomeGrantCents,
		MarkupBPS:             previous.MarkupBPS,
		WarningThresholdBPS:   previous.WarningThresholdBPS,
		DefaultHostedProvider: &provider,
		DefaultHostedModel:    &model,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		conn := database.Conn()
		_, err := models.UpdateInstallationLLMSettings(conn, *previous)
		require.NoError(t, err)
		_ = conn.Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
	})
}

func assertSuperPlaneRunnerNode(t *testing.T, node *yaml.Node) {
	t.Helper()
	require.NotNil(t, node)
	assert.Equal(t, models.SuperPlaneRunnerComponent, node.Component)
	assert.Nil(t, node.Configuration["model"])
	assert.Nil(t, node.Configuration["credentials"])
}

func assertNoRunnerNode(t *testing.T, canvas *yaml.Canvas) {
	t.Helper()
	for _, node := range canvas.Spec.Nodes {
		assert.NotEqual(t, models.SuperPlaneRunnerComponent, node.Component)
		assert.NotContains(t, []string{"runnerClaudeCode", "runnerCodex", "runnerOpenRouter"}, node.Component)
	}
}
