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

	t.Run("none scope blocks the email", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeNone,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, owner.GetEmail(), email.ToEmail)
		}
	})

	t.Run("comment notifies the owner but never the actor", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
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

	t.Run("mention notifies mentioned users and wins over owner comment", func(t *testing.T) {
		mentioned := support.CreateUser(t, r, r.Organization.ID)
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, mentioned.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		message := commentMessage(creator.ID.String())
		message.MentionedUserIDs = []string{mentioned.ID.String(), owner.ID.String()}
		consume(t, newConsumer(emailService), message)

		sent := emailService.SentWorkOrderNotificationEmails()
		recipients := make([]string, 0, len(sent))
		for _, email := range sent {
			recipients = append(recipients, email.ToEmail)
			assert.Contains(t, email.Subject, "mentioned you")
		}
		assert.ElementsMatch(t, []string{mentioned.GetEmail(), owner.GetEmail()}, recipients)
	})

	t.Run("filtered type list without mentions blocks the mention email", func(t *testing.T) {
		mentioned := support.CreateUser(t, r, r.Organization.ID)
		enableNotifications(t, mentioned.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeFiltered,
			WorkspaceFilters: []models.NotificationWorkspaceFilter{{
				WorkspaceID: factoryModel.ID.String(),
				EventTypes:  []string{models.NotificationTypeWorkOrderAssigned},
			}},
		})
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		message := commentMessage(creator.ID.String())
		message.MentionedUserIDs = []string{mentioned.ID.String()}
		consume(t, newConsumer(emailService), message)

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, mentioned.GetEmail(), email.ToEmail)
		}
	})

	t.Run("all scope type list without comments blocks the email", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes:     []string{models.NotificationTypeWorkOrderAssigned},
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, owner.GetEmail(), email.ToEmail)
		}
	})

	t.Run("filtered type list without comments blocks the email", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeFiltered,
			WorkspaceFilters: []models.NotificationWorkspaceFilter{{
				WorkspaceID: factoryModel.ID.String(),
				EventTypes:  []string{models.NotificationTypeWorkOrderAssigned},
			}},
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, owner.GetEmail(), email.ToEmail)
		}
	})

	t.Run("filtered workspace scope excludes other factories", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeFiltered,
			WorkspaceFilters: []models.NotificationWorkspaceFilter{{
				WorkspaceID: uuid.NewString(),
				EventTypes:  []string{models.NotificationTypeWorkOrderCommentOwned},
			}},
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), commentMessage(creator.ID.String()))

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, owner.GetEmail(), email.ToEmail)
		}
	})

	t.Run("assignment notifies only the newly assigned user", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
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
			assert.Contains(t, email.Subject, "Task")
			assert.NotContains(t, email.Subject, "Work order")
		}
		assert.ElementsMatch(t, []string{owner.GetEmail(), creator.GetEmail()}, recipients)
	})

	t.Run("status note notifies owners and creator", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:     r.Organization.ID.String(),
			FactoryID:          factoryModel.ID.String(),
			OrderID:            order.ID.String(),
			EventType:          factoryevents.EventTypeOrderStatusNoteUpdated,
			ActorName:          "pr-watcher",
			StatusNoteHeadline: "Review the pull request",
			StatusNoteBody:     "Merging the PR completes this work order automatically.",
			StatusNoteCtaLabel: "Review PR #42",
			StatusNoteCtaURL:   "https://github.com/example/repo/pull/42",
		})

		sent := emailService.SentWorkOrderNotificationEmails()
		recipients := make([]string, 0, len(sent))
		for _, email := range sent {
			recipients = append(recipients, email.ToEmail)
			assert.Contains(t, email.Subject, "Review the pull request")
			assert.Contains(t, email.Data.Summary, "waiting on you")
			assert.Contains(t, email.Data.Detail, "Merging the PR completes this work order automatically.")
			assert.Equal(t, "Review PR #42", email.Data.DetailCtaLabel)
			assert.Equal(t, "https://github.com/example/repo/pull/42", email.Data.DetailCtaURL)
		}
		assert.ElementsMatch(t, []string{owner.GetEmail(), creator.GetEmail()}, recipients)
	})

	t.Run("status note setting can be turned off without affecting comments", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes: []string{
				models.NotificationTypeWorkOrderCommentOwned,
				models.NotificationTypeWorkOrderStatusOwned,
				models.NotificationTypeWorkOrderArtifactOwned,
				models.NotificationTypeWorkOrderAssigned,
				models.NotificationTypeWorkOrderMention,
			},
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:     r.Organization.ID.String(),
			FactoryID:          factoryModel.ID.String(),
			OrderID:            order.ID.String(),
			EventType:          factoryevents.EventTypeOrderStatusNoteUpdated,
			ActorName:          "pr-watcher",
			StatusNoteHeadline: "Review the pull request",
		})

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, owner.GetEmail(), email.ToEmail)
		}

		// Restore full opt-in so later subtests aren't affected by this
		// deliberately narrowed type list.
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
	})

	t.Run("soft-deleted members are not emailed", func(t *testing.T) {
		left := support.CreateUser(t, r, r.Organization.ID)
		enableNotifications(t, left.ID, models.UserNotificationSettingsParams{
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

	t.Run("browser channel publishes title body and task path", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consumer, published := capturingConsumer(newConsumer(emailService))
		consume(t, consumer, commentMessage(creator.ID.String()))

		assert.Empty(t, emailService.SentWorkOrderNotificationEmails())
		require.Len(t, *published, 1)
		assert.Equal(t, owner.ID.String(), (*published)[0].UserID)
		assert.Contains(t, (*published)[0].Title, "New comment")
		assert.Contains(t, (*published)[0].Body, factoryModel.WorkOrderKey(order.Number))
		assert.Equal(t, order.URLPath(factoryModel.Key), (*published)[0].URLPath)
		assert.Equal(t, factoryModel.WorkOrderKey(order.Number), (*published)[0].OrderKey)
		assert.Equal(t, factoryModel.Key, (*published)[0].FactoryKey)
	})

	t.Run("browser channel excludes the actor", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consumer, published := capturingConsumer(newConsumer(emailService))
		consume(t, consumer, commentMessage(creator.ID.String()))

		require.Len(t, *published, 1)
		assert.Equal(t, owner.ID.String(), (*published)[0].UserID)
	})

	t.Run("email channel stays unaffected when browser is off", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consumer, published := capturingConsumer(newConsumer(emailService))
		consume(t, consumer, commentMessage(creator.ID.String()))

		sent := emailService.SentWorkOrderNotificationEmails()
		require.Len(t, sent, 1)
		assert.Equal(t, owner.GetEmail(), sent[0].ToEmail)
		assert.Empty(t, *published)
	})

	t.Run("browser channel off publishes nothing", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeAll,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeNone,
		})

		emailService := services.NewNoopEmailService()
		consumer, published := capturingConsumer(newConsumer(emailService))
		consume(t, consumer, commentMessage(creator.ID.String()))

		assert.Empty(t, *published)
	})
}

