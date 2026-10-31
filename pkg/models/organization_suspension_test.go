package models

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func TestOrganizationSuspensionLifecycle(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	o, err := CreateOrganization("Suspension lifecycle", "")
	require.NoError(t, err)
	other, err := CreateOrganization("Other suspension", "")
	require.NoError(t, err)
	admin, reactivator := uuid.New(), uuid.New()
	status, err := FindOrganizationSuspensionStatus(db, o.ID)
	require.NoError(t, err)
	require.False(t, status.IsSuspended())
	require.Nil(t, status.SuspendedByAccountID)
	require.Empty(t, status.SuspensionReason)
	events, err := ListOrganizationSuspensionEvents(db, o.ID)
	require.NoError(t, err)
	require.Empty(t, events)
	require.NoError(t, o.Reactivate(db, admin))
	stale := *o
	require.NoError(t, o.Suspend(db, admin, "  Unattended automation\n"))
	require.True(t, o.IsSuspended())
	require.NoError(t, stale.Suspend(db, reactivator, "Another reason"))
	require.True(t, o.SuspendedAt.Equal(*stale.SuspendedAt))
	require.Equal(t, &admin, stale.SuspendedByAccountID)
	require.Equal(t, "Unattended automation", stale.SuspensionReason)
	events, err = ListOrganizationSuspensionEvents(db, o.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, admin, events[0].AccountID)
	require.Equal(t, o.ID, events[0].OrganizationID)
	require.Equal(t, OrganizationSuspensionActionSuspend, events[0].Action)
	require.Equal(t, o.SuspensionReason, events[0].Reason)
	require.True(t, o.SuspendedAt.Equal(events[0].CreatedAt))
	require.Nil(t, events[0].CancellationCompletedAt)
	require.ErrorIs(t, stale.Suspend(db, admin, " "), ErrSuspensionReasonRequired)
	require.ErrorIs(t, stale.Reactivate(db, reactivator), ErrSuspensionCancellationPending)
	status, err = FindOrganizationSuspensionStatus(db, o.ID)
	require.NoError(t, err)
	require.True(t, o.SuspendedAt.Equal(*status.SuspendedAt))
	eventsAfter, err := ListOrganizationSuspensionEvents(db, o.ID)
	require.NoError(t, err)
	require.Equal(t, events, eventsAfter)
	require.NoError(t, other.Suspend(db, admin, "Other pending cancellation"))
	require.NoError(t, db.Model(&events[0]).Update("cancellation_completed_at", time.Now()).Error)
	suspendedHandle := *o
	require.NoError(t, o.Reactivate(db, reactivator))
	require.NoError(t, suspendedHandle.Reactivate(db, admin))
	require.False(t, suspendedHandle.IsSuspended())
	require.Nil(t, suspendedHandle.SuspendedByAccountID)
	require.Empty(t, suspendedHandle.SuspensionReason)
	events, err = ListOrganizationSuspensionEvents(db, o.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, OrganizationSuspensionActionReactivate, events[0].Action)
	require.Equal(t, reactivator, events[0].AccountID)
	require.Empty(t, events[0].Reason)
	require.NotNil(t, events[1].CancellationCompletedAt)
	otherEvents, err := ListOrganizationSuspensionEvents(db, other.ID)
	require.NoError(t, err)
	require.Len(t, otherEvents, 1)
}

func TestOrganizationSuspensionReasonValidation(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	for _, tc := range []struct {
		name, reason, trimmed string
		valid                 bool
	}{
		{name: "empty"},
		{name: "whitespace", reason: " \n\t\u2003"},
		{name: "one character", reason: " x ", trimmed: "x", valid: true},
		{name: "2000 characters", reason: strings.Repeat("x", 2000), trimmed: strings.Repeat("x", 2000), valid: true},
		{name: "2001 characters", reason: strings.Repeat("x", 2001)},
		{name: "2000 Unicode characters", reason: "\u2003" + strings.Repeat("界", 2000) + "\n", trimmed: strings.Repeat("界", 2000), valid: true},
		{name: "2001 Unicode characters", reason: strings.Repeat("界", 2001)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, err := CreateOrganization(tc.name, "")
			require.NoError(t, err)
			err = o.Suspend(db, uuid.New(), tc.reason)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrSuspensionReasonRequired)
			}
			status, err := FindOrganizationSuspensionStatus(db, o.ID)
			require.NoError(t, err)
			require.Equal(t, tc.valid, status.IsSuspended())
			require.Equal(t, tc.trimmed, status.SuspensionReason)
			events, err := ListOrganizationSuspensionEvents(db, o.ID)
			require.NoError(t, err)
			if !tc.valid {
				require.Nil(t, status.SuspendedByAccountID)
				require.Empty(t, events)
				return
			}
			require.Len(t, events, 1)
			require.Equal(t, tc.trimmed, events[0].Reason)
		})
	}
}

