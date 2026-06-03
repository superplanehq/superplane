package canvases

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func serializeDraftBranch(
	branch *models.CanvasDraftBranch,
	organizationID string,
	state *models.RepositoryMaterializationState,
) *pb.CanvasDraftBranch {
	if branch == nil {
		return nil
	}

	proto := &pb.CanvasDraftBranch{
		BranchName:  branch.BranchName,
		DisplayName: branch.DisplayName,
		TipSha:      branch.TipSHA,
	}

	if branch.OwnerID != nil {
		ownerID := branch.OwnerID.String()
		ownerName := ""
		if user, err := models.FindMaybeDeletedUserByID(organizationID, ownerID); err == nil && user != nil {
			ownerName = user.Name
		}
		proto.Owner = &pb.UserRef{Id: ownerID, Name: ownerName}
	}

	if branch.CreatedBy != nil {
		createdByID := branch.CreatedBy.String()
		createdByName := ""
		if user, err := models.FindMaybeDeletedUserByID(organizationID, createdByID); err == nil && user != nil {
			createdByName = user.Name
		}
		proto.CreatedBy = &pb.UserRef{Id: createdByID, Name: createdByName}
	}

	if branch.CreatedAt != nil {
		proto.CreatedAt = timestamppb.New(*branch.CreatedAt)
	}
	if branch.UpdatedAt != nil {
		proto.UpdatedAt = timestamppb.New(*branch.UpdatedAt)
	}

	if state != nil {
		proto.MaterializationStatus = state.Status
		proto.MaterializationError = state.Error
	}

	return proto
}
