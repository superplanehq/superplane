package agents

import (
	"context"
	"errors"
	"time"

	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ResumeAgentChat(
	ctx context.Context,
	authService authorization.Authorization,
	jwtSigner *jwt.Signer,
	agentInternalURL string,
	agentPublicURL string,
	orgID string,
	userID string,
	canvasID string,
	chatID string,
) (*pb.ResumeAgentChatResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid organization")
	}

	canvas, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	_, err = models.FindCanvas(org, canvas)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}

		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	conn, err := grpc.NewClient(agentInternalURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to create agent GRPC client")
	}

	client := internalpb.NewAgentsClient(conn)
	response, err := client.DescribeAgentChat(ctx, &internalpb.DescribeAgentChatRequest{
		OrgId:    orgID,
		UserId:   userID,
		CanvasId: canvasID,
		ChatId:   chatID,
	})

	if err != nil || response.Chat == nil {
		return nil, status.Error(codes.Unavailable, "failed to describe agent chat")
	}

	permissions, err := allowedAgentChatPermissions(authService, userID, orgID, canvasID)
	if err != nil {
		return nil, err
	}

	claims := jwt.ScopedTokenClaims{
		Subject: userID,
		OrgID:   orgID,
		Purpose: "agent-builder",
		Scopes:  jwt.ScopesFromPermissions(permissions),
	}

	token, err := jwtSigner.GenerateScopedToken(claims, 15*time.Minute)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to mint agent chat session")
	}

	return &pb.ResumeAgentChatResponse{
		Token: token,
		Url:   BuildAgentChatStreamURL(agentPublicURL, chatID),
	}, nil
}
