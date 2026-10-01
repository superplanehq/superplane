package factories

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

func TestArchiveDraftWorkOrdersFromGitHubIssueClosed(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	type intakeWebhook struct {
		factory *models.Factory
		webhook *models.Webhook
		secret  []byte
	}

	newIntakeWebhook := func(t *testing.T) intakeWebhook {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas := support.CreateFactoryCanvas(t, r, factory.ID, "intake")
		_, err = factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		webhookID := uuid.New()
		secret := []byte("webhook-secret")
		encrypted, err := r.Encryptor.Encrypt(t.Context(), secret, []byte(webhookID.String()))
		require.NoError(t, err)
		now := time.Now()
		webhook := &models.Webhook{
			ID:            webhookID,
			State:         models.WebhookStateReady,
			Secret:        encrypted,
			Configuration: datatypes.NewJSONType[any](map[string]any{}),
			Metadata:      datatypes.NewJSONType[any](map[string]any{}),
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		require.NoError(t, db.Create(webhook).Error)
		require.NoError(t, db.Create(&models.CanvasNode{
			WorkflowID:    canvas.ID,
			NodeID:        "on-issue",
			Name:          "On Issue",
			State:         models.CanvasNodeStateReady,
			Type:          models.NodeTypeTrigger,
			Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "github.onIssue"}}),
			Configuration: datatypes.NewJSONType(map[string]any{}),
			Metadata:      datatypes.NewJSONType(map[string]any{}),
			Position:      datatypes.NewJSONType(models.Position{}),
			WebhookID:     &webhookID,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}).Error)

		return intakeWebhook{factory: factory, webhook: webhook, secret: secret}
	}

	createDraft := func(t *testing.T, factory *models.Factory, title, originURL string) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factory.CreateWorkOrderWithOrigin(
			db,
			title,
			"",
			&r.User,
			nil,
			nil,
			models.WorkOrderOrigin{URL: originURL, Label: models.OriginLabelFromURL(originURL)},
		)
		require.NoError(t, err)
		return order
	}

	closedBody := func(repository string, number int, reason string) []byte {
		t.Helper()
		return []byte(fmt.Sprintf(`{
			"action": "closed",
			"issue": {
				"number": %d,
				"html_url": "https://github.com/%s/issues/%d",
				"state_reason": %q
			},
			"repository": {"full_name": %q}
		}`, number, repository, number, reason, repository))
	}

	deliver := func(t *testing.T, hook intakeWebhook, body []byte, headers http.Header) (int, error) {
		t.Helper()
		return ArchiveDraftWorkOrdersFromGitHubIssueClosed(
			t.Context(),
			r.Encryptor,
			hook.webhook,
			"issues",
			headers,
			body,
		)
	}

	signed := func(secret, body []byte) http.Header {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+crypto.Sign(secret, body))
		return headers
	}

	reload := func(t *testing.T, factory *models.Factory, orderID uuid.UUID) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factory.FindWorkOrder(db, orderID)
		require.NoError(t, err)
		return order
	}

	closePayload := func(t *testing.T, order *models.FactoryWorkOrder) factoryevents.WorkOrderStatusUpdated {
		t.Helper()
		events, err := order.ListEvents(db, 0, nil)
		require.NoError(t, err)
		for _, event := range events {
			if event.Type != factoryevents.EventTypeOrderStatusUpdated {
				continue
			}
			var payload factoryevents.WorkOrderStatusUpdated
			require.NoError(t, json.Unmarshal(event.Data, &payload))
			if payload.ToState == models.FactoryWorkOrderStateClosed {
				return payload
			}
		}
		t.Fatal("close event not found")
		return factoryevents.WorkOrderStatusUpdated{}
	}

	t.Run("archives a matching draft with no actor", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		origin := "https://github.com/Acme/Payments/issues/12"
		first := createDraft(t, hook.factory, "First copy", origin)
		second := createDraft(t, hook.factory, "Second copy", origin)
		body := closedBody("acme/payments", 12, "not_planned")

		code, err := deliver(t, hook, body, signed(hook.secret, body))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		for _, order := range []*models.FactoryWorkOrder{first, second} {
			reloaded := reload(t, hook.factory, order.ID)
			assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
			assert.Equal(t, models.FactoryWorkOrderResultRejected, reloaded.Result)
			payload := closePayload(t, reloaded)
			assert.Equal(t, models.FactoryWorkOrderStateDraft, payload.FromState)
			assert.Equal(t, models.FactoryWorkOrderResultRejected, payload.ToResult)
			assert.Nil(t, payload.User)
		}
	})

	t.Run("leaves an open task unchanged", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		origin := "https://github.com/acme/payments/issues/12"
		order := createDraft(t, hook.factory, "Open task", origin)
		_, err := order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateOpen,
			Actor:   &r.User,
		})
		require.NoError(t, err)
		body := closedBody("acme/payments", 12, "completed")

		code, err := deliver(t, hook, body, signed(hook.secret, body))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		reloaded := reload(t, hook.factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
		assert.Empty(t, reloaded.Result)
	})

	t.Run("leaves a draft for a different issue number unchanged", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		matching := createDraft(t, hook.factory, "Matching", "https://github.com/acme/payments/issues/12")
		other := createDraft(t, hook.factory, "Other", "https://github.com/acme/payments/issues/13")
		body := closedBody("acme/payments", 12, "completed")

		code, err := deliver(t, hook, body, signed(hook.secret, body))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		assert.Equal(t, models.FactoryWorkOrderStateClosed, reload(t, hook.factory, matching.ID).State)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reload(t, hook.factory, other.ID).State)
	})

	t.Run("a second closed delivery does not change an archived draft", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		order := createDraft(t, hook.factory, "Already archived", "https://github.com/acme/payments/issues/12")
		body := closedBody("acme/payments", 12, "completed")
		headers := signed(hook.secret, body)

		code, err := deliver(t, hook, body, headers)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		archived := reload(t, hook.factory, order.ID)
		eventsBefore, err := archived.ListEvents(db, 0, nil)
		require.NoError(t, err)

		code, err = deliver(t, hook, body, headers)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)

		again := reload(t, hook.factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, again.State)
		assert.Equal(t, models.FactoryWorkOrderResultRejected, again.Result)
		assert.True(t, archived.UpdatedAt.Equal(again.UpdatedAt))
		eventsAfter, err := again.ListEvents(db, 0, nil)
		require.NoError(t, err)
		assert.Equal(t, len(eventsBefore), len(eventsAfter))
	})

	t.Run("a draft that leaves draft before the lock is not archived", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		order := createDraft(t, hook.factory, "Dispatched", "https://github.com/acme/payments/issues/12")
		body := closedBody("acme/payments", 12, "completed")

		tx := database.Conn().Begin()
		require.NoError(t, tx.Error)
		committed := false
		t.Cleanup(func() {
			if !committed {
				tx.Rollback()
			}
		})

		var locked models.FactoryWorkOrder
		require.NoError(t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", order.ID).
			First(&locked).Error)
		_, err := locked.UpdateStatus(tx, models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateOpen,
			Actor:   &r.User,
		})
		require.NoError(t, err)

		done := make(chan error, 1)
		go func() {
			_, callErr := ArchiveDraftWorkOrdersFromGitHubIssueClosed(
				context.Background(),
				r.Encryptor,
				hook.webhook,
				"issues",
				signed(hook.secret, body),
				body,
			)
			done <- callErr
		}()

		require.NoError(t, tx.Commit().Error)
		committed = true

		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("archive did not finish")
		}

		reloaded := reload(t, hook.factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
		assert.Empty(t, reloaded.Result)
	})

	t.Run("an action other than closed does not archive", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		order := createDraft(t, hook.factory, "Still open", "https://github.com/acme/payments/issues/12")
		body := []byte(`{
			"action": "opened",
			"issue": {"number": 12, "html_url": "https://github.com/acme/payments/issues/12"},
			"repository": {"full_name": "acme/payments"}
		}`)

		code, err := deliver(t, hook, body, signed(hook.secret, body))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reload(t, hook.factory, order.ID).State)
	})

	t.Run("a database read failure returns an error and leaves the draft", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		order := createDraft(t, hook.factory, "Still a draft", "https://github.com/acme/payments/issues/12")
		body := closedBody("acme/payments", 12, "completed")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		code, err := ArchiveDraftWorkOrdersFromGitHubIssueClosed(
			ctx,
			r.Encryptor,
			hook.webhook,
			"issues",
			signed(hook.secret, body),
			body,
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrArchiveDraftWorkOrders)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reload(t, hook.factory, order.ID).State)
	})

	t.Run("an unverified body does not archive", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		order := createDraft(t, hook.factory, "Unverified", "https://github.com/acme/payments/issues/12")
		body := closedBody("acme/payments", 12, "completed")
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=deadbeef")

		code, err := deliver(t, hook, body, headers)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, code)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reload(t, hook.factory, order.ID).State)
	})

	t.Run("a pull request issue does not archive", func(t *testing.T) {
		hook := newIntakeWebhook(t)
		order := createDraft(t, hook.factory, "Pull request", "https://github.com/acme/payments/issues/12")
		body := []byte(`{
			"action": "closed",
			"issue": {
				"number": 12,
				"html_url": "https://github.com/acme/payments/issues/12",
				"pull_request": {"url": "https://api.github.com/repos/acme/payments/pulls/12"}
			},
			"repository": {"full_name": "acme/payments"}
		}`)

		code, err := deliver(t, hook, body, signed(hook.secret, body))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, reload(t, hook.factory, order.ID).State)
	})
}
