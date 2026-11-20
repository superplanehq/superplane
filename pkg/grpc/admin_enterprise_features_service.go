package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/enterprise"
	actions "github.com/superplanehq/superplane/pkg/grpc/actions/admin/enterprise_features"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/enterprise_features"
)

type AdminEnterpriseFeaturesService struct {
	pb.UnimplementedEnterpriseFeaturesServer
	features *enterprise.OrganizationFeatures
}

func NewAdminEnterpriseFeaturesService(features *enterprise.OrganizationFeatures) *AdminEnterpriseFeaturesService {
	return &AdminEnterpriseFeaturesService{features: features}
}

func (s *AdminEnterpriseFeaturesService) ListOrganizationEnterpriseFeatures(ctx context.Context, req *pb.ListOrganizationEnterpriseFeaturesRequest) (*pb.ListOrganizationEnterpriseFeaturesResponse, error) {
	return actions.ListOrganizationEnterpriseFeatures(ctx, s.features, req)
}

func (s *AdminEnterpriseFeaturesService) SetOrganizationEnterpriseFeatures(ctx context.Context, req *pb.SetOrganizationEnterpriseFeaturesRequest) (*pb.SetOrganizationEnterpriseFeaturesResponse, error) {
	return actions.SetOrganizationEnterpriseFeatures(ctx, s.features, req)
}

func (s *AdminEnterpriseFeaturesService) ListEnterpriseFeatureOrganizations(ctx context.Context, req *pb.ListEnterpriseFeatureOrganizationsRequest) (*pb.ListEnterpriseFeatureOrganizationsResponse, error) {
	return actions.ListEnterpriseFeatureOrganizations(ctx, req)
}

func (s *AdminEnterpriseFeaturesService) ListLicenseActivations(ctx context.Context, req *pb.ListLicenseActivationsRequest) (*pb.ListLicenseActivationsResponse, error) {
	return actions.ListLicenseActivations(ctx, req)
}
