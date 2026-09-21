package factories

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"

	// The intake graph uses built-in components and integration triggers, which
	// only reach the registry through their init functions.
	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__FactoryIntakeActions(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	create := func(t *testing.T, factory *models.Factory, req *pb.CreateFactoryIntakeRequest) *pb.FactoryIntake {
		t.Helper()
		req.FactoryId = factory.ID.String()
		response, err := CreateFactoryIntake(ctx, deps, orgID, req)
		require.NoError(t, err)
		return response.GetIntake()
	}

	t.Run("creating an intake builds a live canvas that can run", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		assert.Equal(t, pb.FactoryIntake_SOURCE_GITHUB_ISSUES, intake.GetSource())
		assert.Equal(t, "GitHub issues", intake.GetName())
		assert.True(t, intake.GetHealthy())
		assert.Equal(t, int32(DefaultIntakeConfidencePct), intake.GetSettings().GetConfidencePct())
		assert.True(t, intake.GetSettings().GetNewIssues())
		assert.True(t, intake.GetSettings().GetReopenedIssues())
		assert.True(t, intake.GetSettings().GetSuperplaneLabelAdded())
		assert.False(t, intake.GetSettings().GetAuthorsWithAccess())
		assert.Equal(t, pb.FactoryIntake_INITIAL_IMPORT_STATUS_SKIPPED, intake.GetInitialImportStatus())
		assert.Nil(t, intake.InitialImportItemCount)

		// The graph has to be live, not staged: a staged graph never receives
		// events.
		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		require.NotNil(t, canvas.FactoryID)
		assert.Equal(t, factory.ID, *canvas.FactoryID)

		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		assert.Len(t, liveVersion.Nodes, 3)
		assert.Len(t, liveVersion.Edges, 2)
	})

	t.Run("creating a Productive.io intake builds a healthy trigger to createWorkOrder canvas", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_PRODUCTIVE_TASKS})

		assert.Equal(t, pb.FactoryIntake_SOURCE_PRODUCTIVE_TASKS, intake.GetSource())
		assert.Equal(t, "Productive.io tasks", intake.GetName())
		assert.False(t, intake.GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_MISSING_INTEGRATION, intake.GetHealth())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		assert.Len(t, liveVersion.Nodes, 2)
		assert.Len(t, liveVersion.Edges, 1)

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		assert.Equal(t, "productive.onTask", trigger.ComponentName())
	})

	t.Run("a Productive.io intake listens to the selected project", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "productive")

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_PRODUCTIVE_TASKS,
			IntegrationId: integrationID,
			ResourceId:    "project-42",
		})

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, integrationID, *trigger.IntegrationID)
		assert.Equal(t, "project-42", trigger.Configuration["project"])
		assert.Equal(t, []any{"created"}, trigger.Configuration["actions"])
	})

	t.Run("a Sentry intake listens to the selected project", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			IntegrationId: integrationID,
			ResourceId:    "payments",
		})

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, integrationID, *trigger.IntegrationID)
		assert.Equal(t, "payments", trigger.Configuration["project"])
		assert.Equal(t, []any{"created", "unresolved"}, trigger.Configuration["actions"])
		assert.Equal(t, integrationID, intake.GetIntegrationId())
		assert.Equal(t, "payments", intake.GetResourceId())
		assert.True(t, intake.GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_OK, intake.GetHealth())
	})

	t.Run("a Sentry intake listens to the production project from setup", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			IntegrationId: integrationID,
			ResourceId:    "production",
		})

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, integrationID, *trigger.IntegrationID)
		assert.Equal(t, "production", trigger.Configuration["project"])
	})

	t.Run("skipping the initial import starts a bound Sentry intake empty", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:            pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			IntegrationId:     integrationID,
			ResourceId:        "payments",
			SkipInitialImport: true,
		})

		assert.Equal(t, pb.FactoryIntake_INITIAL_IMPORT_STATUS_SKIPPED, intake.GetInitialImportStatus())
		assert.Nil(t, intake.InitialImportItemCount)

		runs, err := ListFactoryIntakeRuns(ctx, orgID, &pb.ListFactoryIntakeRunsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
		})
		require.NoError(t, err)
		assert.Empty(t, runs.GetRuns())

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, integrationID, *trigger.IntegrationID)
		assert.Equal(t, "payments", trigger.Configuration["project"])
	})

	t.Run("skipping the initial import starts a bound Jira intake empty", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyJiraIntakeIntegration(t, r.Organization.ID, "ENG")

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:            pb.FactoryIntake_SOURCE_JIRA_ISSUES,
			IntegrationId:     integrationID,
			ResourceId:        "ENG",
			SkipInitialImport: true,
		})

		assert.Equal(t, pb.FactoryIntake_INITIAL_IMPORT_STATUS_SKIPPED, intake.GetInitialImportStatus())
		assert.Nil(t, intake.InitialImportItemCount)

		runs, err := ListFactoryIntakeRuns(ctx, orgID, &pb.ListFactoryIntakeRunsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
		})
		require.NoError(t, err)
		assert.Empty(t, runs.GetRuns())

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, integrationID, *trigger.IntegrationID)
		assert.Equal(t, "ENG", trigger.Configuration["project"])
	})

	t.Run("a Jira intake listens to the selected project and stays unhealthy until the webhook is ready", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyJiraIntakeIntegration(t, r.Organization.ID, "ENG")

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_JIRA_ISSUES,
			IntegrationId: integrationID,
			ResourceId:    "ENG",
		})

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, integrationID, *trigger.IntegrationID)
		assert.Equal(t, "ENG", trigger.Configuration["project"])
		assert.Equal(t, pb.FactoryIntake_SOURCE_JIRA_ISSUES, intake.GetSource())
		assert.False(t, intake.GetHealthy(), "a pending Jira webhook must not look like a live intake")
		assert.Equal(t, pb.FactoryIntake_HEALTH_WEBHOOK_NOT_READY, intake.GetHealth())

		canvasID := uuid.MustParse(intake.GetCanvasId())
		node, err := models.FindCanvasNode(database.DB(t.Context()), canvasID, intakeTriggerNodeID)
		require.NoError(t, err)
		reason := ""
		if node.StateReason != nil {
			reason = *node.StateReason
		}
		require.Equal(t, models.CanvasNodeStateReady, node.State, reason)
		require.NotNil(t, node.WebhookID)

		webhook, err := models.FindWebhookInTransaction(database.DB(t.Context()), *node.WebhookID)
		require.NoError(t, err)
		assert.Equal(t, models.WebhookStatePending, webhook.State)
		raw, err := json.Marshal(webhook.Configuration.Data())
		require.NoError(t, err)
		assert.Contains(t, string(raw), `"ENG"`)

		require.NoError(t, webhook.ReadyWithMetadata(database.DB(t.Context()), map[string]any{
			"webhookId": int64(1000),
		}))

		listed, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, listed.GetIntakes(), 1)
		assert.True(t, listed.GetIntakes()[0].GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_OK, listed.GetIntakes()[0].GetHealth())

		require.NoError(t, webhook.MarkFailed(database.DB(t.Context())))
		listed, err = ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, listed.GetIntakes(), 1)
		assert.False(t, listed.GetIntakes()[0].GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_WEBHOOK_NOT_READY, listed.GetIntakes()[0].GetHealth())
	})

	t.Run("a listed Sentry intake is unhealthy after its integration is deleted", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")
		create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			IntegrationId: integrationID,
			ResourceId:    "payments",
		})

		integration, err := models.FindIntegrationInTransaction(
			database.DB(t.Context()),
			r.Organization.ID,
			uuid.MustParse(integrationID),
		)
		require.NoError(t, err)
		require.NoError(t, integration.SoftDeleteInTransaction(database.DB(t.Context())))

		response, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, response.GetIntakes(), 1)
		assert.False(t, response.GetIntakes()[0].GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_MISSING_INTEGRATION, response.GetIntakes()[0].GetHealth())
		assert.Equal(t, integrationID, response.GetIntakes()[0].GetIntegrationId())
		assert.Equal(t, "payments", response.GetIntakes()[0].GetResourceId())
	})

	t.Run("a listed Sentry intake is unhealthy while its integration is not ready", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")
		create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			IntegrationId: integrationID,
			ResourceId:    "payments",
		})

		require.NoError(t, database.DB(t.Context()).Model(&models.Integration{}).
			Where("id = ?", uuid.MustParse(integrationID)).
			Update("state", models.IntegrationStateError).Error)

		response, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, response.GetIntakes(), 1)
		assert.False(t, response.GetIntakes()[0].GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_INTEGRATION_NOT_READY, response.GetIntakes()[0].GetHealth())
	})

	t.Run("an unbound GitHub intake stays healthy when listed", func(t *testing.T) {
		factory := newFactory(t)
		create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		response, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, response.GetIntakes(), 1)
		assert.True(t, response.GetIntakes()[0].GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_OK, response.GetIntakes()[0].GetHealth())
	})

	t.Run("an unbound Sentry intake needs a connection when listed", func(t *testing.T) {
		factory := newFactory(t)
		create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS})

		response, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, response.GetIntakes(), 1)
		assert.False(t, response.GetIntakes()[0].GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_MISSING_INTEGRATION, response.GetIntakes()[0].GetHealth())
	})

	t.Run("creating a Sentry intake with an invalid user id does not panic", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")
		invalidUserCtx := authentication.SetUserIdInMetadata(context.Background(), "not-a-uuid")

		require.NotPanics(t, func() {
			_, err := CreateFactoryIntake(invalidUserCtx, deps, orgID, &pb.CreateFactoryIntakeRequest{
				FactoryId:     factory.ID.String(),
				Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
				IntegrationId: integrationID,
				ResourceId:    "production",
			})
			require.Error(t, err)
			assert.Equal(t, codes.Unauthenticated, grpcerrors.Code(err))
			message, ok := grpcerrors.HandlerMessage(err)
			require.True(t, ok)
			assert.Equal(t, "user not authenticated", message)
		})
	})

	t.Run("a Sentry intake rejects an integration of another type", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")

		_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			IntegrationId: integrationID,
			ResourceId:    "payments",
		})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("a Productive.io intake rejects an integration of another type", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")

		_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			Source:        pb.FactoryIntake_SOURCE_PRODUCTIVE_TASKS,
			IntegrationId: integrationID,
			ResourceId:    "project-42",
		})

		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("a GitHub intake listens with the workspace connection", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		backlogRepository := "acme/backlog"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			VCSIntegrationID:  &integrationID,
			BacklogRepository: &backlogRepository,
		}))

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, integrationID, *trigger.IntegrationID)
		assert.Equal(t, backlogRepository, trigger.Configuration["repository"])
	})

	t.Run("an intake stays unbound when setup named no repository", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		trigger := liveIntakeTrigger(t, r.Organization.ID, intake)
		assert.Nil(t, trigger.IntegrationID)
		assert.NotContains(t, trigger.Configuration, "repository")
	})

	t.Run("creating an intake also creates Backlog template version 2", func(t *testing.T) {
		factory := newFactory(t)
		create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		backlog := liveBacklogCanvas(t, factory)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), backlog)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryAppTemplateBacklogID, models.FactoryAppTemplateID(liveVersion.Nodes))
		assert.Equal(t, backlogTemplateVersion, backlogTemplateVersionFrom(liveVersion.Nodes))
		assert.Len(t, liveVersion.Nodes, 7)
	})

	t.Run("creating a work order emits to the Backlog trigger", func(t *testing.T) {
		factoryModel := newFactory(t)
		create(t, factoryModel, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		_, err := CreateWorkOrder(ctx, orgID, &pb.CreateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			Title:     "Show a clearer empty state",
		})
		require.NoError(t, err)

		backlog := liveBacklogCanvas(t, factoryModel)
		events, err := models.ListCanvasEvents(database.DB(t.Context()), backlog.ID, backlogTriggerNodeID, 10, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)

		payload, ok := events[0].Data.Data().(map[string]any)
		require.True(t, ok)
		assert.Equal(t, factory.OnWorkOrderPayloadType, payload["type"])
	})

	t.Run("the Backlog canvas does not create work orders", func(t *testing.T) {
		factoryModel := newFactory(t)
		create(t, factoryModel, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		backlog := liveBacklogCanvas(t, factoryModel)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), backlog)
		require.NoError(t, err)

		for _, node := range liveVersion.Nodes {
			assert.NotEqual(t, intakeCreateComponent, node.ComponentName())
		}
	})

	t.Run("a second intake reuses the Backlog scorer", func(t *testing.T) {
		factory := newFactory(t)
		create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})
		create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS})

		canvases, err := factory.ListCanvases(database.DB(t.Context()))
		require.NoError(t, err)
		assert.Len(t, canvases, 3)
	})

	t.Run("an intake of a set up workspace has no incomplete node", func(t *testing.T) {
		factory := newFactory(t)
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AgentIntegrationID: &agentID,
		}))

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		for _, node := range liveIntakeNodes(t, r.Organization.ID, intake) {
			if node.ID == intakeTriggerNodeID {
				continue
			}
			assert.Nilf(t, node.ErrorMessage, "node %s is incomplete: %s", node.ID, nodeErrorMessage(node))
		}
	})

	t.Run("an intake analyzes a batch of items in parallel", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		// A node without a concurrency spec runs one execution at a time, so a
		// seeded batch would take as long as the sum of its analyses. Both the
		// canvas version and the node record have to carry the spec: the
		// scheduler reads the record.
		canvasID := uuid.MustParse(intake.GetCanvasId())
		for _, node := range liveIntakeNodes(t, r.Organization.ID, intake) {
			if node.ID == intakeTriggerNodeID {
				continue
			}

			assert.Equalf(t, intakeConcurrencyMax, node.Concurrency.EffectiveMax(), "version node %s", node.ID)

			stored, err := models.FindCanvasNode(database.DB(t.Context()), canvasID, node.ID)
			require.NoError(t, err)
			assert.Equalf(t, intakeConcurrencyMax, stored.ConcurrencySpec().EffectiveMax(), "node record %s", node.ID)
		}
	})

	t.Run("a source can have several intakes", func(t *testing.T) {
		factory := newFactory(t)
		first := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})
		second := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		assert.NotEqual(t, first.GetId(), second.GetId())
		assert.NotEqual(t, first.GetCanvasId(), second.GetCanvasId())
		// Canvas names are unique inside a workspace, so the second one steps
		// aside instead of failing.
		assert.NotEqual(t, first.GetName(), second.GetName())

		response, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Len(t, response.GetIntakes(), 2)
	})

	t.Run("the caller can name the intake", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			Name:   "Crash triage",
		})

		assert.Equal(t, "Crash triage", intake.GetName())
	})

	t.Run("an unspecified source is rejected", func(t *testing.T) {
		factory := newFactory(t)
		_, err := CreateFactoryIntake(ctx, deps, orgID, &pb.CreateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("renaming goes through the update call", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_PAGERDUTY_INCIDENTS})

		name := "Incident triage"
		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Name:      &name,
		})
		require.NoError(t, err)
		assert.Equal(t, "Incident triage", response.GetIntake().GetName())
		assert.True(t, response.GetIntake().GetHealthy())
	})

	t.Run("update applies a new name and settings together", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		name := "Bug intake"
		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Name:      &name,
			Settings: &pb.FactoryIntake_Settings{
				Labels: []string{"bug"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "Bug intake", response.GetIntake().GetName())
		assert.Equal(t, []string{"bug"}, response.GetIntake().GetSettings().GetLabels())
	})

	t.Run("label and assignment filters reach the filter expression", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				Labels:          []string{"bug", "bug", "  "},
				LabelFilterMode: pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE,
				Assignment:      pb.FactoryIntake_Settings_ASSIGNMENT_UNASSIGNED,
			},
		})
		require.NoError(t, err)

		settings := response.GetIntake().GetSettings()
		assert.Equal(t, []string{"bug"}, settings.GetLabels())
		assert.Equal(t, pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE, settings.GetLabelFilterMode())
		assert.Equal(t, pb.FactoryIntake_Settings_ASSIGNMENT_UNASSIGNED, settings.GetAssignment())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		expression := ""
		for _, node := range liveVersion.Nodes {
			if node.ID == intakeFilterNodeID {
				expression, _ = node.Configuration["expression"].(string)
			}
		}
		assert.NotContains(t, expression, ">=")
		assert.Contains(t, expression, `!(any(root().data.issue.labels, .name in ["bug"]))`)
		assert.Contains(t, expression, "len(root().data.issue.assignees) == 0")
	})

	t.Run("the authors filter adds a repository permission gate when on", func(t *testing.T) {
		factory := newFactory(t)
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		backlogRepository := "acme/backlog"
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			VCSIntegrationID:  &integrationID,
			BacklogRepository: &backlogRepository,
		}))
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				AuthorsWithAccess: true,
			},
		})
		require.NoError(t, err)

		settings := response.GetIntake().GetSettings()
		assert.True(t, settings.GetAuthorsWithAccess())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		var permissionNode, authorFilterNode *models.Node
		for _, node := range liveVersion.Nodes {
			switch node.ID {
			case intakeAuthorPermissionNodeID:
				node := node
				permissionNode = &node
			case intakeAuthorFilterNodeID:
				node := node
				authorFilterNode = &node
			}
		}
		require.NotNil(t, permissionNode)
		assert.Equal(t, intakeAuthorPermissionComponent, permissionNode.ComponentName())
		assert.Equal(t, backlogRepository, permissionNode.Configuration["repository"])
		assert.Equal(t, "{{ root().data.issue.user.login }}", permissionNode.Configuration["username"])
		require.NotNil(t, authorFilterNode)
		assert.Equal(t, `root().data.permission != "none"`, authorFilterNode.Configuration["expression"])
		assert.Contains(t, liveVersion.Edges, models.Edge{
			Channel:  "true",
			SourceID: intakeAuthorFilterNodeID,
			TargetID: intakeCreateNodeID,
		})

		response, err = UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				AuthorsWithAccess: false,
			},
		})
		require.NoError(t, err)
		assert.False(t, response.GetIntake().GetSettings().GetAuthorsWithAccess())

		canvas, err = models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err = models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		for _, node := range liveVersion.Nodes {
			assert.NotEqual(t, intakeAuthorPermissionNodeID, node.ID)
			assert.NotEqual(t, intakeAuthorFilterNodeID, node.ID)
		}
		assert.Contains(t, liveVersion.Edges, models.Edge{
			Channel:  "true",
			SourceID: intakeFilterNodeID,
			TargetID: intakeCreateNodeID,
		})
	})

	t.Run("the authors filter stays off by default", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				Labels: []string{"bug"},
			},
		})
		require.NoError(t, err)

		settings := response.GetIntake().GetSettings()
		assert.False(t, settings.GetAuthorsWithAccess())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		expression := ""
		for _, node := range liveVersion.Nodes {
			if node.ID == intakeFilterNodeID {
				expression, _ = node.Configuration["expression"].(string)
			}
		}
		assert.NotContains(t, expression, "author_association")
	})

	t.Run("issue event settings reach the trigger and filter", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})
		newIssues := false
		reopenedIssues := false
		superplaneLabelAdded := true

		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				NewIssues:            &newIssues,
				ReopenedIssues:       &reopenedIssues,
				SuperplaneLabelAdded: &superplaneLabelAdded,
			},
		})
		require.NoError(t, err)

		settings := response.GetIntake().GetSettings()
		assert.False(t, settings.GetNewIssues())
		assert.False(t, settings.GetReopenedIssues())
		assert.True(t, settings.GetSuperplaneLabelAdded())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		for _, node := range liveVersion.Nodes {
			switch node.ID {
			case intakeTriggerNodeID:
				assert.Equal(t, []any{"labeled"}, node.Configuration["actions"])
			case intakeFilterNodeID:
				assert.Contains(t, node.Configuration["expression"], intakeSuperplaneLabelCondition)
			}
		}
	})

	t.Run("the new and re-opened toggles reach the trigger on their own", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})
		reopenedIssues := false

		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				ReopenedIssues: &reopenedIssues,
			},
		})
		require.NoError(t, err)

		settings := response.GetIntake().GetSettings()
		assert.True(t, settings.GetNewIssues())
		assert.False(t, settings.GetReopenedIssues())
		assert.True(t, settings.GetSuperplaneLabelAdded())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		for _, node := range liveVersion.Nodes {
			if node.ID == intakeTriggerNodeID {
				assert.Equal(t, []any{"opened", "labeled"}, node.Configuration["actions"])
			}
		}
	})

	t.Run("a source without a filter ignores label settings", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS})

		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				Labels:     []string{"bug"},
				Assignment: pb.FactoryIntake_Settings_ASSIGNMENT_ASSIGNED,
			},
		})
		require.NoError(t, err)
	})

	t.Run("updating one setting leaves the others alone", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				Labels:          []string{"bug"},
				LabelFilterMode: pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE,
			},
		})
		require.NoError(t, err)

		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				Labels: []string{"bug"},
			},
		})
		require.NoError(t, err)

		settings := response.GetIntake().GetSettings()
		assert.Equal(t, pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE, settings.GetLabelFilterMode())
	})

	t.Run("update sets and clears paused, and list returns it", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS})
		assert.False(t, intake.GetPaused())

		paused := true
		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Paused:    &paused,
		})
		require.NoError(t, err)
		assert.True(t, response.GetIntake().GetPaused())

		listed, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, listed.GetIntakes(), 1)
		assert.True(t, listed.GetIntakes()[0].GetPaused())

		paused = false
		response, err = UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Paused:    &paused,
		})
		require.NoError(t, err)
		assert.False(t, response.GetIntake().GetPaused())
	})

	t.Run("update rejects pause for a GitHub intake", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		paused := true
		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Paused:    &paused,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assert.False(t, intake.GetPaused())

		listed, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, listed.GetIntakes(), 1)
		assert.False(t, listed.GetIntakes()[0].GetPaused())
	})

	t.Run("update rebinds a Sentry intake to a ready integration", func(t *testing.T) {
		factory := newFactory(t)
		oldID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			IntegrationId: oldID,
			ResourceId:    "payments",
		})

		integration, err := models.FindIntegrationInTransaction(
			database.DB(t.Context()),
			r.Organization.ID,
			uuid.MustParse(oldID),
		)
		require.NoError(t, err)
		require.NoError(t, integration.SoftDeleteInTransaction(database.DB(t.Context())))

		newID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")
		resourceID := "checkout"
		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			IntakeId:      intake.GetId(),
			IntegrationId: &newID,
			ResourceId:    &resourceID,
		})
		require.NoError(t, err)
		assert.True(t, response.GetIntake().GetHealthy())
		assert.Equal(t, pb.FactoryIntake_HEALTH_OK, response.GetIntake().GetHealth())
		assert.Equal(t, newID, response.GetIntake().GetIntegrationId())
		assert.Equal(t, "checkout", response.GetIntake().GetResourceId())

		trigger := liveIntakeTrigger(t, r.Organization.ID, response.GetIntake())
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, newID, *trigger.IntegrationID)
		assert.Equal(t, "checkout", trigger.Configuration["project"])
	})

	t.Run("update rejects a binding with only an integration", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS})
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "sentry")

		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			IntakeId:      intake.GetId(),
			IntegrationId: &integrationID,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("update rejects a GitHub connection change", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		resourceID := "acme/backlog"

		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			IntakeId:      intake.GetId(),
			IntegrationId: &integrationID,
			ResourceId:    &resourceID,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("update applies settings and a new connection together", func(t *testing.T) {
		factory := newFactory(t)
		oldID := createReadyJiraIntakeIntegration(t, r.Organization.ID, "ENG")
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_JIRA_ISSUES,
			IntegrationId: oldID,
			ResourceId:    "ENG",
		})

		newID := createReadyJiraIntakeIntegration(t, r.Organization.ID, "OPS")
		resourceID := "OPS"
		newIssues := false
		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			IntakeId:      intake.GetId(),
			IntegrationId: &newID,
			ResourceId:    &resourceID,
			Settings: &pb.FactoryIntake_Settings{
				NewIssues: &newIssues,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, newID, response.GetIntake().GetIntegrationId())
		assert.Equal(t, "OPS", response.GetIntake().GetResourceId())
		assert.False(t, response.GetIntake().GetSettings().GetNewIssues())
		assert.True(t, response.GetIntake().GetSettings().GetReopenedIssues())

		trigger := liveIntakeTrigger(t, r.Organization.ID, response.GetIntake())
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, newID, *trigger.IntegrationID)
		assert.Equal(t, "OPS", trigger.Configuration["project"])
		assert.Equal(t, []any{"updated"}, trigger.Configuration["events"])
	})

	t.Run("update rejects a bad connection without keeping settings", func(t *testing.T) {
		factory := newFactory(t)
		oldID := createReadyJiraIntakeIntegration(t, r.Organization.ID, "ENG")
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_JIRA_ISSUES,
			IntegrationId: oldID,
			ResourceId:    "ENG",
		})
		assert.True(t, intake.GetSettings().GetNewIssues())

		missingID := uuid.NewString()
		resourceID := "OPS"
		newIssues := false
		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			IntakeId:      intake.GetId(),
			IntegrationId: &missingID,
			ResourceId:    &resourceID,
			Settings: &pb.FactoryIntake_Settings{
				NewIssues: &newIssues,
			},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

		listed, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, listed.GetIntakes(), 1)
		assert.Equal(t, oldID, listed.GetIntakes()[0].GetIntegrationId())
		assert.Equal(t, "ENG", listed.GetIntakes()[0].GetResourceId())
		assert.True(t, listed.GetIntakes()[0].GetSettings().GetNewIssues())

		trigger := liveIntakeTrigger(t, r.Organization.ID, listed.GetIntakes()[0])
		require.NotNil(t, trigger.IntegrationID)
		assert.Equal(t, oldID, *trigger.IntegrationID)
		assert.Equal(t, "ENG", trigger.Configuration["project"])
		assert.Equal(t, []any{"created", "updated"}, trigger.Configuration["events"])
	})

	t.Run("update rejects a GitHub connection change without keeping settings", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})
		integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, "github")
		resourceID := "acme/backlog"

		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId:     factory.ID.String(),
			IntakeId:      intake.GetId(),
			IntegrationId: &integrationID,
			ResourceId:    &resourceID,
			Settings: &pb.FactoryIntake_Settings{
				Labels: []string{"bug"},
			},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

		listed, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		require.Len(t, listed.GetIntakes(), 1)
		assert.Empty(t, listed.GetIntakes()[0].GetSettings().GetLabels())
	})

	t.Run("deleting an intake retires its canvas", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		_, err := DeleteFactoryIntake(ctx, orgID, &pb.DeleteFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
		})
		require.NoError(t, err)

		response, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Empty(t, response.GetIntakes())

		_, err = models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		assert.Error(t, err)
	})

	t.Run("deleting an intake leaves existing work orders", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS})
		origin := models.WorkOrderOrigin{
			URL:   "https://acme.sentry.io/issues/1",
			Label: "ISSUE-1",
		}
		order, err := factory.CreateWorkOrderWithOrigin(
			database.DB(t.Context()),
			"Crash in checkout",
			"The checkout page panics.",
			nil,
			[]uuid.UUID{},
			nil,
			origin,
		)
		require.NoError(t, err)

		_, err = DeleteFactoryIntake(ctx, orgID, &pb.DeleteFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
		})
		require.NoError(t, err)

		found, err := factory.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		require.NotNil(t, found.Origin())
		assert.Equal(t, origin.URL, found.Origin().URL)
		assert.Equal(t, origin.Label, found.Origin().Label)
	})

	t.Run("a missing intake reports not found", func(t *testing.T) {
		factory := newFactory(t)
		_, err := DeleteFactoryIntake(ctx, orgID, &pb.DeleteFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  uuid.New().String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("a new intake has no runs yet", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		response, err := ListFactoryIntakeRuns(ctx, orgID, &pb.ListFactoryIntakeRunsRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
		})
		require.NoError(t, err)
		assert.Empty(t, response.GetRuns())
	})
}

