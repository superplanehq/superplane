package eventdistributer

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/public/ws"
)

// AgentSessionWebsocketTopic is the hub key shared by publisher and
// subscriber. Access control is enforced at subscribe time — we trust the
// topic on broadcast.
func AgentSessionWebsocketTopic(sessionID string) string {
	return "agent-session:" + sessionID
}

func HandleAgentSessionEvent(messageBody []byte, wsHub *ws.Hub) error {
	var msg messages.AgentSessionEventMessage
	if err := json.Unmarshal(messageBody, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal agent session event: %w", err)
	}
	if msg.SessionID == "" {
		return fmt.Errorf("missing session_id in agent session event")
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal agent session event: %w", err)
	}

	wsHub.BroadcastToWorkflow(AgentSessionWebsocketTopic(msg.SessionID), payload)
	log.Debugf("Broadcasted agent session event %s to session %s", msg.Event, msg.SessionID)
	return nil
}
