package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

type OrganizationService struct {
}

func NewOrganizationService() *OrganizationService {
	return &OrganizationService{}
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, req *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	return organizations.CreateOrganization(ctx, req)
}

func (s *OrganizationService) DescribeOrganization(ctx context.Context, req *pb.DescribeOrganizationRequest) (*pb.DescribeOrganizationResponse, error) {
	return organizations.DescribeOrganization(ctx, req)
}

func (s *OrganizationService) ListOrganizations(ctx context.Context, req *pb.ListOrganizationsRequest) (*pb.ListOrganizationsResponse, error) {
	return organizations.ListOrganizations(ctx, req)
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, req *pb.UpdateOrganizationRequest) (*pb.UpdateOrganizationResponse, error) {
	return organizations.UpdateOrganization(ctx, req)
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, req *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	return organizations.DeleteOrganization(ctx, req)
}
