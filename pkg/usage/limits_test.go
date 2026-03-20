package usage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeLimitService struct {
	enabled bool

	setupAccountCalls      []string
	setupOrganizationCalls [][2]string

	checkAccountResponse      *usagepb.CheckAccountLimitsResponse
	checkAccountError         error
	checkOrganizationResponse *usagepb.CheckOrganizationLimitsResponse
	checkOrganizationError    error

	checkAccountCalls      int
	checkOrganizationCalls int
}

func (s *fakeLimitService) Enabled() bool {
	return s.enabled
}

func (s *fakeLimitService) SetupAccount(_ context.Context, accountID string) (*usagepb.SetupAccountResponse, error) {
	s.setupAccountCalls = append(s.setupAccountCalls, accountID)
	return &usagepb.SetupAccountResponse{}, nil
}

func (s *fakeLimitService) SetupOrganization(
	_ context.Context,
	organizationID, accountID string,
) (*usagepb.SetupOrganizationResponse, error) {
	s.setupOrganizationCalls = append(s.setupOrganizationCalls, [2]string{organizationID, accountID})
	return &usagepb.SetupOrganizationResponse{}, nil
}

func (s *fakeLimitService) DescribeAccountLimits(context.Context, string) (*usagepb.DescribeAccountLimitsResponse, error) {
	return &usagepb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakeLimitService) DescribeOrganizationLimits(context.Context, string) (*usagepb.DescribeOrganizationLimitsResponse, error) {
	return &usagepb.DescribeOrganizationLimitsResponse{
		Limits: &usagepb.OrganizationLimits{},
	}, nil
}

func (s *fakeLimitService) DescribeOrganizationUsage(context.Context, string) (*usagepb.DescribeOrganizationUsageResponse, error) {
	return &usagepb.DescribeOrganizationUsageResponse{}, nil
}

func (s *fakeLimitService) CheckAccountLimits(
	context.Context,
	string,
	*usagepb.AccountState,
) (*usagepb.CheckAccountLimitsResponse, error) {
	s.checkAccountCalls++
	if s.checkAccountCalls == 1 && s.checkAccountError != nil {
		return nil, s.checkAccountError
	}

	return s.checkAccountResponse, nil
}

func (s *fakeLimitService) CheckOrganizationLimits(
	context.Context,
	string,
	*usagepb.OrganizationState,
	*usagepb.CanvasState,
) (*usagepb.CheckOrganizationLimitsResponse, error) {
	s.checkOrganizationCalls++
	if s.checkOrganizationCalls == 1 && s.checkOrganizationError != nil {
		return nil, s.checkOrganizationError
	}

	return s.checkOrganizationResponse, nil
}

func TestEnsureAccountWithinLimitsReturnsResourceExhaustedForViolations(t *testing.T) {
	service := &fakeLimitService{
		enabled: true,
		checkAccountResponse: &usagepb.CheckAccountLimitsResponse{
			Allowed: false,
			Violations: []*usagepb.LimitViolation{
				{Limit: usagepb.LimitName_LIMIT_NAME_MAX_ORGANIZATIONS},
			},
		},
	}

	err := EnsureAccountWithinLimits(context.Background(), service, "account-id", &usagepb.AccountState{Organizations: 2})
	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.Equal(t, "account organization limit exceeded", status.Convert(err).Message())
}

func TestEnsureAccountWithinLimitsSetsUpMissingAccount(t *testing.T) {
	service := &fakeLimitService{
		enabled:           true,
		checkAccountError: status.Error(codes.NotFound, "account not configured"),
		checkAccountResponse: &usagepb.CheckAccountLimitsResponse{
			Allowed: true,
		},
	}

	err := EnsureAccountWithinLimits(context.Background(), service, "account-id", &usagepb.AccountState{Organizations: 1})
	require.NoError(t, err)
	assert.Equal(t, 2, service.checkAccountCalls)
	assert.Equal(t, []string{"account-id"}, service.setupAccountCalls)
}

func TestEnsureOrganizationWithinLimitsSyncsOnNotFound(t *testing.T) {
	r := support.Setup(t)
	service := &fakeLimitService{
		enabled:                true,
		checkOrganizationError: status.Error(codes.NotFound, "organization not configured"),
		checkOrganizationResponse: &usagepb.CheckOrganizationLimitsResponse{
			Allowed: true,
		},
	}

	err := EnsureOrganizationWithinLimits(
		context.Background(),
		service,
		r.Organization.ID.String(),
		&usagepb.OrganizationState{Canvases: 1},
		nil,
	)
	require.NoError(t, err)
	assert.Len(t, service.setupAccountCalls, 1)
	assert.Len(t, service.setupOrganizationCalls, 1)
	assert.Equal(t, 2, service.checkOrganizationCalls)
}
