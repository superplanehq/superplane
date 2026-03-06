package canvases

import (
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SerializeCanvasVersion(version *models.CanvasVersion, organizationID string) *pb.CanvasVersion {
	var owner *pb.UserRef
	if version.OwnerID != nil {
		ownerID := version.OwnerID.String()
		ownerName := ""
		if user, err := models.FindMaybeDeletedUserByID(organizationID, ownerID); err == nil && user != nil {
			ownerName = user.Name
		}
		owner = &pb.UserRef{Id: ownerID, Name: ownerName}
	}

	metadata := &pb.CanvasVersion_Metadata{
		Id:          version.ID.String(),
		CanvasId:    version.WorkflowID.String(),
		Owner:       owner,
		IsPublished: version.IsPublished,
	}

	if version.PublishedAt != nil {
		metadata.PublishedAt = timestamppb.New(*version.PublishedAt)
	}
	if version.CreatedAt != nil {
		metadata.CreatedAt = timestamppb.New(*version.CreatedAt)
	}
	if version.UpdatedAt != nil {
		metadata.UpdatedAt = timestamppb.New(*version.UpdatedAt)
	}

	return &pb.CanvasVersion{
		Metadata: metadata,
		Spec: &pb.Canvas_Spec{
			Nodes: actions.NodesToProto(version.Nodes),
			Edges: actions.EdgesToProto(version.Edges),
		},
	}
}
