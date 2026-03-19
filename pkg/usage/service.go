package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/superplanehq/superplane/pkg/config"
	pb "github.com/superplanehq/superplane/pkg/protos/usage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultTimeout = 5 * time.Second

type Service interface {
	Enabled() bool
	SetupAccount(ctx context.Context, accountID string) (*pb.SetupAccountResponse, error)
	SetupOrganization(ctx context.Context, organizationID, accountID string) (*pb.SetupOrganizationResponse, error)
	DescribeOrganizationLimits(ctx context.Context, organizationID string) (*pb.DescribeOrganizationLimitsResponse, error)
	DescribeOrganizationUsage(ctx context.Context, organizationID string) (*pb.DescribeOrganizationUsageResponse, error)
}

type disabledService struct{}

func NewServiceFromEnv() (Service, error) {
	url := config.UsageGRPCURL()
	if url == "" {
		return disabledService{}, nil
	}

	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create usage grpc client: %w", err)
	}

	return &grpcService{
		client: pb.NewUsageClient(conn),
	}, nil
}

func (disabledService) Enabled() bool {
	return false
}

func (disabledService) SetupAccount(context.Context, string) (*pb.SetupAccountResponse, error) {
	return nil, ErrUsageDisabled
}

func (disabledService) SetupOrganization(context.Context, string, string) (*pb.SetupOrganizationResponse, error) {
	return nil, ErrUsageDisabled
}

func (disabledService) DescribeOrganizationLimits(context.Context, string) (*pb.DescribeOrganizationLimitsResponse, error) {
	return nil, ErrUsageDisabled
}

func (disabledService) DescribeOrganizationUsage(context.Context, string) (*pb.DescribeOrganizationUsageResponse, error) {
	return nil, ErrUsageDisabled
}

type grpcService struct {
	client pb.UsageClient
}

func (s *grpcService) Enabled() bool {
	return true
}

func (s *grpcService) SetupAccount(ctx context.Context, accountID string) (*pb.SetupAccountResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.SetupAccount(callCtx, &pb.SetupAccountRequest{AccountId: accountID})
}

func (s *grpcService) SetupOrganization(
	ctx context.Context,
	organizationID, accountID string,
) (*pb.SetupOrganizationResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.SetupOrganization(callCtx, &pb.SetupOrganizationRequest{
		OrganizationId: organizationID,
		AccountId:      accountID,
	})
}

func (s *grpcService) DescribeOrganizationLimits(
	ctx context.Context,
	organizationID string,
) (*pb.DescribeOrganizationLimitsResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.DescribeOrganizationLimits(callCtx, &pb.DescribeOrganizationLimitsRequest{
		OrganizationId: organizationID,
	})
}

func (s *grpcService) DescribeOrganizationUsage(
	ctx context.Context,
	organizationID string,
) (*pb.DescribeOrganizationUsageResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	return s.client.DescribeOrganizationUsage(callCtx, &pb.DescribeOrganizationUsageRequest{
		OrganizationId: organizationID,
	})
}
