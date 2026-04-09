package agents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUsageService struct {
	enabled            bool
	usageResponse      *usagepb.DescribeOrganizationUsageResponse
	usageError         error
}

func (s *fakeUsageService) Enabled() bool { return s.enabled }
func (s *fakeUsageService) SetupAccount(context.Context, string) (*usagepb.SetupAccountResponse, error) {
	return nil, nil
}
func (s *fakeUsageService) SetupOrganization(context.Context, string, string) (*usagepb.SetupOrganizationResponse, error) {
	return nil, nil
}
func (s *fakeUsageService) DescribeAccountLimits(context.Context, string) (*usagepb.DescribeAccountLimitsResponse, error) {
	return nil, nil
}
func (s *fakeUsageService) DescribeOrganizationLimits(context.Context, string) (*usagepb.DescribeOrganizationLimitsResponse, error) {
	return nil, nil
}
func (s *fakeUsageService) DescribeOrganizationUsage(context.Context, string) (*usagepb.DescribeOrganizationUsageResponse, error) {
	return s.usageResponse, s.usageError
}
func (s *fakeUsageService) CheckAccountLimits(context.Context, string, *usagepb.AccountState) (*usagepb.CheckAccountLimitsResponse, error) {
	return nil, nil
}
func (s *fakeUsageService) CheckOrganizationLimits(context.Context, string, *usagepb.OrganizationState, *usagepb.CanvasState) (*usagepb.CheckOrganizationLimitsResponse, error) {
	return nil, nil
}

// compile-time check
var _ usage.Service = (*fakeUsageService)(nil)

func TestEnsureAgentTokensWithinLimits_NilService(t *testing.T) {
	err := EnsureAgentTokensWithinLimits(context.Background(), nil, "org-1")
	require.NoError(t, err)
}

func TestEnsureAgentTokensWithinLimits_DisabledService(t *testing.T) {
	svc := &fakeUsageService{enabled: false}
	err := EnsureAgentTokensWithinLimits(context.Background(), svc, "org-1")
	require.NoError(t, err)
}

func TestEnsureAgentTokensWithinLimits_UnlimitedCapacity(t *testing.T) {
	svc := &fakeUsageService{
		enabled: true,
		usageResponse: &usagepb.DescribeOrganizationUsageResponse{
			Usage: &usagepb.OrganizationUsage{
				AgentTokenBucketLevel:    50000,
				AgentTokenBucketCapacity: -1,
			},
		},
	}
	err := EnsureAgentTokensWithinLimits(context.Background(), svc, "org-1")
	require.NoError(t, err)
}

func TestEnsureAgentTokensWithinLimits_WithinLimit(t *testing.T) {
	svc := &fakeUsageService{
		enabled: true,
		usageResponse: &usagepb.DescribeOrganizationUsageResponse{
			Usage: &usagepb.OrganizationUsage{
				AgentTokenBucketLevel:    500,
				AgentTokenBucketCapacity: 100000,
			},
		},
	}
	err := EnsureAgentTokensWithinLimits(context.Background(), svc, "org-1")
	require.NoError(t, err)
}

func TestEnsureAgentTokensWithinLimits_AtCapacity(t *testing.T) {
	svc := &fakeUsageService{
		enabled: true,
		usageResponse: &usagepb.DescribeOrganizationUsageResponse{
			Usage: &usagepb.OrganizationUsage{
				AgentTokenBucketLevel:    100000,
				AgentTokenBucketCapacity: 100000,
			},
		},
	}
	err := EnsureAgentTokensWithinLimits(context.Background(), svc, "org-1")
	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.Equal(t, "organization agent token limit exceeded", status.Convert(err).Message())
}

func TestEnsureAgentTokensWithinLimits_OverCapacity(t *testing.T) {
	svc := &fakeUsageService{
		enabled: true,
		usageResponse: &usagepb.DescribeOrganizationUsageResponse{
			Usage: &usagepb.OrganizationUsage{
				AgentTokenBucketLevel:    150000,
				AgentTokenBucketCapacity: 100000,
			},
		},
	}
	err := EnsureAgentTokensWithinLimits(context.Background(), svc, "org-1")
	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestEnsureAgentTokensWithinLimits_UsageServiceError(t *testing.T) {
	svc := &fakeUsageService{
		enabled:    true,
		usageError: status.Error(codes.Internal, "service unavailable"),
	}
	err := EnsureAgentTokensWithinLimits(context.Background(), svc, "org-1")
	require.NoError(t, err, "should not block chat when usage service is unavailable")
}
