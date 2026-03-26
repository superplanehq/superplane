package agents

import (
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/jwt"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const streamTokenLifetime = 15 * time.Minute

func MintCreateAgentChatStreamResponse(
	jwtSigner *jwt.Signer,
	publicURL string,
	userID string,
	orgID string,
	canvasID string,
	agentID string,
) (*pb.CreateAgentChatResponse, error) {
	token, err := mintAgentStreamToken(jwtSigner, userID, orgID, canvasID, agentID)
	if err != nil {
		return nil, err
	}

	return &pb.CreateAgentChatResponse{
		Token: token,
		Url:   buildAgentChatStreamURL(publicURL, agentID),
	}, nil
}

func MintResumeAgentChatStreamResponse(
	jwtSigner *jwt.Signer,
	publicURL string,
	userID string,
	orgID string,
	canvasID string,
	agentID string,
) (*pb.ResumeAgentChatResponse, error) {
	token, err := mintAgentStreamToken(jwtSigner, userID, orgID, canvasID, agentID)
	if err != nil {
		return nil, err
	}

	return &pb.ResumeAgentChatResponse{
		Token: token,
		Url:   buildAgentChatStreamURL(publicURL, agentID),
	}, nil
}

func buildAgentChatStreamURL(publicURL string, agentID string) string {
	return strings.TrimRight(publicURL, "/") + "/agents/chats/" + agentID + "/stream"
}

func mintAgentStreamToken(
	jwtSigner *jwt.Signer,
	userID string,
	orgID string,
	canvasID string,
	agentID string,
) (string, error) {
	claims := jwt.ScopedTokenClaims{
		Subject: userID,
		OrgID:   orgID,
		Purpose: "agent-builder",
		AgentID: agentID,
		Scopes: jwt.ScopesFromPermissions([]jwt.Permission{
			{ResourceType: "org", Action: "read"},
			{ResourceType: "integrations", Action: "read"},
			{ResourceType: "canvases", Action: "read", Resources: []string{canvasID}},
		}),
	}

	token, err := jwtSigner.GenerateScopedToken(claims, streamTokenLifetime)
	if err != nil {
		return "", status.Error(codes.Internal, "failed to mint agent stream token")
	}

	return token, nil
}

func SerializeAgentChatInfo(agent *internalpb.ChatInfo) (*pb.AgentChatInfo, error) {
	if agent == nil {
		return nil, status.Error(codes.Internal, "missing agent chat")
	}

	return &pb.AgentChatInfo{
		Id:             agent.Id,
		InitialMessage: agent.InitialMessage,
		CreatedAt:      timestampOrNil(agent.CreatedAt),
	}, nil
}

func SerializeAgentChatMessages(messages []*internalpb.AgentChatMessage) []*pb.AgentChatMessage {
	serialized := make([]*pb.AgentChatMessage, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}

		serialized = append(serialized, &pb.AgentChatMessage{
			Id:         message.Id,
			Role:       message.Role,
			Content:    message.Content,
			ToolCallId: message.ToolCallId,
			ToolStatus: message.ToolStatus,
			CreatedAt:  timestampOrNil(message.CreatedAt),
		})
	}

	return serialized
}

func timestampOrNil(ts *timestamppb.Timestamp) *timestamppb.Timestamp {
	if ts == nil {
		return nil
	}

	return timestamppb.New(ts.AsTime())
}