func capturingConsumer(consumer *FactoryNotificationConsumer) (*FactoryNotificationConsumer, *[]messages.UserNotificationMessage) {
	published := []messages.UserNotificationMessage{}
	consumer.publishUserNotification = func(message messages.UserNotificationMessage) error {
		published = append(published, message)
		return nil
	}
	return consumer, &published
}

func consume(t *testing.T, consumer *FactoryNotificationConsumer, message messages.FactoryWorkOrderNotificationMessage) {
	t.Helper()

	payload, err := json.Marshal(message)
	require.NoError(t, err)
	require.NoError(t, consumer.Consume(tackle.NewFakeDelivery(payload)))
}

func TestBuildWorkOrderNotificationContent_SubjectsUseTask(t *testing.T) {
	factoryModel := &models.Factory{Key: "SP"}
	order := &models.FactoryWorkOrder{Number: 42, Title: "Fix login"}

	t.Run("status change", func(t *testing.T) {
		content := buildWorkOrderNotificationContent(
			factoryModel,
			order,
			messages.FactoryWorkOrderNotificationMessage{
				EventType: factoryevents.EventTypeOrderStatusUpdated,
				ToState:   models.FactoryWorkOrderStateClosed,
				Result:    models.FactoryWorkOrderResultCompleted,
			},
			"Ana",
			models.NotificationTypeWorkOrderStatusOwned,
		)
		assert.Equal(t, "[SP-42] Task closed as completed", content.Subject)
		assert.NotContains(t, content.Subject, "Work order")
	})

	t.Run("unknown event type", func(t *testing.T) {
		content := buildWorkOrderNotificationContent(
			factoryModel,
			order,
			messages.FactoryWorkOrderNotificationMessage{EventType: "order.unknown"},
			"Ana",
			"",
		)
		assert.Equal(t, "[SP-42] Task update", content.Subject)
	})
}
