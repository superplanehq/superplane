package canvases

import (
	"context"

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

	versionID := ""
	if canvas.LiveVersionID != nil {
		versionID = canvas.LiveVersionID.String()
	}

	spec, err := SerializeCanvasSpecFromVersion(liveVersion)
	if err != nil {
		return nil, err
	}

	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Id:             canvas.ID.String(),
			OrganizationId: canvas.OrganizationID.String(),
			Name:           canvas.Name,
			Description:    canvas.Description,
			CreatedAt:      timestamppb.New(*canvas.CreatedAt),
			UpdatedAt:      timestamppb.New(*canvas.UpdatedAt),
			CreatedBy:      createdBy,
			FolderId:       canvasFolderID,
			VersionId:      versionID,
		},
		Spec:   spec,
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
