package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/jwt"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

// Embedding the interface makes the one transport operation under test explicit.
type organizationBillingStub struct {
	enterprise.BillingOperations
	describe func(context.Context, string, *pb.DescribeOrganizationBillingRequest) (*pb.DescribeOrganizationBillingResponse, error)
}

func (s organizationBillingStub) DescribeOrganizationBilling(ctx context.Context, orgID string, req *pb.DescribeOrganizationBillingRequest) (*pb.DescribeOrganizationBillingResponse, error) {
	return s.describe(ctx, orgID, req)
}

func TestOrganizationBillingDispatchesToTheRegisteredCapability(t *testing.T) {
	registry := enterprise.NewRegistry()
	request := &pb.DescribeOrganizationBillingRequest{Id: "requested-organization"}
	response := &pb.DescribeOrganizationBillingResponse{BillingEnabled: true}
	registry.Register(enterprise.Billing, organizationBillingStub{describe: func(_ context.Context, orgID string, req *pb.DescribeOrganizationBillingRequest) (*pb.DescribeOrganizationBillingResponse, error) {
		require.Equal(t, "authorized-organization", orgID)
		require.Same(t, request, req)
		return response, nil
	}})
	services, err := NewServices(ServicesConfig{JWTSigner: jwt.NewSigner("test-secret"), Enterprise: registry})
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), authorization.DomainIdContextKey, "authorized-organization")
	actual, err := services.Organizations.DescribeOrganizationBilling(ctx, request)
	require.NoError(t, err)
	require.Same(t, response, actual)
}

func TestNewServicesRejectsAMissingBillingCapability(t *testing.T) {
	_, err := NewServices(ServicesConfig{JWTSigner: jwt.NewSigner("test-secret"), Enterprise: enterprise.NewRegistry()})
	require.ErrorContains(t, err, "enterprise capability billing is not registered")
}
