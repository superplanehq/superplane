package telemetry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestBeaconEnabledForRealInstallations(t *testing.T) {
	tests := []struct {
		name    string
		enabled string
		kind    string
		wantOn  bool
	}{
		{name: "kubernetes", enabled: "yes", kind: "kubernetes", wantOn: true},
		{name: "single host", enabled: "yes", kind: "single-host", wantOn: true},
		{name: "custom install", enabled: "yes", kind: "on-prem", wantOn: true},
		{name: "development", enabled: "yes", kind: "development", wantOn: false},
		{name: "demo", enabled: "yes", kind: "demo", wantOn: false},
		{name: "missing type", enabled: "yes", kind: "", wantOn: false},
		{name: "flag off", enabled: "no", kind: "kubernetes", wantOn: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(beaconEnabledEnv, test.enabled)
			t.Setenv(installationTypeEnv, test.kind)
			assert.Equal(t, test.wantOn, BeaconEnabled())
		})
	}
}

func TestBuildBeaconPayload(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.Conn()
	t.Setenv(beaconVersionEnv, "")

	organization, err := models.CreateOrganizationInTransaction(db, "Acme", "")
	require.NoError(t, err)
	account, err := models.CreateAccountInTransaction(db, "Ada Lovelace", "ada@example.com")
	require.NoError(t, err)
	require.NoError(t, db.Model(account).Update("installation_admin", true).Error)
	user, err := models.CreateUserInTransaction(db, organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)

	factoryModel, err := models.CreateFactory(db, organization.ID, "factory", "", "")
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	_, err = factoryModel.CreateWorkOrder(db, "Created", "", &user.ID, nil, nil)
	require.NoError(t, err)

	started, err := factoryModel.CreateWorkOrder(db, "Started", "", &user.ID, nil, nil)
	require.NoError(t, err)
	insertBeaconDispatch(t, db, organization.ID, factoryModel.ID, started.ID, line)
	insertBeaconDispatch(t, db, organization.ID, factoryModel.ID, started.ID, line)

	finished, err := factoryModel.CreateWorkOrder(db, "Finished", "", &user.ID, nil, nil)
	require.NoError(t, err)
	insertBeaconDispatch(t, db, organization.ID, factoryModel.ID, finished.ID, line)
	_, err = finished.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	_, err = finished.Close(db, models.FactoryWorkOrderResultCompleted, &user.ID)
	require.NoError(t, err)

	rejected, err := factoryModel.CreateWorkOrder(db, "Rejected", "", &user.ID, nil, nil)
	require.NoError(t, err)
	insertBeaconDispatch(t, db, organization.ID, factoryModel.ID, rejected.ID, line)
	_, err = rejected.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultRejected,
	})
	require.NoError(t, err)

	stale, err := factoryModel.CreateWorkOrder(db, "Stale", "", &user.ID, nil, nil)
	require.NoError(t, err)
	insertBeaconDispatch(t, db, organization.ID, factoryModel.ID, stale.ID, line)
	_, err = stale.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	_, err = stale.Close(db, models.FactoryWorkOrderResultCompleted, &user.ID)
	require.NoError(t, err)
	outsideWindow := time.Now().Add(-8 * 24 * time.Hour)
	require.NoError(t, db.Model(stale).Update("created_at", outsideWindow).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrderLineDispatch{}).
		Where("work_order_id = ?", stale.ID).
		Update("created_at", outsideWindow).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrderEvent{}).
		Where("work_order_id = ?", stale.ID).
		Update("created_at", outsideWindow).Error)

	since := time.Now().Add(-7 * 24 * time.Hour)
	payload, err := buildBeaconPayload(db, "kubernetes", since)
	require.NoError(t, err)
	assert.Equal(t, "kubernetes", payload.InstallationType)
	_, err = uuid.Parse(payload.InstallationID)
	require.NoError(t, err)
	again, err := buildBeaconPayload(db, "kubernetes", since)
	require.NoError(t, err)
	assert.Equal(t, payload.InstallationID, again.InstallationID)
	assert.False(t, payload.InstalledAt.IsZero())
	assert.Equal(t, developmentVersion, payload.Version)
	assert.Equal(t, int64(4), payload.TasksCreated7d)
	assert.Equal(t, int64(3), payload.TasksStarted7d)
	assert.Equal(t, int64(1), payload.TasksFinished7d)
	assert.Empty(t, payload.OwnerEmail)
	assert.Empty(t, payload.OrganizationName)

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(body, &wire))
	assert.Contains(t, wire, "installed_at")
	assert.Contains(t, wire, "version")
	assert.NotContains(t, wire, "owner_email")
	assert.NotContains(t, wire, "organization_name")

	require.NoError(t, models.SetShareOwnerIdentity(db, true))
	t.Setenv(beaconVersionEnv, "v1.0.0-rc2")
	shared, err := buildBeaconPayload(db, "kubernetes", since)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0-rc2", shared.Version)
	assert.Equal(t, account.Email, shared.OwnerEmail)
	assert.Equal(t, organization.Name, shared.OrganizationName)

	require.NoError(t, models.SetShareOwnerIdentity(db, false))
	cleared, err := buildBeaconPayload(db, "kubernetes", since)
	require.NoError(t, err)
	assert.Empty(t, cleared.OwnerEmail)
	assert.Empty(t, cleared.OrganizationName)
}

func insertBeaconDispatch(
	t *testing.T,
	db *gorm.DB,
	organizationID uuid.UUID,
	factoryID uuid.UUID,
	orderID uuid.UUID,
	line *models.FactoryLine,
) {
	t.Helper()
	now := time.Now()
	row := models.FactoryWorkOrderLineDispatch{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		WorkOrderID:    orderID,
		LineID:         line.ID,
		LineName:       line.Name,
		Steps:          datatypes.JSONSlice[models.FactoryLineStep]{},
		State:          models.FactoryWorkOrderLineDispatchStateFinished,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, db.Create(&row).Error)
}
