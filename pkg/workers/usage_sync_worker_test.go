package workers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/usage"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeUsageSyncWorkerService struct {
	enabled             bool
	retentionWindowDays int32
	describeCalls       []string
}

func (s *fakeUsageSyncWorkerService) Enabled() bool {
	return s.enabled
}

func (s *fakeUsageSyncWorkerService) SetupAccount(context.Context, string) (*pb.SetupAccountResponse, error) {
	return &pb.SetupAccountResponse{}, nil
}

func (s *fakeUsageSyncWorkerService) SetupOrganization(context.Context, string, string) (*pb.SetupOrganizationResponse, error) {
	return &pb.SetupOrganizationResponse{}, nil
}

func (s *fakeUsageSyncWorkerService) DescribeAccountLimits(context.Context, string) (*pb.DescribeAccountLimitsResponse, error) {
	return &pb.DescribeAccountLimitsResponse{}, nil
}

func (s *fakeUsageSyncWorkerService) DescribeOrganizationLimits(_ context.Context, orgID string) (*pb.DescribeOrganizationLimitsResponse, error) {
	s.describeCalls = append(s.describeCalls, orgID)
	return &pb.DescribeOrganizationLimitsResponse{
		Limits: &pb.OrganizationLimits{
			RetentionWindowDays: s.retentionWindowDays,
		},
	}, nil
}

func (s *fakeUsageSyncWorkerService) DescribeOrganizationUsage(context.Context, string) (*pb.DescribeOrganizationUsageResponse, error) {
	return &pb.DescribeOrganizationUsageResponse{}, nil
}

func (s *fakeUsageSyncWorkerService) CheckAccountLimits(context.Context, string, *pb.AccountState) (*pb.CheckAccountLimitsResponse, error) {
	return &pb.CheckAccountLimitsResponse{Allowed: true}, nil
}

func (s *fakeUsageSyncWorkerService) CheckOrganizationLimits(context.Context, string, *pb.OrganizationState, *pb.CanvasState) (*pb.CheckOrganizationLimitsResponse, error) {
	return &pb.CheckOrganizationLimitsResponse{Allowed: true}, nil
}

var _ usage.Service = (*fakeUsageSyncWorkerService)(nil)

func Test__UsageSyncWorker_BackfillRefreshesUsageLimitsCache(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	require.NoError(t, models.MarkOrganizationUsageSynced(r.Organization.ID.String(), time.Now()))
	staleLimitsSyncedAt := time.Now().Add(-2 * usageLimitsRefreshAfter)
	require.NoError(
		t,
		database.Conn().
			Model(&models.Organization{}).
			Where("id = ?", r.Organization.ID).
			Updates(map[string]any{
				"usage_retention_window_days": nil,
				"usage_limits_synced_at":      staleLimitsSyncedAt,
			}).
			Error,
	)

	service := &fakeUsageSyncWorkerService{
		enabled:             true,
		retentionWindowDays: 45,
	}
	worker := NewUsageSyncWorker("amqp://unused", service)

	worker.backfill(context.Background())

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	assert.Equal(t, int32(45), *organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
	assert.True(t, organization.UsageLimitsSyncedAt.After(staleLimitsSyncedAt))
	assert.Equal(t, []string{r.Organization.ID.String()}, service.describeCalls)
}

func Test__UsageSyncWorker_ConsumeOrganizationPlanChangedUpdatesUsageLimitsCache(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewUsageSyncWorker("amqp://unused", &fakeUsageSyncWorkerService{enabled: true})
	eventTimestamp := time.Now().Add(-5 * time.Minute).UTC()

	body, err := proto.Marshal(&pb.OrganizationPlanChanged{
		OrganizationId: r.Organization.ID.String(),
		PlanName:       "growth",
		Limits: &pb.OrganizationLimits{
			RetentionWindowDays: 60,
		},
		Timestamp: timestamppb.New(eventTimestamp),
	})
	require.NoError(t, err)

	require.NoError(t, worker.ConsumeOrganizationPlanChanged(tackle.NewFakeDelivery(body)))

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	assert.Equal(t, int32(60), *organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
	assert.WithinDuration(t, eventTimestamp, *organization.UsageLimitsSyncedAt, time.Second)
}

func Test__UsageSyncWorker_ConsumeOrganizationPlanChangedSkipsStaleMessage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	worker := NewUsageSyncWorker("amqp://unused", &fakeUsageSyncWorkerService{enabled: true})
	newerTimestamp := time.Now().UTC()
	olderTimestamp := newerTimestamp.Add(-5 * time.Minute)

	newerBody, err := proto.Marshal(&pb.OrganizationPlanChanged{
		OrganizationId: r.Organization.ID.String(),
		PlanName:       "growth",
		Limits: &pb.OrganizationLimits{
			RetentionWindowDays: 60,
		},
		Timestamp: timestamppb.New(newerTimestamp),
	})
	require.NoError(t, err)

	olderBody, err := proto.Marshal(&pb.OrganizationPlanChanged{
		OrganizationId: r.Organization.ID.String(),
		PlanName:       "free",
		Limits: &pb.OrganizationLimits{
			RetentionWindowDays: 30,
		},
		Timestamp: timestamppb.New(olderTimestamp),
	})
	require.NoError(t, err)

	require.NoError(t, worker.ConsumeOrganizationPlanChanged(tackle.NewFakeDelivery(newerBody)))
	require.NoError(t, worker.ConsumeOrganizationPlanChanged(tackle.NewFakeDelivery(olderBody)))

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	assert.Equal(t, int32(60), *organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
	assert.WithinDuration(t, newerTimestamp, *organization.UsageLimitsSyncedAt, time.Second)
}

func Test__UsageSyncWorker_ConsumeOrganizationPlanChangedSkipsMissingOrganization(t *testing.T) {
	worker := NewUsageSyncWorker("amqp://unused", &fakeUsageSyncWorkerService{enabled: true})
	missingOrganizationID := uuid.New()

	body, err := proto.Marshal(&pb.OrganizationPlanChanged{
		OrganizationId: missingOrganizationID.String(),
		PlanName:       "growth",
		Limits: &pb.OrganizationLimits{
			RetentionWindowDays: 45,
		},
		Timestamp: timestamppb.New(time.Now().UTC()),
	})
	require.NoError(t, err)

	require.NoError(t, worker.ConsumeOrganizationPlanChanged(tackle.NewFakeDelivery(body)))
}
