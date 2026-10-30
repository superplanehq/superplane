package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestListFactoryCodexUsageShowsOnlyConnectedMembersInOrganization(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	owner := support.CreateUser(t, r, r.Organization.ID)
	connection, err := models.CreateCodexConnection(db, r.Organization.ID, owner.ID, now)
	require.NoError(t, err)
	require.NoError(t, db.Model(connection).Updates(map[string]any{
		"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("private-credentials"), "account_label": "private-chatgpt-account",
	}).Error)
	used := 25.0
	reset := now.Add(time.Hour).Unix()
	require.NoError(t, connection.StoreUsage(db, []models.CodexUsageLimit{{LimitID: "codex", Primary: &models.CodexUsageWindow{UsedPercent: &used, ResetsAt: &reset}}}, nil, now))
	disconnected, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, now)
	require.NoError(t, err)
	assert.Equal(t, models.CodexConnectionStatusDisconnected, disconnected.Status)
	foreignOrg, err := models.CreateOrganization(support.RandomName("other"), "")
	require.NoError(t, err)
	foreignUser := support.CreateUser(t, r, foreignOrg.ID)
	foreign, err := models.CreateCodexConnection(db, foreignOrg.ID, foreignUser.ID, now)
	require.NoError(t, err)
	require.NoError(t, db.Model(foreign).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("foreign-secret")}).Error)
	req := &pb.ListFactoryCodexUsageRequest{FactoryId: factory.ID.String()}
	response, err := ListFactoryCodexUsage(onboardingContext(r), r.Organization.ID.String(), req)
	require.NoError(t, err)
	require.Len(t, response.Members, 1)
	member := response.Members[0]
	assert.Equal(t, owner.ID.String(), member.UserId)
	assert.Equal(t, owner.Name, member.DisplayName)
	assert.Equal(t, owner.GetEmail(), member.Email)
	require.Len(t, member.Limits, 1)
	assert.Equal(t, used, member.Limits[0].Primary.UsedPercent)
	assert.Equal(t, reset, member.Limits[0].Primary.ResetsAt.AsTime().Unix())
	assert.WithinDuration(t, now, member.ObservedAt.AsTime(), time.Second)
	serialized, err := protojson.Marshal(response)
	require.NoError(t, err)
	assert.NotContains(t, string(serialized), "private-")
	_, err = ListFactoryCodexUsage(onboardingContext(r), foreignOrg.ID.String(), req)
	require.Error(t, err, "workspace must belong to the requested organization")
	for _, updates := range []map[string]any{
		{"status": models.CodexConnectionStatusReconnectRequired},
		{"status": models.CodexConnectionStatusReady, "revocation_pending_at": now},
	} {
		require.NoError(t, db.Model(connection).Updates(updates).Error)
		response, err = ListFactoryCodexUsage(onboardingContext(r), r.Organization.ID.String(), req)
		require.NoError(t, err)
		assert.Empty(t, response.Members)
	}
	require.NoError(t, db.Model(connection).Update("revocation_pending_at", nil).Error)
	require.NoError(t, db.Model(owner).Update("deleted_at", now).Error)
	response, err = ListFactoryCodexUsage(onboardingContext(r), r.Organization.ID.String(), req)
	require.NoError(t, err)
	assert.Empty(t, response.Members)
	_, err = ListFactoryCodexUsage(onboardingContext(r), r.Organization.ID.String(), &pb.ListFactoryCodexUsageRequest{FactoryId: uuid.NewString()})
	require.Error(t, err)
}
