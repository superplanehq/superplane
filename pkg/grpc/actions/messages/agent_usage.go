package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
)

const AgentsExchange = "superplane.agents-exchange"

const AgentRunFinishedRoutingKey = "agent.run.finished"

type AgentRunFinishedMessage struct {
	message *pb.AgentRunFinishedMessage
}

func NewAgentRunFinishedMessage(
	organizationID, chatID, model, usageID, sessionID string,
	inputTokens, outputTokens, totalTokens, cacheReadTokens, cacheWriteTokens int64,
) AgentRunFinishedMessage {
	return AgentRunFinishedMessage{
		message: &pb.AgentRunFinishedMessage{
			OrganizationId:   organizationID,
			ChatId:           chatID,
			Model:            model,
			InputTokens:      inputTokens,
			OutputTokens:     outputTokens,
			TotalTokens:      totalTokens,
			CacheReadTokens:  cacheReadTokens,
			CacheWriteTokens: cacheWriteTokens,
			UsageId:          usageID,
			SessionId:        sessionID,
		},
	}
}

func (m AgentRunFinishedMessage) Publish() error {
	return Publish(AgentsExchange, AgentRunFinishedRoutingKey, toBytes(m.message))
}
