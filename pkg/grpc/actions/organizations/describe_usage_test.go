package organizations

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUsageService struct {
	enabled bool

	describeLimitsResponse *pb.DescribeOrganizationLimitsResponse
	describeLimitsError    error
	describeUsageResponse  *pb.DescribeOrganizationUsageResponse
	describeUsageError     error
	setupAccountError      error
	setupOrganizationError error
	setupOrganizationResp  *pb.SetupOrganizationResponse
	checkAccountResponse   *pb.CheckAccountLimitsResponse
	checkAccountError      error
	checkOrganizationResp  *pb.CheckOrganizationLimitsResponse
	checkOrganizationError error

	setupAccountCalls      []string
	setupOrganizationCalls [][2]string
	checkAccountCalls      []struct {
		accountID string
		state     *pb.AccountState
	}
	checkOrganizationCalls []struct {
		organizationID string
		state          *pb.OrganizationState
		canvas         *pb.CanvasState
	}
}

func (s *fakeUsageService) Enabled() bool {
	return s.enabled
}

func (s *fakeUsageService) SetupAccount(_ context.Context, accountID string) (*pb.SetupAccountResponse, error) {
	s.setupAccountCalls = append(s.setupAccountCalls, accountID)
	if s.setupAccountError != nil {
		return nil, s.setupAccountError
	}

	return &pb.SetupAccountResponse{}, nil
}

func (s *fakeUsageService) SetupOrganization(
	_ context.Context,
	organizationID, accountID string,
) (*pb.SetupOrganizationResponse, error) {
	s.setupOrganizationCalls = append(s.setupOrganizationCalls, [2]string{organizationID, accountID})
	if s.setupOrganizationError != nil {
		return nil, s.setupOrganizationError
	}

	if s.setupOrganizationResp != nil {
		return s.setupOrganizationResp, nil
	}

	return &pb.SetupOrganizationResponse{}, nil
}

func (s *fakeUsageService) DescribeOrganizationLimits(
	_ context.Context,
	_ string,
) (*pb.DescribeOrganizationLimitsResponse, error) {
	if s.describeLimitsError != nil && len(s.setupOrganizationCalls) == 0 {
		return nil, s.describeLimitsError
	}

	return s.describeLimitsResponse, nil
}

func (s *fakeUsageService) DescribeAccountLimits(
	context.Context,
	string,
) (*pb.DescribeAccountLimitsResponse, error) {
	return &pb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakeUsageService) DescribeOrganizationUsage(
	_ context.Context,
	_ string,
) (*pb.DescribeOrganizationUsageResponse, error) {
	if s.describeUsageError != nil && len(s.setupOrganizationCalls) == 0 {
		return nil, s.describeUsageError
	}

	return s.describeUsageResponse, nil
}

func (s *fakeUsageService) CheckAccountLimits(
	_ context.Context,
	accountID string,
	state *pb.AccountState,
) (*pb.CheckAccountLimitsResponse, error) {
	s.checkAccountCalls = append(s.checkAccountCalls, struct {
		accountID string
		state     *pb.AccountState
	}{
		accountID: accountID,
		state:     state,
	})
	if s.checkAccountError != nil {
		return nil, s.checkAccountError
	}
	if s.checkAccountResponse != nil {
		return s.checkAccountResponse, nil
	}

	return &pb.CheckAccountLimitsResponse{Allowed: true}, nil
}

func (s *fakeUsageService) CheckOrganizationLimits(
	_ context.Context,
	organizationID string,
	state *pb.OrganizationState,
	canvas *pb.CanvasState,
) (*pb.CheckOrganizationLimitsResponse, error) {
	s.checkOrganizationCalls = append(s.checkOrganizationCalls, struct {
		organizationID string
		state          *pb.OrganizationState
		canvas         *pb.CanvasState
	}{
		organizationID: organizationID,
		state:          state,
		canvas:         canvas,
	})
	if s.checkOrganizationError != nil {
		return nil, s.checkOrganizationError
	}
	if s.checkOrganizationResp != nil {
		return s.checkOrganizationResp, nil
	}

	return &pb.CheckOrganizationLimitsResponse{Allowed: true}, nil
}

var _ usage.Service = (*fakeUsageService)(nil)