func TestOrganizationSuspensionRollback(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	for _, action := range []string{"suspend", "reactivate"} {
		for _, failure := range []string{"audit insertion", "caller rollback"} {
			t.Run(action+"/"+failure, func(t *testing.T) {
				o, err := CreateOrganization(action+failure, "")
				require.NoError(t, err)
				admin := uuid.New()
				if action == "reactivate" {
					require.NoError(t, o.Suspend(db, admin, "Abuse"))
					require.NoError(t, db.Model(&OrganizationSuspensionEvent{}).Where("organization_id = ?", o.ID).
						Update("cancellation_completed_at", time.Now()).Error)
				}
				before, err := FindOrganizationSuspensionStatus(db, o.ID)
				require.NoError(t, err)
				historyBefore, err := ListOrganizationSuspensionEvents(db, o.ID)
				require.NoError(t, err)
				outer := db.Begin()
				require.NoError(t, outer.Error)
				defer outer.Rollback()
				if failure == "audit insertion" {
					// Fail the event insert after the state update using a real constraint.
					require.NoError(t, outer.Exec("ALTER TABLE organization_suspension_events ADD CONSTRAINT reject_test_transition CHECK (action <> '"+action+"') NOT VALID").Error)
				}
				err = suspensionTestTransition(outer, o, action, admin)
				if failure == "audit insertion" {
					require.ErrorContains(t, err, "reject_test_transition")
					inside, readErr := FindOrganizationSuspensionStatus(outer, o.ID)
					require.NoError(t, readErr)
					require.Equal(t, before, inside)
					insideHistory, readErr := ListOrganizationSuspensionEvents(outer, o.ID)
					require.NoError(t, readErr)
					require.Equal(t, historyBefore, insideHistory)
				} else {
					require.NoError(t, err)
					inside, readErr := FindOrganizationSuspensionStatus(outer, o.ID)
					require.NoError(t, readErr)
					require.Equal(t, action == "suspend", inside.IsSuspended())
					insideHistory, readErr := ListOrganizationSuspensionEvents(outer, o.ID)
					require.NoError(t, readErr)
					require.Len(t, insideHistory, len(historyBefore)+1)
				}
				require.NoError(t, outer.Rollback().Error)
				after, err := FindOrganizationSuspensionStatus(db, o.ID)
				require.NoError(t, err)
				require.Equal(t, before, after)
				historyAfter, err := ListOrganizationSuspensionEvents(db, o.ID)
				require.NoError(t, err)
				require.Equal(t, historyBefore, historyAfter)
			})
		}
	}
}

func TestOrganizationSuspensionReadsDoNotLock(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	o, err := CreateOrganization("Suspension reads", "")
	require.NoError(t, err)
	reader := db.Begin()
	require.NoError(t, reader.Error)
	defer reader.Rollback()
	_, err = FindOrganizationSuspensionStatus(reader, o.ID)
	require.NoError(t, err)
	_, err = ListOrganizationSuspensionEvents(reader, o.ID)
	require.NoError(t, err)
	writer := db.Begin()
	require.NoError(t, writer.Error)
	defer writer.Rollback()
	require.NoError(t, writer.Exec("SET LOCAL lock_timeout = '500ms'").Error)
	require.NoError(t, o.Suspend(writer, uuid.New(), "Abuse"))
	require.NoError(t, writer.Commit().Error)
}

