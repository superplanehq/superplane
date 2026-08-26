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

func UpdateFactoryPRFeedbackHandler(
	ctx context.Context,
	deps PRFeedbackDependencies,
	organizationID string,
	req *pb.UpdateFactoryPRFeedbackHandlerRequest,
) (*pb.UpdateFactoryPRFeedbackHandlerResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	handlerID, err := parsePRFeedbackHandlerID(req.GetHandlerId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	handler, err := factory.FindPRFeedbackHandler(db, handlerID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	canvas, err := models.FindCanvasInTransaction(db, orgID, handler.CanvasID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	if req.Settings != nil {
		if err := applyPRFeedbackSettings(ctx, deps, db, handler, canvas, req.GetSettings()); err != nil {
			return nil, err
		}
	}

	if req.Name != nil {
		name := strings.TrimSpace(req.GetName())
		if name == "" {
			return nil, factoryErrorToStatus(invalidArgument("handler name cannot be empty"), "failed to update factory PR feedback handler")
		}
		if _, err := canvases.UpdateCanvas(ctx, db, canvas, &name, nil, nil); err != nil {
			return nil, err
		}
	}

	handler, err = factory.FindPRFeedbackHandler(db, handlerID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, []uuid.UUID{handler.CanvasID})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	return &pb.UpdateFactoryPRFeedbackHandlerResponse{
		Handler: serializeFactoryPRFeedbackHandler(handler, specs[handler.CanvasID]),
	}, nil
}

func applyPRFeedbackSettings(
	ctx context.Context,
	deps PRFeedbackDependencies,
	db *gorm.DB,
	handler *models.FactoryPRFeedbackHandler,
	canvas *models.Canvas,
	settings *pb.FactoryPRFeedbackHandler_Settings,
) error {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if err != nil {
			return err
		}

		spec := models.LiveCanvasSpec{Nodes: liveVersion.Nodes, Edges: liveVersion.Edges}
		graph := resolvePRFeedbackGraph(spec)
		if strings.TrimSpace(settings.GetRepository()) == "" {
			return invalidArgument("repository cannot be empty")
		}
		if strings.TrimSpace(settings.GetMention()) == "" {
			return invalidArgument("mention cannot be empty")
		}
		updated := parsePRFeedbackSettings(prFeedbackSettingsFromGraph(graph, spec), settings)

		triggerIDs := map[string]bool{}
		for _, nodeID := range graph.triggerNodeIDs() {
			if nodeID != "" {
				triggerIDs[nodeID] = true
			}
		}
		if len(triggerIDs) == 0 {
			return invalidArgument("PR feedback automation has no GitHub triggers to update")
		}

		nodes := slices.Clone(liveVersion.Nodes)
		for i := range nodes {
			if !triggerIDs[nodes[i].ID] {
				continue
			}
			configuration := maps.Clone(nodes[i].Configuration)
			if configuration == nil {
				configuration = map[string]any{}
			}
			configuration["repository"] = updated.Repository
			configuration["contentFilter"] = updated.Mention
			configuration["ignoreBots"] = updated.IgnoreBots
			nodes[i].Configuration = configuration
		}

		if err := canvases.PublishGeneratedCanvasNodes(
			ctx,
			tx,
			canvas,
			uuid.MustParse(userID),
			"Update PR feedback settings",
			nodes,
			slices.Clone(liveVersion.Edges),
			changesets.CanvasPublisherOptions{
				Registry:       deps.Registry,
				OrgID:          canvas.OrganizationID,
				Encryptor:      deps.Encryptor,
				AuthService:    deps.AuthService,
				WebhookBaseURL: deps.WebhookBaseURL,
				GitProvider:    deps.GitProvider,
			},
		); err != nil {
			return err
		}

		return handler.Touch(tx)
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return err
		}
		return factoryErrorToStatus(err, "failed to update factory PR feedback handler")
	}

	return nil
}
