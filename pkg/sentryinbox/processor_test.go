package sentryinbox_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/sentryinbox"
	"github.com/superplanehq/superplane/test/support"
)

func TestAccept(t *testing.T) {
	t.Run("commits the receipt, the encrypted payload, and the known targets together", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})

		accepted := acceptHosted(t, "issue", issueBody)

		entry := reload(t, accepted.InboxID)
		assert.Equal(t, models.SentryWebhookStateQueued, entry.State)
		assert.NotContains(t, string(entry.PayloadCiphertext), `"install-1"`)
		plain, err := sealingEncryptor{}.Decrypt(t.Context(), entry.PayloadCiphertext, []byte(entry.ID.String()))
		require.NoError(t, err)
		assert.JSONEq(t, issueBody, string(plain))

		receipt := receiptOf(t, entry)
		assert.Equal(t, models.SentryWebhookStateQueued, receipt.Outcome)
		assert.Equal(t, 200, receipt.HTTPStatus)
		assert.Equal(t, 1, receipt.IntegrationCount)

		targets := targetsOf(t, entry)
		require.Len(t, targets, 1)
		assert.Equal(t, node.WorkflowID, targets[0].WorkflowID)
		assert.Empty(t, eventsFor(t, node), "acceptance does not run the trigger")
	})

	t.Run("an encryption failure stores nothing", func(t *testing.T) {
		support.Setup(t)

		_, err := sentryinbox.Accept(t.Context(), database.DB(t.Context()), failingEncryptor{}, sentryinbox.Delivery{
			Source:   models.SentryWebhookSourceHostedApp,
			Resource: "issue",
			Body:     []byte(issueBody),
		})
		require.Error(t, err)

		var entries, receipts int64
		require.NoError(t, database.Conn().Model(&models.SentryWebhookInbox{}).Count(&entries).Error)
		require.NoError(t, database.Conn().Model(&models.SentryWebhookReceipt{}).Count(&receipts).Error)
		assert.Zero(t, entries)
		assert.Zero(t, receipts)
	})

	t.Run("every incoming request is stored, also with the same body", func(t *testing.T) {
		support.Setup(t)

		first := acceptHosted(t, "issue", issueBody)
		second := acceptHosted(t, "issue", issueBody)

		assert.NotEqual(t, first.InboxID, second.InboxID)
	})
}

func TestProcessorDeliversIssueEvents(t *testing.T) {
	t.Run("one attempt emits one event and removes the payload", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})
		accepted := acceptHosted(t, "issue", issueBody)

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)

		events := eventsFor(t, node)
		require.Len(t, events, 1)
		assert.Equal(t, models.CanvasEventStatePending, events[0].State)
		require.NotNil(t, events[0].DedupKey)
		assert.Equal(t, "sentry-delivery:"+accepted.InboxID.String(), *events[0].DedupKey)

		entry := reload(t, accepted.InboxID)
		assert.Equal(t, models.SentryWebhookStateProcessed, entry.State)
		assert.Nil(t, entry.PayloadCiphertext)
		assert.NotNil(t, entry.CompletedAt)
		assert.Equal(t, models.SentryWebhookStateProcessed, receiptOf(t, entry).Outcome)
	})

	t.Run("a failed optional enrichment keeps the event and records a warning", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})
		accepted := acceptHosted(t, "issue", issueBody)

		_, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)

		require.Len(t, eventsFor(t, node), 1)
		targets := targetsOf(t, reload(t, accepted.InboxID))
		require.Len(t, targets, 1)
		assert.Equal(t, models.SentryWebhookTargetProcessed, targets[0].State)
		assert.Contains(t, targets[0].LastError, "issue enrichment")
	})

	t.Run("a RabbitMQ outage delays the notification but keeps the event pending", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})
		accepted := acceptHosted(t, "issue", issueBody)

		processor := newProcessor(r).WithEventPublisher(func(*models.CanvasEvent) error {
			return assert.AnError
		})
		result, err := processor.Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)

		pending, err := models.ListPendingCanvasEvents()
		require.NoError(t, err)
		events := eventsFor(t, node)
		require.Len(t, events, 1)
		assert.Contains(t, eventIDs(pending), events[0].ID.String(), "the event router polls the stored event")
	})
}

