package agents

import (
	"context"

	log "github.com/sirupsen/logrus"
	pb "github.com/superplanehq/superplane/pkg/protos/agents"
	internalpb "github.com/superplanehq/superplane/pkg/protos/private/agents"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func DescribeAgentUsage(ctx context.Context, agentURL string, orgID string) (*pb.DescribeAgentUsageResponse, error) {
	conn, err := grpc.NewClient(agentURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, status.Error(codes.Unavailable, "failed to create agent GRPC client")
	}
	defer closeAgentConnection(conn)

	client := internalpb.NewAgentsClient(conn)
	response, err := client.DescribeOrganizationAgentUsage(ctx, &internalpb.DescribeOrganizationAgentUsageRequest{
		OrgId: orgID,
	})

	if err != nil {
		log.WithError(err).Errorf("failed to describe agent usage for org %s", orgID)
		return nil, status.Error(codes.Unavailable, "failed to describe agent usage")
	}

	return &pb.DescribeAgentUsageResponse{
		Usage: serializeChatUsage(response.GetUsage()),
	}, nil
}
