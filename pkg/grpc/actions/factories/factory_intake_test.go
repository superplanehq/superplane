package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

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
		assert.Len(t, liveVersion.Nodes, 3)
		assert.Len(t, liveVersion.Edges, 2)
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

	t.Run("creating an intake also creates a Backlog scorer", func(t *testing.T) {
		factory := newFactory(t)
		create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		require.NotNil(t, liveBacklogCanvas(t, factory))
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
		assert.Contains(t, expression, `!(root().data.issue.labels.exists(label, label.name in ["bug"]))`)
		assert.Contains(t, expression, "size(root().data.issue.assignees) == 0")
	})

	t.Run("the authors filter reaches the filter expression when on", func(t *testing.T) {
		factory := newFactory(t)
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

		expression := ""
		for _, node := range liveVersion.Nodes {
			if node.ID == intakeFilterNodeID {
				expression, _ = node.Configuration["expression"].(string)
			}
		}
		assert.Contains(t, expression, `root().data.issue.author_association in ["COLLABORATOR", "MEMBER", "OWNER"]`)
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

	t.Run("a github intake whose filter node was removed rejects a filter change", func(t *testing.T) {
		factory := newFactory(t)
		intake := create(t, factory, &pb.CreateFactoryIntakeRequest{Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES})

		canvas, err := models.FindCanvasInTransaction(database.DB(t.Context()), r.Organization.ID, uuid.MustParse(intake.GetCanvasId()))
		require.NoError(t, err)
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(t.Context()), canvas)
		require.NoError(t, err)

		// Simulate a manual edit that removed the filter node (and the edges
		// that touched it), the way `applyIntakeSettings` would find the
		// canvas if a user rewired it by hand.
		nodesWithoutFilter := make([]models.Node, 0, len(liveVersion.Nodes)-1)
		for _, node := range liveVersion.Nodes {
			if node.ID != intakeFilterNodeID {
				nodesWithoutFilter = append(nodesWithoutFilter, node)
			}
		}
		require.Len(t, nodesWithoutFilter, len(liveVersion.Nodes)-1)

		edgesWithoutFilter := make([]models.Edge, 0, len(liveVersion.Edges))
		for _, edge := range liveVersion.Edges {
			if edge.SourceID != intakeFilterNodeID && edge.TargetID != intakeFilterNodeID {
				edgesWithoutFilter = append(edgesWithoutFilter, edge)
			}
		}

		err = database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
			return canvases.PublishGeneratedCanvasNodes(
				ctx,
				tx,
				canvas,
				r.User,
				"test: remove filter node",
				nodesWithoutFilter,
				edgesWithoutFilter,
				changesets.CanvasPublisherOptions{
					Registry:       r.Registry,
					OrgID:          canvas.OrganizationID,
					Encryptor:      r.Encryptor,
					AuthService:    r.AuthService,
					WebhookBaseURL: "http://localhost:8000",
					GitProvider:    r.GitProvider,
				},
			)
		})
		require.NoError(t, err)

		_, err = UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings: &pb.FactoryIntake_Settings{
				Labels: []string{"bug"},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)

		// A request that does not actually change any filter is still a
		// no-op, even without a filter node to write into.
		_, err = UpdateFactoryIntake(ctx, deps, orgID, &pb.UpdateFactoryIntakeRequest{
			FactoryId: factory.ID.String(),
			IntakeId:  intake.GetId(),
			Settings:  &pb.FactoryIntake_Settings{},
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
