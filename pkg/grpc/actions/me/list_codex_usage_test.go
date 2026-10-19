package me

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/metadata"
)

func TestListCodexUsageScopesSnapshotsToOwnerAndOrganization(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	now := time.Now()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, factory.ID, r.User, now)
	require.NoError(t, err)
	taskID := uuid.New()
	require.NoError(t, connection.BeginLogin(db, taskID, now))
	require.NoError(t, connection.StoreBundle(db, taskID, connection.ReservationGeneration, 1, []byte("secret"), "ada@example.com", true, now))
	used := 25.0
	duration := int64(300)
	reset := now.Add(time.Hour).Unix()
	limits := []models.CodexUsageLimit{{LimitID: "codex", Primary: &models.CodexUsageWindow{UsedPercent: &used, WindowDurationMins: &duration, ResetsAt: &reset}}}
	require.NoError(t, connection.StoreUsage(db, limits, nil, now))
	response, err := ListCodexUsage(meContext(r))
	require.NoError(t, err)
	require.Len(t, response.Connections, 1)
	snapshot := response.Connections[0]
	assert.Equal(t, "ada@example.com", snapshot.AccountLabel)
	assert.WithinDuration(t, now, snapshot.ObservedAt.AsTime(), time.Second)
	require.Len(t, snapshot.Limits, 1)
	assert.Equal(t, used, snapshot.Limits[0].Primary.UsedPercent)
	assert.Equal(t, duration, *snapshot.Limits[0].Primary.WindowDurationMins)
	assert.Equal(t, reset, snapshot.Limits[0].Primary.ResetsAt.AsTime().Unix())

	for _, ids := range [][2]string{{uuid.NewString(), r.Organization.ID.String()}, {r.User.String(), uuid.NewString()}} {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", ids[0], "x-organization-id", ids[1]))
		other, err := ListCodexUsage(ctx)
		require.NoError(t, err)
		assert.Empty(t, other.Connections)
	}
	_, err = ListCodexUsage(context.Background())
	require.Error(t, err)

	require.NoError(t, db.Model(connection).Updates(map[string]any{"reservation_task_id": nil, "status": models.CodexConnectionStatusReady}).Error)
	connection.ReservationTaskID = nil
	connection.Status = models.CodexConnectionStatusReady
	require.NoError(t, connection.BeginLogin(db, uuid.New(), now))
	response, err = ListCodexUsage(meContext(r))
	require.NoError(t, err)
	require.Len(t, response.Connections, 1)
	assert.Empty(t, response.Connections[0].Limits, "new sign-in must not show the previous account quota")
	assert.Nil(t, response.Connections[0].ObservedAt)
	require.NoError(t, db.Model(connection).Update("reservation_task_id", nil).Error)
	connection.ReservationTaskID = nil
	require.NoError(t, connection.Disconnect(db, now))
	response, err = ListCodexUsage(meContext(r))
	require.NoError(t, err)
	assert.Empty(t, response.Connections)
	require.NoError(t, db.First(connection, "id = ?", connection.ID).Error)
	assert.Empty(t, connection.UsageLimits)
	assert.Nil(t, connection.UsageObservedAt)
}
