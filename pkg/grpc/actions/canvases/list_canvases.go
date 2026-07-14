package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListCanvases(
	ctx context.Context,
	registry *registry.Registry,
	organizationID string,
	userID string,
) (*pb.ListCanvasesResponse, error) {
	organizationUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid organization id")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid user id")
	}

	canvases, err := models.ListCanvases(organizationID)
	if err != nil {
		log.Errorf("failed to list canvases for organization %s: %v", organizationID, err)
		return nil, grpcerrors.Internal(err, "failed to list canvases")
	}

	protoCanvases, err := serializeCanvasSummaries(database.DB(ctx), organizationUUID, userUUID, canvases)
	if err != nil {
		log.Errorf("failed to serialize canvases for organization %s: %v", organizationID, err)
		return nil, grpcerrors.Internal(err, "failed to serialize canvases")
	}

	return &pb.ListCanvasesResponse{
		Canvases: protoCanvases,
	}, nil
}

func serializeCanvasSummaries(
	db *gorm.DB,
	organizationID uuid.UUID,
	userID uuid.UUID,
	canvases []models.Canvas,
) ([]*pb.CanvasSummary, error) {
	//
	// Get all users with a single query, to avoid N+1 queries.
	//
	userIDs := []uuid.UUID{}
	for _, canvas := range canvases {
		if canvas.CreatedBy != nil {
			userIDs = append(userIDs, *canvas.CreatedBy)
		}
	}

	users, err := models.FindMaybeDeletedUsersByIDs(db, userIDs)
	if err != nil {
		return nil, err
	}

	usersByID := make(map[string]models.User, len(users))
	for _, user := range users {
		usersByID[user.ID.String()] = user
	}

	canvasIDs := make([]uuid.UUID, len(canvases))
	for i, canvas := range canvases {
		canvasIDs[i] = canvas.ID
	}

	liveSpecs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, canvasIDs)
	if err != nil {
		return nil, err
	}

	preferencesByCanvasID, err := models.FindUserCanvasPreferencesForCanvases(db, organizationID, userID, canvasIDs)
	if err != nil {
		return nil, err
	}

	//
	// Serialize all canvases now
	//
	protoCanvases := make([]*pb.CanvasSummary, len(canvases))
	for i, canvas := range canvases {
		var user *models.User
		if canvas.CreatedBy != nil {
			u, _ := usersByID[canvas.CreatedBy.String()]
			user = &u
		}

		liveSpec := liveSpecs[canvas.ID]
		preference := preferencesByCanvasID[canvas.ID]

		protoCanvases[i] = &pb.CanvasSummary{
			Id:                        canvas.ID.String(),
			Name:                      canvas.Name,
			Description:               canvas.Description,
			CreatedAt:                 timestamppb.New(*canvas.CreatedAt),
			UpdatedAt:                 timestamppb.New(*canvas.UpdatedAt),
			Edges:                     actions.EdgesToProto(liveSpec.Edges),
			Nodes:                     []*pb.CanvasSummary_Node{},
			Pinned:                    preference.PinnedAt != nil,
			Starred:                   preference.StarredAt != nil,
			AutoLayoutOnUpdateEnabled: preference.AutoLayoutOnUpdateEnabled,
		}

		if preference.PinnedAt != nil {
			protoCanvases[i].PinnedAt = timestamppb.New(*preference.PinnedAt)
		}

		if preference.StarredAt != nil {
			protoCanvases[i].StarredAt = timestamppb.New(*preference.StarredAt)
		}

		if user != nil {
			protoCanvases[i].CreatedBy = &pb.UserRef{Id: user.ID.String(), Name: user.Name}
		}

		for _, node := range liveSpec.Nodes {
			protoCanvases[i].Nodes = append(protoCanvases[i].Nodes, &pb.CanvasSummary_Node{
				Id:       node.ID,
				Position: &componentpb.Position{X: int32(node.Position.X), Y: int32(node.Position.Y)},
			})
		}

		if canvas.CanvasFolderID != nil {
			protoCanvases[i].FolderId = canvas.CanvasFolderID.String()
		}
	}

	return protoCanvases, nil
}
