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

	liveVersionID := ""
	if canvas.LiveVersionID != nil {
		liveVersionID = canvas.LiveVersionID.String()
	}

	factoryID := ""
	if canvas.FactoryID != nil {
		factoryID = canvas.FactoryID.String()
	}

	return &pb.Canvas{
		Metadata: &pb.Canvas_Metadata{
			Id:                          canvas.ID.String(),
			OrganizationId:              canvas.OrganizationID.String(),
			Name:                        canvas.Name,
			Description:                 canvas.Description,
			CreatedAt:                   timestamppb.New(*canvas.CreatedAt),
			UpdatedAt:                   timestamppb.New(*canvas.UpdatedAt),
			CreatedBy:                   createdBy,
			FolderId:                    canvasFolderID,
			LiveVersionId:               liveVersionID,
			FactoryId:                   factoryID,
			DismissedAgentSuggestionIds: append([]string(nil), canvas.DismissedAgentSuggestionIDs...),
		},
		Spec: &pb.Canvas_Spec{
			Nodes:      actions.NodesToProto(liveVersion.Nodes),
			Edges:      actions.EdgesToProto(liveVersion.Edges),
			NodeGroups: NodeGroupsToProto(liveVersion.NodeGroups),
		},
		Status: status,
	}, nil
}

func NodeGroupsToProto(groups []models.NodeGroup) []*pb.NodeGroup {
	result := make([]*pb.NodeGroup, len(groups))
	for i, group := range groups {
		result[i] = &pb.NodeGroup{
			Id:    group.ID,
			Nodes: append([]string(nil), group.Nodes...),
		}

		if group.Max != nil {
			max := int32(*group.Max)
			result[i].Max = &max
		}
	}
	return result
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
