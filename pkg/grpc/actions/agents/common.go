package agents

import (
	"fmt"
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

func MintAgentStreamResponse(
	jwtSigner *jwt.Signer,
	publicURL string,
	userID string,
	orgID string,
	canvasID string,
	agentID string,
) (*pb.CreateAgentResponse, error) {
	token, err := mintAgentStreamToken(jwtSigner, userID, orgID, canvasID, agentID)
	if err != nil {
		return nil, err
	}

	return &pb.CreateAgentResponse{
		Token: token,
		Url:   buildAgentStreamURL(publicURL, agentID),
	}, nil
}

func MintResumeAgentStreamResponse(
	jwtSigner *jwt.Signer,
	publicURL string,
	userID string,
	orgID string,
	canvasID string,
	agentID string,
) (*pb.ResumeAgentResponse, error) {
	token, err := mintAgentStreamToken(jwtSigner, userID, orgID, canvasID, agentID)
	if err != nil {
		return nil, err
	}

	return &pb.ResumeAgentResponse{
		Token: token,
		Url:   buildAgentStreamURL(publicURL, agentID),
	}, nil
}

func buildAgentStreamURL(publicURL string, agentID string) string {
	return strings.TrimRight(publicURL, "/") + "/agents/" + agentID + "/stream"
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

func SerializeAgentInfo(agent *internalpb.AgentInfo) (*pb.AgentInfo, error) {
	if agent == nil {
		return nil, status.Error(codes.Internal, "missing agent")
	}

	return &pb.AgentInfo{
		Id:             agent.Id,
		InitialMessage: agent.InitialMessage,
		CreatedAt:      timestampOrNil(agent.CreatedAt),
	}, nil
}

func SerializeAgentMessages(messages []*internalpb.AgentMessage) []*pb.AgentMessage {
	serialized := make([]*pb.AgentMessage, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}

		serialized = append(serialized, &pb.AgentMessage{
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

func RequireAgentPublicURL(publicURL string) (string, error) {
	normalized := strings.TrimSpace(publicURL)
	if normalized == "" {
		return "", status.Error(codes.Internal, "agent public url is not configured")
	}

	if !strings.HasPrefix(normalized, "http://") && !strings.HasPrefix(normalized, "https://") {
		return "", status.Error(codes.Internal, fmt.Sprintf("invalid agent public url: %s", normalized))
	}

	return normalized, nil
}
