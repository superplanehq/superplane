package usage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeSyncService struct {
	enabled bool

	setupAccountCalls      []string
	setupOrganizationCalls [][2]string
	setupAccountError      error
	setupOrganizationError error
	describeLimitsError    error
}

func (s *fakeSyncService) Enabled() bool {
	return s.enabled
}

func (s *fakeSyncService) SetupAccount(_ context.Context, accountID string) (*usagepb.SetupAccountResponse, error) {
	s.setupAccountCalls = append(s.setupAccountCalls, accountID)
	if s.setupAccountError != nil {
		return nil, s.setupAccountError
	}

	return &usagepb.SetupAccountResponse{}, nil
}

func (s *fakeSyncService) SetupOrganization(
	_ context.Context,
	organizationID, accountID string,
) (*usagepb.SetupOrganizationResponse, error) {
	s.setupOrganizationCalls = append(s.setupOrganizationCalls, [2]string{organizationID, accountID})
	if s.setupOrganizationError != nil {
		return nil, s.setupOrganizationError
	}

	return &usagepb.SetupOrganizationResponse{}, nil
}

func (s *fakeSyncService) DescribeOrganizationLimits(
	_ context.Context,
	_ string,
) (*usagepb.DescribeOrganizationLimitsResponse, error) {
	if s.describeLimitsError != nil {
		return nil, s.describeLimitsError
	}

	return &usagepb.DescribeOrganizationLimitsResponse{
		Limits: &usagepb.OrganizationLimits{},
	}, nil
}

func (s *fakeSyncService) DescribeAccountLimits(
	context.Context,
	string,
) (*usagepb.DescribeAccountLimitsResponse, error) {
	return &usagepb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakeSyncService) DescribeOrganizationUsage(
	context.Context,
	string,
) (*usagepb.DescribeOrganizationUsageResponse, error) {
	return &usagepb.DescribeOrganizationUsageResponse{}, nil
}

func (s *fakeSyncService) CheckAccountLimits(
	context.Context,
	string,
	*usagepb.AccountState,
) (*usagepb.CheckAccountLimitsResponse, error) {
	return &usagepb.CheckAccountLimitsResponse{Allowed: true}, nil
}

func (s *fakeSyncService) CheckOrganizationLimits(
	context.Context,
	string,
	*usagepb.OrganizationState,
	*usagepb.CanvasState,
) (*usagepb.CheckOrganizationLimitsResponse, error) {
	return &usagepb.CheckOrganizationLimitsResponse{Allowed: true}, nil
}

func TestSyncOrganizationMarksOrganizationAsSynced(t *testing.T) {
	r := support.Setup(t)
	service := &fakeSyncService{enabled: true}

	err := SyncOrganization(context.Background(), service, r.Organization.ID.String())
	require.NoError(t, err)

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageSyncedAt)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	assert.Equal(t, int32(0), *organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
	assert.Len(t, service.setupAccountCalls, 1)
	assert.Len(t, service.setupOrganizationCalls, 1)
}

func TestSyncOrganizationSkipsAlreadySyncedOrganizations(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.MarkOrganizationUsageSynced(r.Organization.ID.String(), time.Now()))

	service := &fakeSyncService{enabled: true}
	err := SyncOrganization(context.Background(), service, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, service.setupAccountCalls)
	assert.Empty(t, service.setupOrganizationCalls)
}

func TestSyncOrganizationTreatsExistingRemoteOrganizationAsSynced(t *testing.T) {
	r := support.Setup(t)
	service := &fakeSyncService{
		enabled:                true,
		setupOrganizationError: status.Error(codes.AlreadyExists, "already exists"),
	}

	err := SyncOrganization(context.Background(), service, r.Organization.ID.String())
	require.NoError(t, err)

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageSyncedAt)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
}

func TestSyncOrganizationTreatsResourceExhaustedWithExistingRemoteOrganizationAsSynced(t *testing.T) {
	r := support.Setup(t)
	service := &fakeSyncService{
		enabled:                true,
		setupOrganizationError: status.Error(codes.ResourceExhausted, "account limit exceeded"),
	}

	err := SyncOrganization(context.Background(), service, r.Organization.ID.String())
	require.NoError(t, err)

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageSyncedAt)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
}

func TestMarkOrganizationUsageSyncedPreservesNewerLimitsCache(t *testing.T) {
	r := support.Setup(t)

	newerRetentionWindowDays := int32(60)
	newerLimitsSyncedAt := time.Now().UTC()
	require.NoError(
		t,
		models.MarkOrganizationUsageLimitsSynced(
			r.Organization.ID.String(),
			&newerRetentionWindowDays,
			newerLimitsSyncedAt,
		),
	)

	readStartedAt := newerLimitsSyncedAt.Add(-1 * time.Minute)
	usageSyncedAt := newerLimitsSyncedAt.Add(1 * time.Minute)

	require.NoError(
		t,
		markOrganizationUsageSynced(
			r.Organization.ID.String(),
			readStartedAt,
			usageSyncedAt,
			&usagepb.OrganizationLimits{RetentionWindowDays: 30},
		),
	)

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageSyncedAt)
	assert.WithinDuration(t, usageSyncedAt, *organization.UsageSyncedAt, time.Second)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	assert.Equal(t, int32(60), *organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
	assert.WithinDuration(t, newerLimitsSyncedAt, *organization.UsageLimitsSyncedAt, time.Second)
}
