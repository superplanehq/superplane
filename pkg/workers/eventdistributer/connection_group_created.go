package eventdistributer

import (
	"context"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	groups "github.com/superplanehq/superplane/pkg/grpc/actions/connection_groups"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/public/ws"
	"google.golang.org/protobuf/proto"
)

func HandleConnectionGroupCreated(messageBody []byte, wsHub *ws.Hub) error {
	log.Debugf("Received connection_group_added event")

	pbMsg := &pb.ConnectionGroupCreated{}
	if err := proto.Unmarshal(messageBody, pbMsg); err != nil {
		return fmt.Errorf("failed to unmarshal ConnectionGroupCreated message: %w", err)
	}

	response, err := groups.DescribeConnectionGroup(context.Background(), &pb.DescribeConnectionGroupRequest{
		CanvasIdOrName: pbMsg.CanvasId,
		IdOrName:       pbMsg.ConnectionGroupId,
	})

	if err != nil {
		return fmt.Errorf("failed to describe connection group: %w", err)
	}

	wsEvent := map[string]interface{}{
		"event":   "connection_group_added",
		"payload": response.ConnectionGroup,
	}

	// Convert to JSON for websocket transmission
	wsEventJSON, err := json.Marshal(wsEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket event: %w", err)
	}

	// Send to all clients subscribed to this canvas
	wsHub.BroadcastToCanvas(pbMsg.CanvasId, wsEventJSON)
	log.Debugf("Broadcasted connection_group_added event to canvas %s", pbMsg.CanvasId)

	return nil
}
