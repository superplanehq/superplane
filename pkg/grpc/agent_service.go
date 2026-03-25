package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	agents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/jwt"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
)

type AgentsService struct {
	authService authorization.Authorization
	jwtSigner   *jwt.Signer
}

func NewAgentsService(authService authorization.Authorization, jwtSigner *jwt.Signer) *AgentsService {
	return &AgentsService{
		authService: authService,
		jwtSigner:   jwtSigner,
	}
}

func (s *AgentsService) GenerateAgentChatToken(ctx context.Context, req *pb.GenerateAgentChatTokenRequest) (*pb.GenerateAgentChatTokenResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return agents.GenerateAgentChatToken(
		s.authService,
		s.jwtSigner,
		userID,
		organizationID,
		req.CanvasId,
	)
}
