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
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"

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
		GitProvider:    r.GitProvider,
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

		// The graph has to be live, not staged: a staged graph never receives
		// events.
		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		require.NotNil(t, canvas.FactoryID)
		assert.Equal(t, factory.ID, *canvas.FactoryID)

		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)
		assert.Len(t, liveVersion.Nodes, 6)
		assert.Len(t, liveVersion.Edges, 5)
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

	t.Run("an intake of a set up workspace has no incomplete node", func(t *testing.T) {
		factory := newFactory(t)
		agentID := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
		require.NoError(t, factory.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{
			AgentIntegrationID: &agentID,
		}))

		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		// A node the canvas reports an error on cannot run, so the intake
		// would look ready and never produce a work order. The trigger stays
		// out: it validates against the GitHub API, which the installation of
		// the test cannot answer. Its binding has a test of its own.
		for _, node := range liveIntakeNodes(t, r.Organization.ID, intake) {
			if node.ID == intakeTriggerNodeID {
				continue
			}
			assert.Nilf(t, node.ErrorMessage, "node %s is incomplete: %s", node.ID, nodeErrorMessage(node))
		}

		analysis := liveIntakeNode(t, r.Organization.ID, intake, intakeAnalysisNodeID)
		assert.Equal(t, "runnerClaudeCode", analysis.ComponentName())
		credentials, ok := analysis.Configuration["credentials"].(map[string]any)
		require.True(t, ok, "analysis node has no credentials")
		assert.Equal(t, "integration", credentials["source"])
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
		// Canvas names are unique per organization, so the second one steps
		// aside instead of failing.
		assert.NotEqual(t, first.GetName(), second.GetName())

		response, err := ListFactoryIntakes(ctx, orgID, &pb.ListFactoryIntakesRequest{FactoryId: factory.ID.String()})
		require.NoError(t, err)
		assert.Len(t, response.GetIntakes(), 2)
	})

	t.Run("the caller can name the intake and set its threshold", func(t *testing.T) {
		factory := newFactory(t)
		confidence := int32(90)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{
			Source:        pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS,
			Name:          "Crash triage",
			ConfidencePct: &confidence,
		})

		assert.Equal(t, "Crash triage", intake.GetName())
		assert.Equal(t, int32(90), intake.GetSettings().GetConfidencePct())
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

	t.Run("renaming and rethresholding go through one call", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_PAGERDUTY_INCIDENTS})

		name := "Incident triage"
		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Name:      &name,
			Settings:  &pb.FactoryIntake_Settings{ConfidencePct: 40},
		})
		require.NoError(t, err)
		assert.Equal(t, "Incident triage", response.GetIntake().GetName())
		assert.Equal(t, int32(40), response.GetIntake().GetSettings().GetConfidencePct())
		assert.True(t, response.GetIntake().GetHealthy())

		// The threshold has to end up in the live graph, because that is what
		// the workers read.
		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		graph := resolveIntakeGraph(models.FactoryIntakeSourcePagerDutyIncidents, models.LiveCanvasSpec{
			Nodes: liveVersion.Nodes,
			Edges: liveVersion.Edges,
		})
		assert.Equal(t, 40, graph.ConfidencePct)
	})

	t.Run("label and assignment filters reach the threshold expression", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				ConfidencePct:   70,
				Labels:          []string{"bug", "bug", "  "},
				LabelFilterMode: pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE,
				Assignment:      pb.FactoryIntake_Settings_ASSIGNMENT_UNASSIGNED,
			},
		})
		require.NoError(t, err)

		settings := response.GetIntake().GetSettings()
		assert.Equal(t, int32(70), settings.GetConfidencePct())
		assert.Equal(t, []string{"bug"}, settings.GetLabels())
		assert.Equal(t, pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE, settings.GetLabelFilterMode())
		assert.Equal(t, pb.FactoryIntake_Settings_ASSIGNMENT_UNASSIGNED, settings.GetAssignment())

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		expression := ""
		for _, node := range liveVersion.Nodes {
			if node.ID == intakeThresholdNodeID {
				expression, _ = node.Configuration["expression"].(string)
			}
		}
		assert.Contains(t, expression, ">= 70")
		assert.Contains(t, expression, `!(root().data.issue.labels.exists(label, label.name in ["bug"]))`)
		assert.Contains(t, expression, "size(root().data.issue.assignees) == 0")
	})

	t.Run("a source without labels keeps a plain threshold", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_SENTRY_EXCEPTIONS})

		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				ConfidencePct: 55,
				Labels:        []string{"bug"},
				Assignment:    pb.FactoryIntake_Settings_ASSIGNMENT_ASSIGNED,
			},
		})
		require.NoError(t, err)

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		for _, node := range liveVersion.Nodes {
			if node.ID == intakeThresholdNodeID {
				assert.Equal(t, intakeThresholdExpression(55), node.Configuration["expression"])
			}
		}
	})

	t.Run("updating one setting leaves the others alone", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		_, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				ConfidencePct:   70,
				Labels:          []string{"bug"},
				LabelFilterMode: pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE,
			},
		})
		require.NoError(t, err)

		response, err := UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				ConfidencePct: 30,
				Labels:        []string{"bug"},
			},
		})
		require.NoError(t, err)

		settings := response.GetIntake().GetSettings()
		assert.Equal(t, int32(30), settings.GetConfidencePct())
		assert.Equal(t, pb.FactoryIntake_Settings_LABEL_FILTER_MODE_EXCLUDE, settings.GetLabelFilterMode())
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
