package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const RepositoryBranchUpdatedRoutingKey = "repository-branch-updated"

type RepositoryBranchUpdatedMessage struct {
	message *pb.RepositoryBranchUpdatedMessage
}

func NewRepositoryBranchUpdatedMessage(
	canvasID string,
	branch string,
	headSHA string,
	materializationStatus pb.MaterializationStatus,
	materializationError string,
	pushedByUserID string,
) RepositoryBranchUpdatedMessage {
	return RepositoryBranchUpdatedMessage{
		message: &pb.RepositoryBranchUpdatedMessage{
			CanvasId:              canvasID,
			Branch:                branch,
			HeadSha:               headSHA,
			MaterializationStatus: materializationStatus,
			MaterializationError:  materializationError,
			Timestamp:             timestamppb.Now(),
			PushedByUserId:        pushedByUserID,
		},
	}
}

func (m RepositoryBranchUpdatedMessage) PublishBranchUpdated() error {
	return Publish(CanvasExchange, RepositoryBranchUpdatedRoutingKey, toBytes(m.message))
}
