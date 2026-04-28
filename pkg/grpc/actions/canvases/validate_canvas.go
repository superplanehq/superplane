package canvases

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func ValidateCanvas(
	ctx context.Context,
	reg *registry.Registry,
	organizationID uuid.UUID,
	pbCanvas *pb.Canvas,
) (*pb.ValidateCanvasResponse, error) {
	_, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if pbCanvas == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas is required")
	}

	if strings.TrimSpace(pbCanvas.GetMetadata().GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas name is required")
	}

	nodes, edges, err := ParseCanvas(reg, organizationID.String(), pbCanvas)
	if err != nil {
		return nil, err
	}

	version := &models.CanvasVersion{
		Nodes: datatypes.NewJSONSlice([]models.Node{}),
		Edges: datatypes.NewJSONSlice([]models.Edge{}),
	}

	if len(nodes) > 0 {
		changeset, err := changesets.NewChangesetBuilder([]models.Node{}, []models.Edge{}, nodes, edges).Build()
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to build changeset: %v", err)
		}

		patcher := changesets.NewCanvasPatcher(database.Conn(), organizationID, reg, version)
		if err := patcher.ApplyChangeset(changeset, nil); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "canvas is invalid: %v", err)
		}

		version = patcher.GetVersion()
	}

	return &pb.ValidateCanvasResponse{
		Version: SerializeCanvasVersion(version, organizationID.String()),
	}, nil
}
