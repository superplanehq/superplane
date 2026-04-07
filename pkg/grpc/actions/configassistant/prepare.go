package configassistant

import (
	"context"
	"errors"
	"strings"
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

const (
	configAssistantSuggestPath = "/config-assistant/suggest"
	scopedTokenTTL             = 15 * time.Minute
	configAssistantJWTPurpose  = "config-assistant"
)

// BuildConfigAssistantSuggestURL returns the full POST URL for the agent config-assistant suggest route.
func BuildConfigAssistantSuggestURL(agentHTTPBaseURL string) string {
	base := strings.TrimSpace(agentHTTPBaseURL)
	if base == "" {
		return ""
	}

	return strings.TrimRight(base, "/") + configAssistantSuggestPath
}

// PrepareConfigAssistantSuggest validates access, mints a scoped JWT, and returns the browser-callable suggest URL.
func PrepareConfigAssistantSuggest(
	ctx context.Context,
	authService authorization.Authorization,
	jwtSigner *jwt.Signer,
	agentHTTPBaseURL string,
	userID string,
	organizationID string,
	req *pb.PrepareConfigAssistantSuggestRequest,
) (*pb.PrepareConfigAssistantSuggestResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	baseURL := strings.TrimSpace(agentHTTPBaseURL)
	if baseURL == "" {
		return nil, status.Error(codes.Unavailable, "agent HTTP URL not configured")
	}

	suggestURL := BuildConfigAssistantSuggestURL(baseURL)
	if suggestURL == "" {
		return nil, status.Error(codes.Unavailable, "agent HTTP URL not configured")
	}

	canvasID := strings.TrimSpace(req.GetCanvasId())
	if canvasID == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas_id is required")
	}

	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid organization")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid canvas ID")
	}

	_, err = models.FindCanvas(orgUUID, canvasUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas not found")
		}

		return nil, status.Error(codes.Internal, "failed to load canvas")
	}

	permissions, err := allowedConfigAssistantJWTScopes(authService, userID, organizationID, canvasID)
	if err != nil {
		return nil, err
	}

	claims := jwt.ScopedTokenClaims{
		Subject: userID,
		OrgID:   organizationID,
		Purpose: configAssistantJWTPurpose,
		Scopes:  jwt.ScopesFromPermissions(permissions),
	}

	token, err := jwtSigner.GenerateScopedToken(claims, scopedTokenTTL)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to mint config assistant token")
	}

	return &pb.PrepareConfigAssistantSuggestResponse{
		Token:      token,
		SuggestUrl: suggestURL,
	}, nil
}

func allowedConfigAssistantJWTScopes(
	authService authorization.Authorization,
	userID, orgID, canvasID string,
) ([]jwt.Permission, error) {
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
			return nil, status.Error(codes.Internal, "failed to resolve config assistant permissions")
		}

		if !allowed {
			return nil, status.Error(codes.PermissionDenied, "config assistant is not allowed")
		}
	}

	return permissions, nil
}
