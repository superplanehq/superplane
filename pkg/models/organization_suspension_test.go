package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestOrganizationSuspensionLifecycle(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	tx := database.DB(t.Context())
	organization, err := CreateOrganization("Suspended organization", "")
	require.NoError(t, err)
	other, err := CreateOrganization("Active organization", "")
	require.NoError(t, err)
	admin := uuid.New()

	require.ErrorIs(t, organization.Suspend(tx, admin, "  "), ErrSuspensionReasonRequired)
	require.NoError(t, organization.Suspend(tx, admin, "Unattended automation"))
	require.ErrorIs(t, RequireOrganizationActive(tx, organization.ID), ErrOrganizationSuspended)
	require.NoError(t, RequireOrganizationActive(tx, other.ID))
	require.ErrorIs(t, organization.Reactivate(tx, admin), ErrSuspensionCancellationPending)
	// Repeated requests must not replace the original audit record.
	require.NoError(t, organization.Suspend(tx, admin, "Repeated request"))
	var events []OrganizationSuspensionEvent
	require.NoError(t, tx.Where("organization_id = ?", organization.ID).Find(&events).Error)
	require.Len(t, events, 1)
	require.Equal(t, "Unattended automation", events[0].Reason)
	require.NoError(t, tx.Model(&events[0]).Update("cancellation_completed_at", time.Now()).Error)
	require.NoError(t, organization.Reactivate(tx, admin))
	require.NoError(t, RequireOrganizationActive(tx, organization.ID))
	require.NoError(t, organization.Reactivate(tx, admin))
	require.NoError(t, tx.Where("organization_id = ?", organization.ID).Order("created_at").Find(&events).Error)
	require.Len(t, events, 2)
	require.Equal(t, "reactivate", events[1].Action)
}

func TestSuspensionDoesNotWaitForStatusReaders(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	organization, err := CreateOrganization("In-flight integration", "")
	require.NoError(t, err)
	reader := db.Begin()
	require.NoError(t, reader.Error)
	defer reader.Rollback()
	require.NoError(t, RequireOrganizationActive(reader, organization.ID))
	// An integration can keep its transaction open during an external call.
	// The admin must be able to suspend it before that call returns.
	suspension := db.Begin()
	require.NoError(t, suspension.Error)
	defer suspension.Rollback()
	require.NoError(t, suspension.Exec("SET LOCAL lock_timeout = '100ms'").Error)
	require.NoError(t, organization.Suspend(suspension, uuid.New(), "Abuse"))
	require.NoError(t, suspension.Commit().Error)
}