func Test__SerializeFactoryIntakeInitialImport(t *testing.T) {
	itemCount := 0
	intake := &models.FactoryIntake{
		ID:                     uuid.New(),
		FactoryID:              uuid.New(),
		CanvasID:               uuid.New(),
		Source:                 models.FactoryIntakeSourceGitHubIssues,
		InitialImportStatus:    models.FactoryIntakeInitialImportStatusCompleted,
		InitialImportItemCount: &itemCount,
	}

	serialized := serializeFactoryIntake(nil, intake, models.LiveCanvasSpec{}, nil)

	assert.Equal(t, pb.FactoryIntake_INITIAL_IMPORT_STATUS_COMPLETED, serialized.GetInitialImportStatus())
	require.NotNil(t, serialized.InitialImportItemCount)
	assert.Zero(t, serialized.GetInitialImportItemCount())
}

func Test__SerializeFactoryIntakeJiraWebhookHealth(t *testing.T) {
	spec := intakeSpecFromTemplate(t, models.FactoryIntakeSourceJiraIssues)
	graph := resolveIntakeGraph(models.FactoryIntakeSourceJiraIssues, spec)
	require.NotEmpty(t, graph.TriggerNodeID)

	integrationID := uuid.NewString()
	for i := range spec.Nodes {
		if spec.Nodes[i].ID != graph.TriggerNodeID {
			continue
		}
		spec.Nodes[i].IntegrationID = &integrationID
	}
	intake := &models.FactoryIntake{
		ID:        uuid.New(),
		FactoryID: uuid.New(),
		CanvasID:  uuid.New(),
		Source:    models.FactoryIntakeSourceJiraIssues,
	}

	serialized := serializeFactoryIntake(nil, intake, spec, map[string]string{
		integrationID: models.IntegrationStateReady,
	})

	assert.False(t, serialized.GetHealthy(), "a Jira intake without a ready webhook must not be healthy")
	assert.Equal(t, pb.FactoryIntake_HEALTH_WEBHOOK_NOT_READY, serialized.GetHealth())
}

