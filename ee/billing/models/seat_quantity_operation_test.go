package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestDeletingTheOnlyActorTokenKeepsTheSeatOperation(t *testing.T) {
	r := support.Setup(t)
	t.Cleanup(r.Close)
	db := database.Conn()
	token := models.NewUserAPIToken(r.User, "billing", "hash-"+uuid.NewString())
	require.NoError(t, models.CreateUserAPIToken(db, token))

	op := seatOperation(r.Organization.ID, billingmodels.SeatQuantityOriginAdmin, billingmodels.SeatQuantityStatusScheduled)
	op.ActorAPIKeyID = &token.ID
	require.NoError(t, op.Create(db))

	require.NoError(t, token.HardDelete(db))

	stored, err := billingmodels.FindSeatQuantityOperation(db, r.Organization.ID, op.ID)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Nil(t, stored.ActorAPIKeyID)
	assert.Nil(t, stored.ActorUserID)
	assert.Equal(t, billingmodels.SeatQuantityStatusScheduled, stored.Status)

	withoutActor := seatOperation(r.Organization.ID, billingmodels.SeatQuantityOriginAdmin, billingmodels.SeatQuantityStatusCanceled)
	require.Error(t, withoutActor.Create(db))

	protective := seatOperation(r.Organization.ID, billingmodels.SeatQuantityOriginProtective, billingmodels.SeatQuantityStatusCanceled)
	require.NoError(t, protective.Create(db))
}

func seatOperation(orgID uuid.UUID, origin, status string) *billingmodels.SeatQuantityOperation {
	now := time.Now().UTC().Truncate(time.Second)
	end := now.Add(24 * time.Hour)
	return &billingmodels.SeatQuantityOperation{
		OrganizationID: orgID, Origin: origin, Action: billingmodels.SeatQuantityActionSchedule,
		SubscriptionID: "sub_team", PeriodStart: now.Add(-time.Hour), PeriodEnd: end,
		SourcePaidQuantity: 10, TargetQuantity: 7, EffectiveAt: end,
		OperationKey: uuid.NewString(), RequestFingerprint: "fingerprint",
		Status: status, SeatChangeVersion: 1,
	}
}
