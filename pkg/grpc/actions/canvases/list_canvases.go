package canvases

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListCanvases(ctx context.Context, registry *registry.Registry, organizationID string) (*pb.ListCanvasesResponse, error) {
	canvases, err := models.ListCanvases(organizationID)
	if err != nil {
		log.Errorf("failed to list canvases for organization %s: %v", organizationID, err)
		return nil, status.Error(codes.Internal, "failed to list canvases")
	}

	protoCanvases, err := serializeCanvasSummaries(canvases)
	if err != nil {
		log.Errorf("failed to serialize canvases for organization %s: %v", organizationID, err)
		return nil, status.Error(codes.Internal, "failed to serialize canvases")
	}

	return &pb.ListCanvasesResponse{
		Canvases: protoCanvases,
	}, nil
}

func serializeCanvasSummaries(canvases []models.Canvas) ([]*pb.CanvasSummary, error) {
	//
	// Get all users with a single query, to avoid N+1 queries.
	//
	userIDs := []uuid.UUID{}
	for _, canvas := range canvases {
		if canvas.CreatedBy != nil {
			userIDs = append(userIDs, *canvas.CreatedBy)
		}
	}

	users, err := models.FindMaybeDeletedUsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	usersByID := make(map[string]models.User, len(users))
	for _, user := range users {
		usersByID[user.ID.String()] = user
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

		protoCanvases[i] = &pb.CanvasSummary{
			Id:          canvas.ID.String(),
			Name:        canvas.Name,
			Description: canvas.Description,
			CreatedAt:   timestamppb.New(*canvas.CreatedAt),
			UpdatedAt:   timestamppb.New(*canvas.UpdatedAt),
			CreatedBy:   &pb.UserRef{Id: canvas.CreatedBy.String(), Name: user.Name},
			Edges:       actions.EdgesToProto(canvas.Edges),
			Nodes:       []*pb.CanvasSummary_Node{},
		}

		for _, node := range canvas.Nodes {
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
