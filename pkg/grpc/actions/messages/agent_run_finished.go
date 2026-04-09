package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

const (
	AgentExchange              = "superplane.agent-exchange"
	AgentRunFinishedRoutingKey = "agent-run-finished"
)

type AgentRunFinishedMessage struct {
	message *pb.AgentRunFinishedMessage
}

func NewAgentRunFinishedMessage(organizationID, chatID, model string, inputTokens, outputTokens, totalTokens int64) AgentRunFinishedMessage {
	return AgentRunFinishedMessage{
		message: &pb.AgentRunFinishedMessage{
			OrganizationId: organizationID,
			ChatId:         chatID,
			Model:          model,
			InputTokens:    inputTokens,
			OutputTokens:   outputTokens,
			TotalTokens:    totalTokens,
		},
	}
}

func (m AgentRunFinishedMessage) Publish() error {
	return Publish(AgentExchange, AgentRunFinishedRoutingKey, toBytes(m.message))
}
