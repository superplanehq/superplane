package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/agentservice"
	"github.com/superplanehq/superplane/pkg/authorization"
	agents "github.com/superplanehq/superplane/pkg/grpc/actions/agents"
	"github.com/superplanehq/superplane/pkg/jwt"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentsService struct {
	agentService agentservice.Service
	jwtSigner    *jwt.Signer
	publicURL    string
}

func NewAgentsService(agentService agentservice.Service, jwtSigner *jwt.Signer, publicURL string) *AgentsService {
	return &AgentsService{
		agentService: agentService,
		jwtSigner:    jwtSigner,
		publicURL:    publicURL,
	}
}

func (s *AgentsService) ListAgents(ctx context.Context, req *pb.ListAgentsRequest) (*pb.ListAgentsResponse, error) {
	organizationID, userID, err := agentContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.ListAgents(ctx, &internalpb.ListAgentsRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		UserId:   userID,
	})
	if err != nil {
		return nil, err
	}

	serialized := make([]*pb.AgentInfo, 0, len(response.Agents))
	for _, agent := range response.Agents {
		info, serializeErr := agents.SerializeAgentInfo(agent)
		if serializeErr != nil {
			return nil, serializeErr
		}
		serialized = append(serialized, info)
	}

	return &pb.ListAgentsResponse{
		Agents: serialized,
	}, nil
}

func (s *AgentsService) CreateAgent(ctx context.Context, req *pb.CreateAgentRequest) (*pb.CreateAgentResponse, error) {
	organizationID, userID, err := agentContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	publicURL, err := agents.RequireAgentPublicURL(s.publicURL)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.CreateAgent(ctx, &internalpb.CreateAgentRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		UserId:   userID,
	})
	if err != nil {
		return nil, err
	}

	if response.GetAgent() == nil {
		return nil, status.Error(codes.Internal, "agent service did not return an agent")
	}

	return agents.MintAgentStreamResponse(
		s.jwtSigner,
		publicURL,
		userID,
		organizationID,
		req.CanvasId,
		response.Agent.Id,
	)
}

func (s *AgentsService) DescribeAgent(ctx context.Context, req *pb.DescribeAgentRequest) (*pb.DescribeAgentResponse, error) {
	organizationID, userID, err := agentContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.DescribeAgent(ctx, &internalpb.DescribeAgentRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		AgentId:  req.AgentId,
		UserId:   userID,
	})
	if err != nil {
		return nil, err
	}

	info, err := agents.SerializeAgentInfo(response.Agent)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeAgentResponse{
		Agent: info,
	}, nil
}

func (s *AgentsService) ListAgentMessages(ctx context.Context, req *pb.ListAgentMessagesRequest) (*pb.ListAgentMessagesResponse, error) {
	organizationID, userID, err := agentContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.ListAgentMessages(ctx, &internalpb.ListAgentMessagesRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		AgentId:  req.AgentId,
		UserId:   userID,
	})
	if err != nil {
		return nil, err
	}

	return &pb.ListAgentMessagesResponse{
		Messages: agents.SerializeAgentMessages(response.Messages),
	}, nil
}

func (s *AgentsService) ResumeAgent(ctx context.Context, req *pb.ResumeAgentRequest) (*pb.ResumeAgentResponse, error) {
	organizationID, userID, err := agentContextFromRequest(ctx)
	if err != nil {
		return nil, err
	}

	publicURL, err := agents.RequireAgentPublicURL(s.publicURL)
	if err != nil {
		return nil, err
	}

	response, err := s.agentService.DescribeAgent(ctx, &internalpb.DescribeAgentRequest{
		OrgId:    organizationID,
		CanvasId: req.CanvasId,
		AgentId:  req.AgentId,
		UserId:   userID,
	})
	if err != nil {
		return nil, err
	}

	if response.GetAgent() == nil {
		return nil, status.Error(codes.Internal, "agent service did not return an agent")
	}

	return agents.MintResumeAgentStreamResponse(
		s.jwtSigner,
		publicURL,
		userID,
		organizationID,
		req.CanvasId,
		response.GetAgent().GetId(),
	)
}

func agentContextFromRequest(ctx context.Context) (string, string, error) {
	organizationID, ok := ctx.Value(authorization.OrganizationContextKey).(string)
	if !ok || organizationID == "" {
		return "", "", status.Error(codes.Internal, "organization context is missing")
	}

	userID, err := userIDFromContext(ctx)
	if err != nil {
		return "", "", err
	}

	return organizationID, userID, nil
}
