package materialize

import (
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

// protoMaterializationStatus maps the model's string status to the protobuf enum
// used on the wire by the RepositoryBranchUpdated message.
func protoMaterializationStatus(status string) pb.MaterializationStatus {
	switch status {
	case models.MaterializationStatusPending:
		return pb.MaterializationStatus_MATERIALIZATION_STATUS_PENDING
	case models.MaterializationStatusReady:
		return pb.MaterializationStatus_MATERIALIZATION_STATUS_READY
	case models.MaterializationStatusError:
		return pb.MaterializationStatus_MATERIALIZATION_STATUS_ERROR
	case models.MaterializationStatusDeleted:
		return pb.MaterializationStatus_MATERIALIZATION_STATUS_DELETED
	default:
		return pb.MaterializationStatus_MATERIALIZATION_STATUS_UNSPECIFIED
	}
}
