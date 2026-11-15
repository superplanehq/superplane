package sso

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/enterprise"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/sso"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetOrganization(ctx context.Context, service enterprise.SSO, organizationID string) (*pb.GetOrganizationSSOResponse, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "The organization is not valid.")
	}
	view, err := service.AdminGet(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return adminToProto(view), nil
}

func DisableOrganization(ctx context.Context, service enterprise.SSO, organizationID string) (*pb.GetOrganizationSSOResponse, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "The organization is not valid.")
	}
	view, err := service.AdminDisable(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return adminToProto(view), nil
}

func VerifyDomain(ctx context.Context, service enterprise.SSO, organizationID, domainID, accountID string) (*pb.VerifyOrganizationDomainResponse, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "The organization is not valid.")
	}
	domain, err := uuid.Parse(domainID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "The domain is not valid.")
	}
	actor, err := uuid.Parse(accountID)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "account not found")
	}
	view, err := service.AdminVerifyDomain(ctx, orgID, domain, actor)
	if err != nil {
		return nil, err
	}
	return &pb.VerifyOrganizationDomainResponse{
		Id:                 view.ID.String(),
		Domain:             view.Domain,
		VerificationToken:  view.VerificationToken,
		TxtName:            view.TXTName,
		TxtValue:           view.TXTValue,
		VerificationMethod: view.VerificationMethod,
		Verified:           view.VerifiedAt != nil,
	}, nil
}

func adminToProto(view *enterprise.AdminSSO) *pb.GetOrganizationSSOResponse {
	response := &pb.GetOrganizationSSOResponse{
		Connection: &pb.AdminConnection{},
		Domains:    []*pb.AdminDomain{},
	}
	if view == nil {
		return response
	}
	response.Connection = &pb.AdminConnection{
		Exists:      view.Connection.Exists,
		Id:          view.Connection.ID.String(),
		Status:      view.Connection.Status,
		Enforcement: view.Connection.Enforcement,
		DisplayName: view.Connection.DisplayName,
		Protocol:    view.Connection.Protocol,
	}
	for _, domain := range view.Domains {
		response.Domains = append(response.Domains, &pb.AdminDomain{
			Id:                 domain.ID.String(),
			Domain:             domain.Domain,
			Verified:           domain.Verified,
			VerificationMethod: domain.VerificationMethod,
		})
	}
	return response
}
