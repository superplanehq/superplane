package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

const RepositoryBranchUpdatedEvent = "repository_branch_updated"

// materializationStatusTokens maps the proto enum to the lowercase tokens the UI
// consumes, keeping the websocket payload independent of the protobuf naming.
var materializationStatusTokens = map[pb.MaterializationStatus]string{
	pb.MaterializationStatus_MATERIALIZATION_STATUS_PENDING: "pending",
	pb.MaterializationStatus_MATERIALIZATION_STATUS_READY:   "ready",
	pb.MaterializationStatus_MATERIALIZATION_STATUS_ERROR:   "error",
	pb.MaterializationStatus_MATERIALIZATION_STATUS_DELETED: "deleted",
}

type RepositoryBranchUpdatedPayload struct {
	ID                    string `json:"id"`
	CanvasID              string `json:"canvasId"`
	Branch                string `json:"branch"`
	HeadSHA               string `json:"headSha"`
	MaterializationStatus string `json:"materializationStatus"`
	MaterializationError  string `json:"materializationError,omitempty"`
}

type RepositoryBranchUpdatedWebsocketEvent struct {
	Event   string                         `json:"event"`
	Payload RepositoryBranchUpdatedPayload `json:"payload"`
}

func HandleRepositoryBranchUpdated(messageBody []byte, wsHub *ws.Hub) error {
	pbMsg := &pb.RepositoryBranchUpdatedMessage{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal %s message: %w", RepositoryBranchUpdatedEvent, err)
	}

	if pbMsg.CanvasId == "" {
		return fmt.Errorf("missing canvas id in %s message", RepositoryBranchUpdatedEvent)
	}

	wsEvent, err := json.Marshal(RepositoryBranchUpdatedWebsocketEvent{
		Event: RepositoryBranchUpdatedEvent,
		Payload: RepositoryBranchUpdatedPayload{
			ID:                    pbMsg.CanvasId,
			CanvasID:              pbMsg.CanvasId,
			Branch:                pbMsg.Branch,
			HeadSHA:               pbMsg.HeadSha,
			MaterializationStatus: materializationStatusTokens[pbMsg.MaterializationStatus],
			MaterializationError:  pbMsg.MaterializationError,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal %s websocket event: %w", RepositoryBranchUpdatedEvent, err)
	}

	wsHub.BroadcastToWorkflow(pbMsg.CanvasId, wsEvent)
	log.Debugf("Broadcasted %s event to workflow %s", RepositoryBranchUpdatedEvent, pbMsg.CanvasId)

	return nil
}
