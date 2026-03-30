package agents

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func CreateAgentChat(
	ctx context.Context,
	authService authorization.Authorization,
	jwtSigner *jwt.Signer,
	agentInternalURL string,
	agentPublicURL string,
	userID string,
	organizationID string,
	canvasID string,
) (*pb.CreateAgentChatResponse, error) {
	org, err := uuid.Parse(organizationID)
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
	response, err := client.CreateAgentChat(ctx, &internalpb.CreateAgentChatRequest{
		OrgId:    organizationID,
		UserId:   userID,
		CanvasId: canvasID,
	})

	if err != nil {
		log.WithError(err).Errorf("failed to create agent chat for org %s, user %s, canvas %s", organizationID, userID, canvasID)
		return nil, status.Error(codes.Unavailable, "failed to create agent chat")
	}

	if response.Chat == nil {
		return nil, status.Error(codes.NotFound, "agent chat not found")
	}

	permissions, err := allowedAgentChatPermissions(authService, userID, organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	claims := jwt.ScopedTokenClaims{
		Subject: userID,
		OrgID:   organizationID,
		Purpose: "agent-builder",
		Scopes:  jwt.ScopesFromPermissions(permissions),
	}

	token, err := jwtSigner.GenerateScopedToken(claims, 15*time.Minute)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to mint agent chat session")
	}

	return &pb.CreateAgentChatResponse{
		Token: token,
		Url:   BuildAgentChatStreamURL(agentPublicURL, response.Chat.Id),
	}, nil
}

func allowedAgentChatPermissions(authService authorization.Authorization, userID, orgID, canvasID string) ([]jwt.Permission, error) {
	permissions := []jwt.Permission{
		{ResourceType: "org", Action: "read"},
		{ResourceType: "integrations", Action: "read"},
		{ResourceType: "canvases", Action: "read", Resources: []string{canvasID}},
	}

	for _, permission := range permissions {
		allowed, err := authService.CheckOrganizationPermission(
			userID,
			orgID,
			permission.ResourceType,
			permission.Action,
		)

		if err != nil {
			return nil, status.Error(codes.Internal, "failed to resolve agent permissions")
		}

		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "agent chat is not allowed")
		}
	}

	return permissions, nil
}

func BuildAgentChatStreamURL(publicURL string, chatID string) string {
	return strings.TrimRight(publicURL, "/") + "/agents/chats/" + chatID + "/stream"
}
