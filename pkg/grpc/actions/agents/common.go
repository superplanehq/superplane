// Package agents implements the gRPC handlers for the public Agents service.
package agents

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	agentservice "github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type AgentsService interface {
	EnsureSession(ctx context.Context, organizationID, userID, canvasID uuid.UUID) (*models.AgentSession, error)
	GetSession(organizationID, userID, sessionID uuid.UUID) (*models.AgentSession, error)
	ListMessages(sessionID, beforeID uuid.UUID, limit int) ([]models.AgentSessionMessage, error)
	SendMessage(ctx context.Context, organizationID, userID, sessionID uuid.UUID, content string, mode ...string) (*models.AgentSessionMessage, error)
	InterruptSession(ctx context.Context, organizationID, userID, sessionID uuid.UUID) error
	DefineOutcome(ctx context.Context, organizationID, userID, sessionID uuid.UUID, description, rubric string, maxIterations int) error
}

func agentModeFromProto(mode pb.AgentMode) string {
	switch mode {
	case pb.AgentMode_MODE_BUILDER:
		return string(agentservice.ModeBuilder)
	case pb.AgentMode_MODE_ARCHITECT:
		return string(agentservice.ModeOperator)
	default:
		return string(agentservice.ModeOperator)
	}
}

func parseOrgUser(orgID, userID string) (org, user uuid.UUID, err error) {
	org, err = uuid.Parse(orgID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.Internal, "invalid organization")
	}
	user, err = uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.Internal, "invalid user")
	}
	return org, user, nil
}

// translateAgentServiceError maps errors returned by the agent service to
// appropriate gRPC status codes. We use this for /agents/chats/* mutations so
// that transient/upstream failures (request cancelled, Anthropic 429/5xx,
// missing rows) surface as the correct HTTP status to the client and avoid
// generating Sentry 500 noise for issues that are not server bugs.
//
// `fallbackMessage` is what's returned for unclassified errors that map to
// codes.Internal. Pass a short, user-facing description (e.g. "failed to send
// agent chat message").
func translateAgentServiceError(err error, fallbackMessage string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "request was cancelled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, agentservice.ErrSessionAlreadyTerminated) {
		return status.Error(codes.NotFound, "agent chat not found")
	}
	if errors.Is(err, agentservice.ErrSessionForbidden) {
		return status.Error(codes.PermissionDenied, "agent chat is not accessible")
	}
	var providerErr agentservice.ProviderHTTPError
	if errors.As(err, &providerErr) {
		return providerStatusError(providerErr)
	}
	return status.Error(codes.Internal, fallbackMessage)
}

// providerStatusError maps an upstream HTTP status code into a sensible gRPC
// code. We intentionally avoid codes.Internal for upstream failures so that
// they don't pollute Sentry's HTTP 500 issue feed: those issues should track
// SuperPlane bugs, not Anthropic outages or rate limits.
func providerStatusError(err agentservice.ProviderHTTPError) error {
	code := err.HTTPStatusCode()
	switch {
	case code == http.StatusTooManyRequests:
		return status.Error(codes.ResourceExhausted, "agent provider is rate-limiting requests, please try again shortly")
	case code == http.StatusUnauthorized, code == http.StatusForbidden:
		return status.Error(codes.FailedPrecondition, "agent provider rejected the request, check the provider configuration")
	case code == http.StatusRequestTimeout, code == http.StatusGatewayTimeout:
		return status.Error(codes.DeadlineExceeded, "agent provider timed out")
	case code >= 500:
		return status.Error(codes.Unavailable, "agent provider is unavailable, please try again")
	case code >= 400:
		return status.Error(codes.InvalidArgument, "agent provider rejected the request")
	}
	return status.Error(codes.Internal, err.Error())
}

func ensureCanvas(orgID, canvasID uuid.UUID) error {
	if _, err := models.FindCanvas(orgID, canvasID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return status.Error(codes.NotFound, "canvas not found")
		}
		return status.Error(codes.Internal, "failed to load canvas")
	}
	return nil
}

func serializeChat(session *models.AgentSession) *pb.AgentChatInfo {
	info := &pb.AgentChatInfo{
		Id:       session.ID.String(),
		CanvasId: session.CanvasID.String(),
		Provider: session.Provider,
		Status:   session.Status,
	}
	if session.CreatedAt != nil {
		info.CreatedAt = timestamppb.New(*session.CreatedAt)
	}
	if session.UpdatedAt != nil {
		info.UpdatedAt = timestamppb.New(*session.UpdatedAt)
	}
	return info
}

func serializeMessage(message *models.AgentSessionMessage) *pb.AgentChatMessage {
	out := &pb.AgentChatMessage{
		Id:         message.ID.String(),
		Role:       message.Role,
		Content:    message.Content,
		ToolCallId: message.ToolCallID,
		ToolName:   message.ToolName,
		ToolStatus: message.ToolStatus,
	}
	if message.CreatedAt != nil {
		out.CreatedAt = timestamppb.New(*message.CreatedAt)
	}
	return out
}