func liveBacklogCanvas(t *testing.T, factoryModel *models.Factory) *models.Canvas {
	t.Helper()

	canvases, err := factoryModel.ListCanvases(database.DB(t.Context()))
	require.NoError(t, err)

	ids := make([]uuid.UUID, 0, len(canvases))
	byID := make(map[uuid.UUID]*models.Canvas, len(canvases))
	for i := range canvases {
		if canvases[i].LiveVersionID == nil {
			continue
		}
		ids = append(ids, canvases[i].ID)
		byID[canvases[i].ID] = &canvases[i]
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(database.DB(t.Context()), ids)
	require.NoError(t, err)

	for canvasID, spec := range specs {
		if onWorkOrderNodeIDFromSpec(spec) != "" {
			return byID[canvasID]
		}
	}

	require.Fail(t, "factory has no Backlog canvas")
	return nil
}

func liveIntakeTrigger(t *testing.T, organizationID uuid.UUID, intake *pb.FactoryIntake) models.Node {
	t.Helper()

	return liveIntakeNode(t, organizationID, intake, intakeTriggerNodeID)
}

func liveIntakeNode(t *testing.T, organizationID uuid.UUID, intake *pb.FactoryIntake, nodeID string) models.Node {
	t.Helper()

	for _, node := range liveIntakeNodes(t, organizationID, intake) {
		if node.ID == nodeID {
			return node
		}
	}

	require.Failf(t, "node not found", "intake canvas %s has no node %q", intake.GetCanvasId(), nodeID)
	return models.Node{}
}

func nodeErrorMessage(node models.Node) string {
	if node.ErrorMessage == nil {
		return ""
	}
	return *node.ErrorMessage
}

func liveIntakeNodes(t *testing.T, organizationID uuid.UUID, intake *pb.FactoryIntake) []models.Node {
	t.Helper()

	canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), organizationID, uuid.MustParse(intake.GetCanvasId()))
	require.NoError(t, err)

	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
	require.NoError(t, err)

	return liveVersion.Nodes
}

func createReadyJiraIntakeIntegration(t *testing.T, organizationID uuid.UUID, projectKey string) string {
	t.Helper()

	integration, err := models.CreateIntegration(
		uuid.New(),
		organizationID,
		"jira",
		support.RandomName("jira"),
		map[string]any{},
	)
	require.NoError(t, err)

	integration.State = models.IntegrationStateReady
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"projects": []any{
			map[string]any{"id": "10000", "key": projectKey, "name": projectKey},
		},
	})
	require.NoError(t, database.DB(t.Context()).Save(integration).Error)
	return integration.ID.String()
}
