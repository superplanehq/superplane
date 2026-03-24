package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	grpcagents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
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

func (s *AgentsService) CreateAgentChatSession(
	ctx context.Context,
	req *pb.CreateAgentChatSessionRequest,
) (*pb.CreateAgentChatSessionResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return grpcagents.CreateAgentChatSession(
		s.authService,
		s.jwtSigner,
		userID,
		organizationID,
		req.CanvasId,
	)
}