func Test__DescribeUsage(t *testing.T) {
	r := support.Setup(t)

	t.Run("usage disabled", func(t *testing.T) {
		response, err := DescribeUsage(context.Background(), &fakeUsageService{enabled: false}, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.False(t, response.Enabled)
		assert.Equal(t, "Usage tracking is disabled for this SuperPlane instance.", response.StatusMessage)
		assert.Nil(t, response.Limits)
		assert.Nil(t, response.Usage)
	})

	t.Run("returns existing usage data", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		service := &fakeUsageService{
			enabled: true,
			describeLimitsResponse: &pb.DescribeOrganizationLimitsResponse{
				Limits: &pb.OrganizationLimits{
					MaxCanvases:         10,
					MaxNodesPerCanvas:   20,
					MaxUsers:            5,
					RetentionWindowDays: 30,
					MaxEventsPerMonth:   1000,
					MaxIntegrations:     3,
				},
			},
			describeUsageResponse: &pb.DescribeOrganizationUsageResponse{
				Usage: &pb.OrganizationUsage{
					Canvases:                            4,
					EventBucketLevel:                    42,
					EventBucketCapacity:                 1000,
					EventBucketLastUpdatedAtUnixSeconds: now.Unix(),
					NextEventBucketLeakAtUnixSeconds:    now.Add(24 * time.Hour).Unix(),
				},
			},
		}

		response, err := DescribeUsage(context.Background(), service, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.True(t, response.Enabled)
		require.NotNil(t, response.Limits)
		assert.Equal(t, int32(10), response.Limits.MaxCanvases)
		require.NotNil(t, response.Usage)
		assert.Equal(t, int32(4), response.Usage.Canvases)
		assert.Equal(t, 42.0, response.Usage.EventBucketLevel)
		require.NotNil(t, response.Usage.NextEventBucketDecreaseAt)
		assert.WithinDuration(t, now.Add(24*time.Hour), response.Usage.NextEventBucketDecreaseAt.AsTime(), time.Second)
		assert.Empty(t, service.setupAccountCalls)
		assert.Empty(t, service.setupOrganizationCalls)

		organization, err := models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, organization.UsageRetentionWindowDays)
		assert.Equal(t, int32(30), *organization.UsageRetentionWindowDays)
		require.NotNil(t, organization.UsageLimitsSyncedAt)
	})

	t.Run("sets up remote organization when not configured", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)

		service := &fakeUsageService{
			enabled:             true,
			describeLimitsError: status.Error(codes.NotFound, "organization not configured"),
			describeLimitsResponse: &pb.DescribeOrganizationLimitsResponse{
				Limits: &pb.OrganizationLimits{
					MaxCanvases:       12,
					MaxNodesPerCanvas: 30,
				},
			},
			describeUsageError: status.Error(codes.NotFound, "organization not configured"),
			setupOrganizationResp: &pb.SetupOrganizationResponse{
				Limits: &pb.OrganizationLimits{
					MaxCanvases:       12,
					MaxNodesPerCanvas: 30,
				},
			},
			describeUsageResponse: &pb.DescribeOrganizationUsageResponse{
				Usage: &pb.OrganizationUsage{
					Canvases:                            1,
					EventBucketLevel:                    2,
					EventBucketCapacity:                 12,
					EventBucketLastUpdatedAtUnixSeconds: now.Unix(),
					NextEventBucketLeakAtUnixSeconds:    now.Add(24 * time.Hour).Unix(),
				},
			},
		}

		response, err := DescribeUsage(context.Background(), service, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, service.setupAccountCalls, 1)
		assert.Equal(t, r.Account.ID.String(), service.setupAccountCalls[0])
		require.Len(t, service.setupOrganizationCalls, 1)
		assert.Equal(t, r.Organization.ID.String(), service.setupOrganizationCalls[0][0])
		assert.Equal(t, r.Account.ID.String(), service.setupOrganizationCalls[0][1])
		require.NotNil(t, response.Limits)
		assert.Equal(t, int32(12), response.Limits.MaxCanvases)
		require.NotNil(t, response.Usage)
		assert.Equal(t, int32(1), response.Usage.Canvases)
		require.NotNil(t, response.Usage.NextEventBucketDecreaseAt)
		assert.WithinDuration(t, now.Add(24*time.Hour), response.Usage.NextEventBucketDecreaseAt.AsTime(), time.Second)
	})
}
