package configassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/config_assistant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	maxInstructionRunes       = 2000
	maxFieldContextJSONBytes  = 100 * 1024
	configAssistantHTTPPath   = "/config-assistant/suggest"
	suggestHTTPClientTimeout  = 120 * time.Second
	scopedTokenTTL            = 15 * time.Minute
	configAssistantJWTPurpose = "config-assistant"
)

type agentSuggestResponse struct {
	Value       string `json:"value"`
	Explanation string `json:"explanation,omitempty"`
}

// SuggestConfigurationField forwards the request to the Python agent HTTP service with a scoped JWT.
func SuggestConfigurationField(
	ctx context.Context,
	authService authorization.Authorization,
	jwtSigner *jwt.Signer,
	configAssistantBaseURL string,
	userID string,
	organizationID string,
	req *pb.SuggestConfigurationFieldRequest,
) (*pb.SuggestConfigurationFieldResponse, error) {
	baseURL := strings.TrimSpace(configAssistantBaseURL)
	if baseURL == "" {
		return nil, status.Error(codes.Unavailable, "CONFIG_ASSISTANT_HTTP_URL is not set")
	}

	canvasID := strings.TrimSpace(req.GetCanvasId())
	if canvasID == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas_id is required")
	}

	nodeID := strings.TrimSpace(req.GetNodeId())
	if nodeID == "" {
		return nil, status.Error(codes.InvalidArgument, "node_id is required")
	}

	instruction := strings.TrimSpace(req.GetInstruction())
	if instruction == "" {
		return nil, status.Error(codes.InvalidArgument, "instruction is required")
	}

	if utf8.RuneCountInString(instruction) > maxInstructionRunes {
		return nil, status.Error(codes.InvalidArgument, "instruction is too long")
	}

	fieldContextJSON := req.GetFieldContextJson()
	if len(fieldContextJSON) > maxFieldContextJSONBytes {
		return nil, status.Error(codes.InvalidArgument, "field_context_json is too large")
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

	payload := map[string]string{
		"canvas_id":          canvasID,
		"node_id":            nodeID,
		"instruction":        instruction,
		"field_context_json": fieldContextJSON,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode request")
	}

	endpoint := strings.TrimRight(baseURL, "/") + configAssistantHTTPPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to build agent request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: suggestHTTPClientTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.WithError(err).Errorf("config assistant HTTP request failed for org %s canvas %s", organizationID, canvasID)
		return nil, status.Error(codes.Unavailable, "config assistant request failed")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to read config assistant response")
	}

	if resp.StatusCode != http.StatusOK {
		log.Warnf("config assistant returned %d: %s", resp.StatusCode, truncateForLog(string(respBody)))
		return nil, status.Error(codes.Internal, "config assistant request rejected")
	}

	var parsed agentSuggestResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, status.Error(codes.Internal, "invalid config assistant response")
	}

	out := strings.TrimSpace(parsed.Value)
	if out == "" {
		return nil, status.Error(codes.Internal, "config assistant returned an empty value")
	}

	return &pb.SuggestConfigurationFieldResponse{
		Value:       out,
		Explanation: strings.TrimSpace(parsed.Explanation),
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

func truncateForLog(s string) string {
	const max = 500
	if len(s) <= max {
		return s
	}

	return s[:max] + "…"
}
