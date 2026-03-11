package canvases

import (
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ensureCanvasChangeRequestReadyToPublish(
	canvas *models.Canvas,
	approvals []models.CanvasChangeRequestApproval,
) error {
	approvers := canvas.EffectiveChangeRequestApprovers()
	activeByIndex := activeApprovalsByIndex(approvals)

	for index := range approvers {
		activeApproval := activeByIndex[index]
		if activeApproval == nil {
			return status.Error(codes.FailedPrecondition, "change request does not have all required approvals")
		}
		if activeApproval.State != models.CanvasChangeRequestApprovalStateApproved {
			return status.Error(codes.FailedPrecondition, "change request has pending or rejected approvals")
		}
	}

	return nil
}
