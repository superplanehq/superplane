package usage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/test/support"
)

func Test__CacheOrganizationLimits_SkipsOverwriteWhenNewerCacheExists(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	newerRetentionWindowDays := int32(60)
	newerSyncedAt := time.Now().UTC()
	require.NoError(
		t,
		models.MarkOrganizationUsageLimitsSynced(
			r.Organization.ID.String(),
			&newerRetentionWindowDays,
			newerSyncedAt,
		),
	)

	readStartedAt := newerSyncedAt.Add(-1 * time.Minute)
	writeAttemptedAt := newerSyncedAt.Add(1 * time.Minute)

	require.NoError(
		t,
		CacheOrganizationLimits(
			r.Organization.ID.String(),
			&pb.OrganizationLimits{RetentionWindowDays: 30},
			readStartedAt,
			writeAttemptedAt,
		),
	)

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	assert.Equal(t, int32(60), *organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
	assert.WithinDuration(t, newerSyncedAt, *organization.UsageLimitsSyncedAt, time.Second)
}

func Test__CacheOrganizationLimits_UpdatesWhenNoNewerCacheExists(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	existingRetentionWindowDays := int32(30)
	existingSyncedAt := time.Now().Add(-2 * time.Minute).UTC()
	require.NoError(
		t,
		models.MarkOrganizationUsageLimitsSynced(
			r.Organization.ID.String(),
			&existingRetentionWindowDays,
			existingSyncedAt,
		),
	)

	readStartedAt := time.Now().Add(-1 * time.Minute).UTC()
	writeAttemptedAt := time.Now().UTC()

	require.NoError(
		t,
		CacheOrganizationLimits(
			r.Organization.ID.String(),
			&pb.OrganizationLimits{RetentionWindowDays: 45},
			readStartedAt,
			writeAttemptedAt,
		),
	)

	organization, err := models.FindOrganizationByID(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, organization.UsageRetentionWindowDays)
	assert.Equal(t, int32(45), *organization.UsageRetentionWindowDays)
	require.NotNil(t, organization.UsageLimitsSyncedAt)
	assert.WithinDuration(t, writeAttemptedAt, *organization.UsageLimitsSyncedAt, time.Second)
}
