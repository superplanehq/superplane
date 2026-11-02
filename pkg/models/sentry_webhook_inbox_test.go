package models_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func createInboxEntry(t *testing.T, db *gorm.DB, resource, installationUUID string, receivedAt time.Time) models.SentryWebhookInbox {
	t.Helper()

	entry := models.NewSentryWebhookInbox(models.SentryWebhookSourceHostedApp, resource, receivedAt)
	entry.InstallationUUID = installationUUID
	entry.PayloadCiphertext = []byte("sealed")
	entry.TargetsResolved = true
	receipt := models.SentryWebhookReceipt{ReceivedAt: receivedAt, HookResource: resource, Outcome: models.SentryWebhookStateQueued, HTTPStatus: 200}
	require.NoError(t, models.CreateSentryWebhookDelivery(db, receipt, &entry, nil))
	return entry
}

func claim(t *testing.T, db *gorm.DB, owner uuid.UUID, now time.Time, lease time.Duration, limit int) []models.SentryWebhookInbox {
	t.Helper()

	var claimed []models.SentryWebhookInbox
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		claimed, err = models.ClaimSentryWebhookInbox(tx, owner, now, lease, limit)
		return err
	}))
	return claimed
}

func claimedIDs(entries []models.SentryWebhookInbox) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func Test__ClaimSentryWebhookInbox(t *testing.T) {
	db := database.DB(t.Context())

	t.Run("concurrent workers never claim the same entry", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		now := time.Now()
		for i := 0; i < 30; i++ {
			createInboxEntry(t, db, "issue", "install-1", now.Add(-time.Duration(i)*time.Second))
		}

		var mu sync.Mutex
		seen := map[uuid.UUID]int{}
		var wg sync.WaitGroup
		for worker := 0; worker < 4; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				claimed := claim(t, db, uuid.New(), now, time.Minute, 10)
				mu.Lock()
				defer mu.Unlock()
				for _, id := range claimedIDs(claimed) {
					seen[id]++
				}
			}()
		}
		wg.Wait()

		assert.Len(t, seen, 30)
		for id, count := range seen {
			assert.Equal(t, 1, count, "entry %s was claimed more than once", id)
		}
	})

	t.Run("an expired lease can be claimed again and the old owner cannot commit", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		now := time.Now()
		createInboxEntry(t, db, "issue", "install-1", now)

		first := claim(t, db, uuid.New(), now, time.Minute, 25)
		require.Len(t, first, 1)
		assert.Empty(t, claim(t, db, uuid.New(), now, time.Minute, 25))

		later := now.Add(2 * time.Minute)
		second := claim(t, db, uuid.New(), later, time.Minute, 25)
		require.Len(t, second, 1)
		assert.Equal(t, 2, second[0].Attempts)

		err := db.Transaction(func(tx *gorm.DB) error {
			return first[0].LockLease(tx, later)
		})
		assert.ErrorIs(t, err, models.ErrSentryWebhookLeaseLost)

		held, err := first[0].RenewLease(db, later, time.Minute)
		require.NoError(t, err)
		assert.False(t, held)

		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return second[0].LockLease(tx, later)
		}))
	})

	t.Run("installation events run first and one at a time per installation", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		now := time.Now()
		issue := createInboxEntry(t, db, "issue", "install-1", now.Add(-time.Hour))
		created := createInboxEntry(t, db, "installation", "install-1", now.Add(-time.Minute))
		deleted := createInboxEntry(t, db, "installation", "install-1", now)

		claimed := claim(t, db, uuid.New(), now, time.Minute, 1)
		require.Len(t, claimed, 1)
		assert.Equal(t, created.ID, claimed[0].ID)

		next := claim(t, db, uuid.New(), now, time.Minute, 25)
		assert.Equal(t, []uuid.UUID{issue.ID}, claimedIDs(next), "the deletion waits while the creation holds a lease")

		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			if err := claimed[0].LockLease(tx, now); err != nil {
				return err
			}
			return claimed[0].Complete(tx, models.SentryWebhookStateProcessed, "", now)
		}))

		afterCreation := claim(t, db, uuid.New(), now, time.Minute, 25)
		assert.Equal(t, []uuid.UUID{deleted.ID}, claimedIDs(afterCreation))
	})

	t.Run("a rescheduled entry waits for its next attempt", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())
		now := time.Now()
		entry := createInboxEntry(t, db, "issue", "install-1", now)

		claimed := claim(t, db, uuid.New(), now, time.Minute, 25)
		require.Len(t, claimed, 1)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return claimed[0].Reschedule(tx, models.SentryWebhookStateBlocked, "no connection", now.Add(time.Minute), now)
		}))

		assert.Empty(t, claim(t, db, uuid.New(), now.Add(30*time.Second), time.Minute, 25))
		again := claim(t, db, uuid.New(), now.Add(2*time.Minute), time.Minute, 25)
		assert.Equal(t, []uuid.UUID{entry.ID}, claimedIDs(again))

		var receipt models.SentryWebhookReceipt
		require.NoError(t, db.Where("id = ?", *entry.ReceiptID).First(&receipt).Error)
		assert.Equal(t, models.SentryWebhookStateProcessing, receipt.Outcome)
		assert.Equal(t, "no connection", receipt.ProcessingError)
		assert.Equal(t, 200, receipt.HTTPStatus)
	})
}

