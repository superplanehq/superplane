package factories

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func UpdateFactoryIntake(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.UpdateFactoryIntakeRequest,
) (*pb.UpdateFactoryIntakeResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	intakeID, err := parseIntakeID(req.GetIntakeId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	intake, err := factory.FindIntake(db, intakeID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	canvas, err := models.FindCanvasInTransaction(db, orgID, intake.CanvasID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	name, err := parseUpdatedIntakeName(req.Name)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}
	if err := validateIntakePause(intake.Source, req.Paused); err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	binding, err := resolveUpdatedIntakeBinding(db, factory, intake, req)
	if err != nil {
		return nil, err
	}

	if err := applyIntakeUpdate(ctx, deps, db, intake, canvas, req.Settings, binding, name, req.Paused); err != nil {
		return nil, err
	}

	intake, err = factory.FindIntake(db, intakeID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, []uuid.UUID{intake.CanvasID})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	states, err := intakeIntegrationStates(db, orgID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	return &pb.UpdateFactoryIntakeResponse{
		Intake: serializeFactoryIntake(db, intake, specs[intake.CanvasID], states),
	}, nil
}

func parseUpdatedIntakeName(name *string) (*string, error) {
	if name == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*name)
	if trimmed == "" {
		return nil, invalidArgument("intake name cannot be empty")
	}
	return &trimmed, nil
}

func validateIntakePause(source string, paused *bool) error {
	if paused == nil {
		return nil
	}
	if !intakeSourceSupportsPause(source) {
		return invalidArgument("pause is not supported for this intake")
	}
	return nil
}

func intakeSourceSupportsPause(source string) bool {
	return source == models.FactoryIntakeSourceGitHubIssues ||
		source == models.FactoryIntakeSourceSentryExceptions ||
		source == models.FactoryIntakeSourceJiraIssues ||
		source == models.FactoryIntakeSourceProductiveTasks ||
		source == models.FactoryIntakeSourceDependabotAlerts
}

func resolveUpdatedIntakeBinding(
	db *gorm.DB,
	factory *models.Factory,
	intake *models.FactoryIntake,
	req *pb.UpdateFactoryIntakeRequest,
) (*intakeBinding, error) {
	if req.IntegrationId == nil && req.ResourceId == nil {
		return nil, nil
	}
	if req.IntegrationId == nil || req.ResourceId == nil {
		return nil, factoryErrorToStatus(
			invalidArgument("intake integration and project are required"),
			"failed to update factory intake",
		)
	}
	if intake.Source == models.FactoryIntakeSourceGitHubIssues || intake.Source == models.FactoryIntakeSourceDependabotAlerts {
		return nil, factoryErrorToStatus(
			invalidArgument("GitHub intake connection follows workspace setup"),
			"failed to update factory intake",
		)
	}
	if !intakeSourceAllowsRebind(intake.Source) {
		return nil, factoryErrorToStatus(
			invalidArgument("this intake cannot change its connection"),
			"failed to update factory intake",
		)
	}

	binding, err := resolveIntakeBinding(db, factory, intake.Source, req.GetIntegrationId(), req.GetResourceId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}
	if binding == nil || binding.integrationRef() == nil {
		return nil, factoryErrorToStatus(
			invalidArgument("intake integration and project are required"),
			"failed to update factory intake",
		)
	}

	return binding, nil
}

// applyIntakeUpdate writes every requested change in one transaction. Settings
// and a new connection share one canvas publish so a failed save cannot leave
// only part of the request live.
func applyIntakeUpdate(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	intake *models.FactoryIntake,
	canvas *models.Canvas,
	settings *pb.FactoryIntake_Settings,
	binding *intakeBinding,
	name *string,
	paused *bool,
) error {
	if settings == nil && binding == nil && name == nil && paused == nil {
		return nil
	}

	var userID uuid.UUID
	if settings != nil || binding != nil {
		rawID, ok := authentication.GetUserIdFromMetadata(ctx)
		if !ok {
			return grpcerrors.Unauthenticated(nil, "user not authenticated")
		}
		userID = uuid.MustParse(rawID)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if settings != nil || binding != nil {
			if err := applyIntakeGraphUpdate(ctx, deps, tx, intake, canvas, userID, settings, binding); err != nil {
				return err
			}
		}
		if name != nil {
			if err := canvases.UpdateCanvasInTransaction(tx, canvas, name, nil, nil); err != nil {
				return err
			}
		}
		if paused != nil {
			if err := intake.SetPaused(tx, *paused); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return err
		}
		return factoryErrorToStatus(err, "failed to update factory intake")
	}

	if name != nil {
		canvases.PublishCanvasUpdated(canvas)
	}

	return nil
}

func applyIntakeGraphUpdate(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	intake *models.FactoryIntake,
	canvas *models.Canvas,
	userID uuid.UUID,
	settings *pb.FactoryIntake_Settings,
	binding *intakeBinding,
) error {
	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return err
	}

	spec := models.LiveCanvasSpec{Nodes: liveVersion.Nodes, Edges: liveVersion.Edges}
	graph := resolveIntakeGraph(intake.Source, spec)
	nodes := slices.Clone(liveVersion.Nodes)
	edges := slices.Clone(liveVersion.Edges)

	if settings != nil {
		nodes, edges, err = applyIntakeSettingsToGraph(intake.Source, graph, spec, settings, nodes, edges)
		if err != nil {
			return err
		}
	}
	if binding != nil {
		nodes, err = applyIntakeBindingToGraph(graph, binding, nodes)
		if err != nil {
			return err
		}
	}

	return canvases.PublishGeneratedCanvasNodes(
		ctx,
		tx,
		canvas,
		userID,
		intakeGraphUpdateMessage(settings, binding),
		nodes,
		edges,
		intakePublisherOptions(deps, canvas.OrganizationID),
	)
}

func applyIntakeSettingsToGraph(
	source string,
	graph intakeGraph,
	spec models.LiveCanvasSpec,
	settings *pb.FactoryIntake_Settings,
	nodes []models.Node,
	edges []models.Edge,
) ([]models.Node, []models.Edge, error) {
	current := intakeSettingsFromGraph(source, graph, spec)
	updated := parseIntakeSettings(current, settings)
	if intakeSourceHasFilterNode(source) &&
		intakeSettingsChangeTrigger(source, current, updated) &&
		graph.TriggerNodeID == "" {
		return nil, nil, invalidArgument("intake automation has no trigger to update")
	}
	expression := intakeFilterExpressionFor(source, updated)
	if graph.FilterNodeID == "" &&
		intakeSourceHasFilterNode(source) &&
		(intakeSettingsChangeFilters(current, updated) || expression != "true") {
		var err error
		nodes, edges, graph, err = ensureIntakeFilterNode(nodes, edges, graph)
		if err != nil {
			return nil, nil, invalidArgument(err.Error())
		}
	}
	for i := range nodes {
		switch nodes[i].ID {
		case graph.TriggerNodeID:
			configuration := maps.Clone(nodes[i].Configuration)
			if configuration == nil {
				configuration = map[string]any{}
			}
			switch source {
			case models.FactoryIntakeSourceGitHubIssues:
				configuration["actions"] = intakeTriggerActionsFor(updated)
				nodes[i].Configuration = configuration
			case models.FactoryIntakeSourceSentryExceptions:
				configuration["actions"] = intakeSentryActionsFor(updated)
				nodes[i].Configuration = configuration
			case models.FactoryIntakeSourceJiraIssues:
				configuration["events"] = intakeTriggerEventsFor(updated)
				nodes[i].Configuration = configuration
				nodes[i].Metadata = mergeJiraCompletionMetadata(nodes[i].Metadata, updated)
			default:
				continue
			}
		case graph.FilterNodeID:
			configuration := maps.Clone(nodes[i].Configuration)
			if configuration == nil {
				configuration = map[string]any{}
			}
			configuration["expression"] = expression
			nodes[i].Configuration = configuration
		}
	}

	if source == models.FactoryIntakeSourceGitHubIssues {
		var err error
		nodes, edges, err = configureIntakeAuthorAccess(nodes, edges, graph, updated.AuthorsWithAccess)
		if err != nil {
			return nil, nil, invalidArgument(err.Error())
		}
	}

	return nodes, edges, nil
}

func applyIntakeBindingToGraph(
	graph intakeGraph,
	binding *intakeBinding,
	nodes []models.Node,
) ([]models.Node, error) {
	if graph.TriggerNodeID == "" {
		return nil, invalidArgument("intake automation has no trigger to update")
	}

	boundID := binding.integrationRef().ID
	for i := range nodes {
		if nodes[i].ID != graph.TriggerNodeID {
			continue
		}
		configuration := maps.Clone(nodes[i].Configuration)
		if configuration == nil {
			configuration = map[string]any{}
		}
		for name, value := range binding.configuration() {
			configuration[name] = value
		}
		nodes[i].Configuration = configuration
		nodes[i].IntegrationID = &boundID
	}

	return nodes, nil
}

func intakeGraphUpdateMessage(settings *pb.FactoryIntake_Settings, binding *intakeBinding) string {
	if binding != nil && settings == nil {
		return "Update intake connection"
	}
	if binding != nil {
		return "Update intake"
	}
	return "Update intake settings"
}

func intakePublisherOptions(deps IntakeDependencies, orgID uuid.UUID) changesets.CanvasPublisherOptions {
	return changesets.CanvasPublisherOptions{
		Registry:       deps.Registry,
		OrgID:          orgID,
		Encryptor:      deps.Encryptor,
		AuthService:    deps.AuthService,
		WebhookBaseURL: deps.WebhookBaseURL,
	}
}
