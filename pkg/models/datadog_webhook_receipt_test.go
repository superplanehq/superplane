package models

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestCreateRejectedDatadogWebhookReceiptSkipsLockWhenFull(t *testing.T) {
	integrationID := uuid.New()
	organizationID := uuid.New()
	since := time.Now().UTC().Add(-time.Hour)
	for range 10 {
		_, err := CreateDatadogWebhookReceipt(database.Conn(), DatadogWebhookReceipt{
			IntegrationID:  integrationID,
			OrganizationID: organizationID,
			HTTPStatus:     403,
			Outcome:        DatadogWebhookOutcomeRejected,
		})
		require.NoError(t, err)
	}

	blocker := database.Conn().Begin()
	require.NoError(t, blocker.Error)
	t.Cleanup(func() { blocker.Rollback() })
	require.NoError(t, lockRejectedDatadogWebhookReceipts(blocker, integrationID))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	started := time.Now()
	_, stored, oldest, err := CreateRejectedDatadogWebhookReceiptIfAllowed(
		database.DB(ctx),
		DatadogWebhookReceipt{
			IntegrationID:  integrationID,
			OrganizationID: organizationID,
			HTTPStatus:     403,
		},
		since,
		10,
	)
	elapsed := time.Since(started)

	require.NoError(t, err)
	assert.False(t, stored)
	assert.False(t, oldest.IsZero())
	assert.Less(t, elapsed, time.Second)
}
