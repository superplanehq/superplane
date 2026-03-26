package agents

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func CreateAgentChat(
	authService authorization.Authorization,
	jwtSigner *jwt.Signer,
	agentURL string,
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
		Url:   fmt.Sprintf("%s/v1/agent/chat/stream", agentURL),
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
