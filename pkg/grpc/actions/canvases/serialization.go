package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeCanvas(
	canvas *models.Canvas,
	liveVersion *models.CanvasVersion,
	user *models.User,
	status *pb.Canvas_Status,
) (*pb.Canvas, error) {
	var createdBy *pb.UserRef
	if user != nil {
		createdBy = &pb.UserRef{Id: user.ID.String(), Name: user.Name}
	}

	canvasFolderID := ""
	if canvas.CanvasFolderID != nil {
		canvasFolderID = canvas.CanvasFolderID.String()
	}

	var createdAt, updatedAt *timestamppb.Timestamp
	if canvas.CreatedAt != nil {
		createdAt = timestamppb.New(*canvas.CreatedAt)
	}
	if canvas.UpdatedAt != nil {
		updatedAt = timestamppb.New(*canvas.UpdatedAt)
	}

	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Id:             canvas.ID.String(),
			OrganizationId: canvas.OrganizationID.String(),
			Name:           canvas.Name,
			Description:    canvas.Description,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			CreatedBy:      createdBy,
			FolderId:       canvasFolderID,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: actions.NodesToProto(liveVersion.Nodes),
			Edges: actions.EdgesToProto(liveVersion.Edges),
		},
		Status: status,
	}, nil
}

func serializeCanvas(
	ctx context.Context,
	canvas *models.Canvas,
	liveVersion *models.CanvasVersion,
	user *models.User,
	status *pb.Canvas_Status,
) (proto *pb.Canvas, err error) {
	ctx, done := telemetry.Span(ctx, "canvases.serialize")
	defer done(&err)

	return SerializeCanvas(canvas, liveVersion, user, status)
}
