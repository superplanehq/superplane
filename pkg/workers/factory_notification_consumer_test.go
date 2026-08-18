package workers

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryNotificationConsumer(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()

	owner := support.CreateUser(t, r, r.Organization.ID)
	creator := support.CreateUser(t, r, r.Organization.ID)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Fix login flow", "", &creator.ID, []uuid.UUID{owner.ID}, nil)
	require.NoError(t, err)

	enableNotifications := func(t *testing.T, userID uuid.UUID, params models.UserNotificationSettingsParams) {
		t.Helper()
		_, err := models.UpsertUserNotificationSettings(db, r.Organization.ID, userID, params)
		require.NoError(t, err)
	}

	newConsumer := func(emailService services.EmailService) *FactoryNotificationConsumer {
		return NewFactoryNotificationConsumer("amqp://localhost:5672", emailService, "https://app.superplane.com")
	}

	commentMessage := func(actorID string) messages.FactoryWorkOrderNotificationMessage {
		return messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      factoryModel.ID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderCommentAdded,
			ActorUserID:    actorID,
			CommentBody:    "Looks good to me",
		}
	}

	t.Run("users without settings receive the default emails", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		sent := emailService.SentWorkOrderNotificationEmails()
		require.Len(t, sent, 1)
		assert.Equal(t, owner.GetEmail(), sent[0].ToEmail)
	})

	t.Run("master switch off blocks the email", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			Enabled:        false,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, owner.GetEmail(), email.ToEmail)
		}
	})

	t.Run("comment notifies the owner but never the actor", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		sent := emailService.SentWorkOrderNotificationEmails()
		require.Len(t, sent, 1)
		assert.Equal(t, owner.GetEmail(), sent[0].ToEmail)
		assert.Contains(t, sent[0].Subject, "New comment")
		assert.Contains(t, sent[0].Subject, factoryModel.WorkOrderKey(order.Number))
		assert.Equal(t, "Looks good to me", sent[0].Data.Detail)
		assert.Contains(t, sent[0].Data.WorkOrderLink, factoryModel.Key)
		assert.Equal(t, "Draft", sent[0].Data.StatusLabel)
		assert.Equal(t, "Fix login flow", sent[0].Data.WorkOrderTitle)
		assert.Equal(t, factoryModel.WorkOrderKey(order.Number), sent[0].Data.WorkOrderKey)
		assert.NotEmpty(t, sent[0].Data.UpdatedLabel)
		assert.NotEmpty(t, sent[0].Data.AssigneeInitials)
	})

	t.Run("type toggle off blocks the email", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			Types:          map[string]bool{models.NotificationTypeWorkOrderCommentOwned: false},
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, owner.GetEmail(), email.ToEmail)
		}
	})

	t.Run("selected workspace scope excludes other factories", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeSelected,
			FactoryIDs:     []string{uuid.NewString()},
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, owner.GetEmail(), email.ToEmail)
		}
	})

	t.Run("assignment notifies only the newly assigned user", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:  r.Organization.ID.String(),
			FactoryID:       factoryModel.ID.String(),
			OrderID:         order.ID.String(),
			EventType:       factoryevents.EventTypeOrderAssigneesUpdated,
			ActorUserID:     creator.ID.String(),
			AssignedUserIDs: []string{owner.ID.String()},
		})

		sent := emailService.SentWorkOrderNotificationEmails()
		require.Len(t, sent, 1)
		assert.Equal(t, owner.GetEmail(), sent[0].ToEmail)
		assert.Contains(t, sent[0].Subject, "You are now an owner")
	})

	t.Run("initial transition into draft sends nothing", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      factoryModel.ID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			FromState:      "",
			ToState:        models.FactoryWorkOrderStateDraft,
		})

		assert.Empty(t, emailService.SentWorkOrderNotificationEmails())
	})

	t.Run("status change notifies owners and creator", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      factoryModel.ID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			FromState:      models.FactoryWorkOrderStateOpen,
			ToState:        models.FactoryWorkOrderStateClosed,
			Result:         models.FactoryWorkOrderResultCompleted,
		})

		sent := emailService.SentWorkOrderNotificationEmails()
		recipients := make([]string, 0, len(sent))
		for _, email := range sent {
			recipients = append(recipients, email.ToEmail)
			assert.Contains(t, email.Subject, "closed as completed")
		}
		assert.ElementsMatch(t, []string{owner.GetEmail(), creator.GetEmail()}, recipients)
	})

	t.Run("soft-deleted members are not emailed", func(t *testing.T) {
		left := support.CreateUser(t, r, r.Organization.ID)
		enableNotifications(t, left.ID, models.UserNotificationSettingsParams{
			Enabled:        true,
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		require.NoError(t, left.Delete())

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:  r.Organization.ID.String(),
			FactoryID:       factoryModel.ID.String(),
			OrderID:         order.ID.String(),
			EventType:       factoryevents.EventTypeOrderAssigneesUpdated,
			ActorUserID:     creator.ID.String(),
			AssignedUserIDs: []string{left.ID.String()},
		})

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, left.GetEmail(), email.ToEmail)
		}
	})

	t.Run("missing work order is skipped without error", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		message := commentMessage(creator.ID.String())
		message.OrderID = uuid.NewString()
		consume(t, newConsumer(emailService), message)

		assert.Empty(t, emailService.SentWorkOrderNotificationEmails())
	})
}

func consume(t *testing.T, consumer *FactoryNotificationConsumer, message messages.FactoryWorkOrderNotificationMessage) {
	t.Helper()

	payload, err := json.Marshal(message)
	require.NoError(t, err)
	require.NoError(t, consumer.Consume(tackle.NewFakeDelivery(payload)))
}