func TestProcessorRecovery(t *testing.T) {
	t.Run("a crash before the event commit runs the target again", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})
		accepted := acceptHosted(t, "issue", issueBody)

		claimEntry(t, accepted.InboxID, time.Millisecond)
		time.Sleep(10 * time.Millisecond)

		_, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Len(t, eventsFor(t, node), 1)
	})

	t.Run("a crash after the event commit does not repeat the event", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})
		accepted := acceptHosted(t, "issue", issueBody)

		crashing := newProcessor(r).WithEventPublisher(func(*models.CanvasEvent) error {
			expireLease(t, accepted.InboxID)
			return nil
		})
		_, err := crashing.Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.ErrorIs(t, err, models.ErrSentryWebhookLeaseLost)
		require.Len(t, eventsFor(t, node), 1)
		assert.Nil(t, reload(t, accepted.InboxID).CompletedAt)

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)
		assert.Len(t, eventsFor(t, node), 1)
	})

	t.Run("a worker with an expired lease cannot commit", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})
		accepted := acceptHosted(t, "issue", issueBody)

		stale := claimEntry(t, accepted.InboxID, time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		current := claimEntry(t, accepted.InboxID, time.Minute)

		_, err := newProcessor(r).Process(t.Context(), stale)
		require.ErrorIs(t, err, models.ErrSentryWebhookLeaseLost)
		assert.Empty(t, eventsFor(t, node))

		_, err = newProcessor(r).Process(t.Context(), current)
		require.NoError(t, err)
		assert.Len(t, eventsFor(t, node), 1)
	})

	t.Run("a failed target does not repeat a finished target", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, ready := createListener(t, r, integration, map[string]any{})
		unpublished, waiting := createListener(t, r, integration, map[string]any{})
		liveVersionID := *unpublished.LiveVersionID
		other, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		require.NoError(t, database.Conn().Model(&models.Canvas{}).Where("id = ?", unpublished.ID).
			Update("live_version_id", *other.LiveVersionID).Error)
		accepted := acceptHosted(t, "issue", issueBody)

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateBlocked, result.State)
		assert.Contains(t, result.Note, "no live version")
		assert.Len(t, eventsFor(t, ready), 1)
		assert.Empty(t, eventsFor(t, waiting))
		assert.NotNil(t, reload(t, accepted.InboxID).PayloadCiphertext)

		require.NoError(t, database.Conn().Model(&models.Canvas{}).Where("id = ?", unpublished.ID).
			Update("live_version_id", liveVersionID).Error)
		makeDue(t, accepted.InboxID)

		result, err = newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)
		assert.Len(t, eventsFor(t, ready), 1)
		assert.Len(t, eventsFor(t, waiting), 1)
	})
}

func TestProcessorExplicitOutcomes(t *testing.T) {
	t.Run("a missing connection stays blocked until a connection exists", func(t *testing.T) {
		r := support.Setup(t)
		accepted := acceptHosted(t, "issue", issueBody)

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateBlocked, result.State)
		entry := reload(t, accepted.InboxID)
		assert.NotNil(t, entry.PayloadCiphertext)
		assert.True(t, entry.NextAttemptAt.After(time.Now()))
		receipt := receiptOf(t, entry)
		assert.Equal(t, models.SentryWebhookStateBlocked, receipt.Outcome)
		assert.Contains(t, receipt.ProcessingError, "install-1")

		integration := createHostedConnection(t, r, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})
		makeDue(t, accepted.InboxID)

		result, err = newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)
		assert.Len(t, eventsFor(t, node), 1)
	})

	t.Run("a delivery for a deleted installation finishes as filtered", func(t *testing.T) {
		r := support.Setup(t)
		require.NoError(t, models.ForgetSentryAppInstallation(database.Conn(), "install-1", time.Now()))
		accepted := acceptHosted(t, "issue", issueBody)

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateFiltered, result.State)
		assert.Nil(t, reload(t, accepted.InboxID).PayloadCiphertext)
	})

	t.Run("an unsupported payload stays blocked with its payload", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		createListener(t, r, integration, map[string]any{})
		accepted := acceptHosted(t, "issue", `{"action":"created","installation":{"uuid":"install-1"},"data":"not-an-object"}`)

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateBlocked, result.State)
		assert.Contains(t, result.Note, "unsupported")
		assert.NotNil(t, reload(t, accepted.InboxID).PayloadCiphertext)
	})

	t.Run("a configured action filter finishes the delivery as filtered", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{"actions": []string{"resolved"}})
		accepted := acceptHosted(t, "issue", issueBody)

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateFiltered, result.State)
		assert.Contains(t, result.Note, "action")
		assert.Empty(t, eventsFor(t, node))
		assert.Nil(t, reload(t, accepted.InboxID).PayloadCiphertext)
	})

	t.Run("a removed trigger finishes its target with an explicit outcome", func(t *testing.T) {
		r, integration := setupHostedConnection(t, "install-1")
		_, node := createListener(t, r, integration, map[string]any{})
		accepted := acceptHosted(t, "issue", issueBody)
		require.NoError(t, models.DeleteIntegrationSubscriptionsForNodeInTransaction(database.Conn(), node.WorkflowID, node.NodeID))

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateFiltered, result.State)
		targets := targetsOf(t, reload(t, accepted.InboxID))
		require.Len(t, targets, 1)
		assert.Equal(t, models.SentryWebhookTargetRemoved, targets[0].State)
	})
}

func TestRetryDelay(t *testing.T) {
	noJitter := func() float64 { return 0 }
	fullJitter := func() float64 { return 1 }

	assert.Equal(t, 5*time.Second, sentryinbox.RetryDelay(1, noJitter))
	assert.Equal(t, 10*time.Second, sentryinbox.RetryDelay(2, noJitter))
	assert.Equal(t, 6*time.Second, sentryinbox.RetryDelay(1, fullJitter))
	assert.Equal(t, 5*time.Minute, sentryinbox.RetryDelay(50, noJitter))
	assert.Equal(t, 5*time.Minute, sentryinbox.RetryDelay(1000, fullJitter))
}

func eventIDs(events []models.CanvasEvent) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID.String())
	}
	return ids
}
