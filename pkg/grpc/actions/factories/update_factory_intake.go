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

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	intakeID, err := parseIntakeID(req.GetIntakeId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
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

	if req.Settings != nil {
		if err := applyIntakeSettings(ctx, deps, db, intake, canvas, req.GetSettings()); err != nil {
			return nil, err
		}
	}

	if req.Name != nil {
		name := strings.TrimSpace(req.GetName())
		if name == "" {
			return nil, factoryErrorToStatus(invalidArgument("intake name cannot be empty"), "failed to update factory intake")
		}
		if _, err := canvases.UpdateCanvas(ctx, db, canvas, &name, nil, nil); err != nil {
			return nil, err
		}
	}

	intake, err = factory.FindIntake(db, intakeID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, []uuid.UUID{intake.CanvasID})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory intake")
	}

	return &pb.UpdateFactoryIntakeResponse{
		Intake: serializeFactoryIntake(intake, specs[intake.CanvasID]),
	}, nil
}

// applyIntakeSettings writes the settings back into the canvas graph and makes
// the result live. The graph is the only place the workers read from, so a
// setting that is not in the graph has no effect.
func applyIntakeSettings(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	intake *models.FactoryIntake,
	canvas *models.Canvas,
	settings *pb.FactoryIntake_Settings,
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
		graph := resolveIntakeGraph(intake.Source, spec)
		if graph.ThresholdNodeID == "" {
			return invalidArgument("intake automation has no confidence threshold to update")
		}

		updated := parseIntakeSettings(intakeSettingsFromGraph(graph, spec), settings)
		expression := intakeThresholdExpressionFor(intake.Source, updated)

		nodes := slices.Clone(liveVersion.Nodes)
		for i := range nodes {
			if nodes[i].ID != graph.ThresholdNodeID {
				continue
			}
			// Copy the configuration so the edit does not reach into the live
			// version's map, which the publisher still compares against.
			configuration := maps.Clone(nodes[i].Configuration)
			if configuration == nil {
				configuration = map[string]any{}
			}
			configuration["expression"] = expression
			nodes[i].Configuration = configuration
		}

		return canvases.PublishGeneratedCanvasNodes(
			ctx,
			tx,
			canvas,
			uuid.MustParse(userID),
			"Update intake settings",
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
		)
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return err
		}
		return factoryErrorToStatus(err, "failed to update factory intake")
	}

	return nil
}