func Test__SentryWebhookInboxCleanup(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	old := time.Now().Add(-30 * 24 * time.Hour)
	cutoff := time.Now().Add(-models.SentryWebhookInboxRetention)

	unfinished := createInboxEntry(t, db, "issue", "install-1", old)
	finished := createInboxEntry(t, db, "issue", "install-1", old)
	claimed := claim(t, db, uuid.New(), time.Now(), time.Minute, 25)
	require.Len(t, claimed, 2)
	for i := range claimed {
		if claimed[i].ID != finished.ID {
			continue
		}
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return claimed[i].Complete(tx, models.SentryWebhookStateProcessed, "", old)
		}))
	}

	deleted, err := models.DeleteExpiredSentryWebhookInbox(db, cutoff, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	deletedReceipts, err := models.DeleteExpiredSentryWebhookReceipts(db, cutoff, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deletedReceipts, "only the receipt of the finished delivery is deleted")

	var remaining models.SentryWebhookInbox
	require.NoError(t, db.Where("id = ?", unfinished.ID).First(&remaining).Error)
	assert.Equal(t, []byte("sealed"), remaining.PayloadCiphertext)
	require.NotNil(t, remaining.ReceiptID)

	var receipt models.SentryWebhookReceipt
	require.NoError(t, db.Where("id = ?", *remaining.ReceiptID).First(&receipt).Error)
}

func Test__FindSentryWebhookInboxStats(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	now := time.Now()
	createInboxEntry(t, db, "issue", "install-1", now.Add(-10*time.Minute))
	blocked := createInboxEntry(t, db, "issue", "install-1", now.Add(-time.Minute))
	require.NoError(t, db.Model(&models.SentryWebhookInbox{}).Where("id = ?", blocked.ID).Update("state", models.SentryWebhookStateBlocked).Error)

	stats, err := models.FindSentryWebhookInboxStats(db, now)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Unfinished)
	assert.Equal(t, int64(1), stats.Blocked)
	assert.InDelta(t, (10 * time.Minute).Seconds(), stats.OldestAge.Seconds(), 1)
}

func Test__SentryWebhookInboxPayloadRotation(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	entry := createInboxEntry(t, db, "issue", "install-1", time.Now())

	page, err := models.ReadSentryWebhookInboxPayloadPage(db, "", 10)
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, entry.ID.String(), page[0].AssociatedData)

	replaced, err := models.ReplaceSentryWebhookInboxPayload(db, page[0], []byte("rotated"))
	require.NoError(t, err)
	assert.True(t, replaced)

	stale, err := models.ReplaceSentryWebhookInboxPayload(db, page[0], []byte("stale"))
	require.NoError(t, err)
	assert.False(t, stale, "a replacement that does not match the stored value changes nothing")

	var stored models.SentryWebhookInbox
	require.NoError(t, db.Where("id = ?", entry.ID).First(&stored).Error)
	assert.Equal(t, []byte("rotated"), stored.PayloadCiphertext)
}

func Test__SentryAppInstallationDeletionBlocksGrants(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	db := database.DB(t.Context())
	now := time.Now()
	grant := models.SentryAppInstallGrant{
		InstallationUUID: "install-1",
		CodeDigest:       "digest-1",
		AccessToken:      []byte("sealed-access"),
		ExpiresAt:        now.Add(20 * time.Minute),
	}

	t.Run("a retry of the same grant without tokens keeps the stored tokens", func(t *testing.T) {
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, grant))
		withoutTokens := grant
		withoutTokens.AccessToken = nil
		require.NoError(t, models.UpsertSentryAppInstallGrant(db, withoutTokens))

		found, err := models.FindSentryAppInstallGrant(db, "install-1", "digest-1", now)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, []byte("sealed-access"), found.AccessToken)
	})

	t.Run("a deleted installation refuses a delayed grant", func(t *testing.T) {
		require.NoError(t, models.ForgetSentryAppInstallation(db, "install-1", now))

		err := models.UpsertSentryAppInstallGrant(db, grant)
		assert.ErrorIs(t, err, models.ErrSentryAppInstallationDeleted)

		found, err := models.FindSentryAppInstallGrant(db, "install-1", "digest-1", now)
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}