func TestOrganizationSuspensionConcurrentTransitions(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	for _, tc := range []struct {
		name, first, second string
		initialSuspended    bool
		expectedActions     []string
		pendingError        bool
	}{
		{"duplicate suspend", "suspend", "suspend", false, []string{"suspend"}, false},
		{"duplicate reactivate", "reactivate", "reactivate", true, []string{"reactivate", "suspend"}, false},
		{"suspend then reactivate", "suspend", "reactivate", false, []string{"suspend"}, true},
		{"reactivate then suspend", "reactivate", "suspend", true, []string{"suspend", "reactivate", "suspend"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, err := CreateOrganization(tc.name, "")
			require.NoError(t, err)
			if tc.initialSuspended {
				require.NoError(t, o.Suspend(db, uuid.New(), "Initial suspension"))
				require.NoError(t, db.Model(&OrganizationSuspensionEvent{}).Where("organization_id = ?", o.ID).
					Update("cancellation_completed_at", time.Now()).Error)
			}
			first := db.Begin()
			require.NoError(t, first.Error)
			defer first.Rollback()
			firstHandle, secondHandle := *o, *o
			require.NoError(t, suspensionTestTransition(first, &firstHandle, tc.first, uuid.New()))
			second := db.Begin()
			require.NoError(t, second.Error)
			defer second.Rollback()
			require.NoError(t, second.Exec("SET LOCAL lock_timeout = '5s'").Error)
			var pid int
			require.NoError(t, second.Raw("SELECT pg_backend_pid()").Scan(&pid).Error)
			result := make(chan error, 1)
			go func() {
				err := suspensionTestTransition(second, &secondHandle, tc.second, uuid.New())
				if err != nil {
					second.Rollback()
					result <- err
					return
				}
				result <- second.Commit().Error
			}()
			// Observe the database lock wait, rather than assume goroutine timing.
			require.Eventually(t, func() bool {
				var waiting bool
				queryErr := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pid = ? AND wait_event_type = 'Lock')", pid).Scan(&waiting).Error
				return queryErr == nil && waiting
			}, 2*time.Second, 10*time.Millisecond)
			require.NoError(t, first.Commit().Error)
			select {
			case err := <-result:
				if tc.pendingError {
					require.ErrorIs(t, err, ErrSuspensionCancellationPending)
				} else {
					require.NoError(t, err)
				}
			case <-time.After(6 * time.Second):
				t.Fatal("concurrent transition did not finish")
			}
			events, err := ListOrganizationSuspensionEvents(db, o.ID)
			require.NoError(t, err)
			actions := make([]string, len(events))
			for i, event := range events {
				actions[i] = event.Action
			}
			require.Equal(t, tc.expectedActions, actions)
			status, err := FindOrganizationSuspensionStatus(db, o.ID)
			require.NoError(t, err)
			require.Equal(t, events[0].Action == "suspend", status.IsSuspended())
		})
	}
}

func suspensionTestTransition(tx *gorm.DB, o *Organization, action string, admin uuid.UUID) error {
	if action == "suspend" {
		return o.Suspend(tx, admin, "Abuse")
	}
	return o.Reactivate(tx, admin)
}

func TestOrganizationSuspensionHistoryRetainsAllEvents(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	o, err := CreateOrganization("Full suspension history", "")
	require.NoError(t, err)
	admin := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	// Timestamp ties must have a stable order, and reads must retain more than
	// the prototype's 100 events. Historical pending events also block reactivation.
	events := make([]OrganizationSuspensionEvent, 101)
	for i := range events {
		events[i] = OrganizationSuspensionEvent{
			ID: uuid.New(), OrganizationID: o.ID, AccountID: admin,
			Action: "suspend", Reason: "Historical suspension", CreatedAt: now,
			CancellationCompletedAt: &now,
		}
	}
	events[0].CancellationCompletedAt = nil
	require.NoError(t, db.Create(&events).Error)
	require.NoError(t, o.Suspend(db, admin, "Latest suspension"))
	require.NoError(t, db.Model(&OrganizationSuspensionEvent{}).
		Where("organization_id = ? AND id <> ?", o.ID, events[0].ID).
		Update("cancellation_completed_at", now).Error)
	require.ErrorIs(t, o.Reactivate(db, admin), ErrSuspensionCancellationPending)
	history, err := ListOrganizationSuspensionEvents(db, o.ID)
	require.NoError(t, err)
	require.Len(t, history, 102)
	for i := 2; i < len(history); i++ {
		require.Greater(t, history[i-1].ID.String(), history[i].ID.String())
	}
	require.NoError(t, db.Model(&events[0]).Update("cancellation_completed_at", now).Error)
	require.NoError(t, o.Reactivate(db, admin))
	history, err = ListOrganizationSuspensionEvents(db, o.ID)
	require.NoError(t, err)
	require.Len(t, history, 103)
}
