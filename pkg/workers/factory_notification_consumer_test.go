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
	starter := support.CreateUser(t, r, r.Organization.ID)

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

	statusMessage := func() messages.FactoryWorkOrderNotificationMessage {
		return messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      factoryModel.ID.String(),
			OrderID:        order.ID.String(),
			EventType:      factoryevents.EventTypeOrderStatusUpdated,
			FromState:      models.FactoryWorkOrderStateOpen,
			ToState:        models.FactoryWorkOrderStateClosed,
			Result:         models.FactoryWorkOrderResultCompleted,
		}
	}

	agentQuestionMessage := func() messages.FactoryWorkOrderNotificationMessage {
		return messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:       r.Organization.ID.String(),
			FactoryID:            factoryModel.ID.String(),
			OrderID:              order.ID.String(),
			EventType:            factoryevents.EventTypeOrderAgentQuestion,
			QuestionPrompt:       "Which service owns retries?",
			SessionStarterUserID: starter.ID.String(),
		}
	}

	planReadyMessage := func() messages.FactoryWorkOrderNotificationMessage {
		return messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:       r.Organization.ID.String(),
			FactoryID:            factoryModel.ID.String(),
			OrderID:              order.ID.String(),
			EventType:            factoryevents.EventTypeOrderPlanReady,
			SessionStarterUserID: starter.ID.String(),
		}
	}

	t.Run("users without settings receive the default emails", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), agentQuestionMessage())

		sent := emailService.SentWorkOrderNotificationEmails()
		recipients := make([]string, 0, len(sent))
		for _, email := range sent {
			recipients = append(recipients, email.ToEmail)
		}
		assert.ElementsMatch(t, []string{creator.GetEmail(), starter.GetEmail()}, recipients)
	})

	t.Run("none scope blocks the email", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeNone,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), agentQuestionMessage())

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, creator.GetEmail(), email.ToEmail)
		}

		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
	})

	t.Run("agent question notifies the task creator and the session starter", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, starter.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), agentQuestionMessage())

		sent := emailService.SentWorkOrderNotificationEmails()
		recipients := make([]string, 0, len(sent))
		for _, email := range sent {
			recipients = append(recipients, email.ToEmail)
			assert.Contains(t, email.Subject, "The agent has a question")
			assert.Contains(t, email.Data.Summary, "waiting for an answer")
			assert.Equal(t, "Which service owns retries?", email.Data.Detail)
		}
		assert.ElementsMatch(t, []string{creator.GetEmail(), starter.GetEmail()}, recipients)
	})

	t.Run("plan ready notifies the task creator and the session starter", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, starter.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), planReadyMessage())

		sent := emailService.SentWorkOrderNotificationEmails()
		recipients := make([]string, 0, len(sent))
		for _, email := range sent {
			recipients = append(recipients, email.ToEmail)
			assert.Contains(t, email.Subject, "Plan is ready")
			assert.Contains(t, email.Data.Summary, "plan is ready")
		}
		assert.ElementsMatch(t, []string{creator.GetEmail(), starter.GetEmail()}, recipients)
	})

	t.Run("plan ready without candidates from a missing starter still notifies the creator", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		message := planReadyMessage()
		message.SessionStarterUserID = ""
		consume(t, newConsumer(emailService), message)

		sent := emailService.SentWorkOrderNotificationEmails()
		recipients := make([]string, 0, len(sent))
		for _, email := range sent {
			recipients = append(recipients, email.ToEmail)
		}
		assert.Contains(t, recipients, creator.GetEmail())
	})

	t.Run("comment assignee and artifact messages produce no recipients", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		dropped := []messages.FactoryWorkOrderNotificationMessage{
			{
				OrganizationID: r.Organization.ID.String(),
				FactoryID:      factoryModel.ID.String(),
				OrderID:        order.ID.String(),
				EventType:      factoryevents.EventTypeOrderCommentAdded,
				CommentBody:    "Looks good to me",
			},
			{
				OrganizationID:  r.Organization.ID.String(),
				FactoryID:       factoryModel.ID.String(),
				OrderID:         order.ID.String(),
				EventType:       factoryevents.EventTypeOrderAssigneesUpdated,
				AssignedUserIDs: []string{owner.ID.String()},
			},
			{
				OrganizationID: r.Organization.ID.String(),
				FactoryID:      factoryModel.ID.String(),
				OrderID:        order.ID.String(),
				EventType:      factoryevents.EventTypeOrderArtifactAdded,
				ArtifactType:   factoryevents.ArtifactTypeMarkdown,
			},
		}

		emailService := services.NewNoopEmailService()
		consumer := newConsumer(emailService)
		for _, message := range dropped {
			consume(t, consumer, message)
		}
		assert.Empty(t, emailService.SentWorkOrderNotificationEmails())
	})

	t.Run("all scope type list without agent questions blocks the email", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes:     []string{models.NotificationTypeWorkOrderStatusOwned},
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), agentQuestionMessage())

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, creator.GetEmail(), email.ToEmail)
		}

		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
	})

	t.Run("filtered workspace scope excludes other factories", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeFiltered,
			WorkspaceFilters: []models.NotificationWorkspaceFilter{{
				WorkspaceID: uuid.NewString(),
				EventTypes:  []string{models.NotificationTypeWorkOrderAgentQuestion},
			}},
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), agentQuestionMessage())

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, creator.GetEmail(), email.ToEmail)
		}

		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
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
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), statusMessage())

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

	t.Run("status note setting can be turned off without affecting status changes", func(t *testing.T) {
		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes: []string{
				models.NotificationTypeWorkOrderStatusOwned,
				models.NotificationTypeWorkOrderAgentQuestion,
				models.NotificationTypeWorkOrderPlanReady,
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

		enableNotifications(t, owner.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
	})

	t.Run("soft-deleted members are not emailed", func(t *testing.T) {
		left := support.CreateUser(t, r, r.Organization.ID)
		leftOrder, err := factoryModel.CreateWorkOrder(db, "Left member task", "", &left.ID, nil, nil)
		require.NoError(t, err)
		enableNotifications(t, left.ID, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		require.NoError(t, left.Delete())

		emailService := services.NewNoopEmailService()
		consume(t, newConsumer(emailService), messages.FactoryWorkOrderNotificationMessage{
			OrganizationID: r.Organization.ID.String(),
			FactoryID:      factoryModel.ID.String(),
			OrderID:        leftOrder.ID.String(),
			EventType:      factoryevents.EventTypeOrderAgentQuestion,
			QuestionPrompt: "Still there?",
		})

		for _, email := range emailService.SentWorkOrderNotificationEmails() {
			assert.NotEqual(t, left.GetEmail(), email.ToEmail)
		}
	})

	t.Run("missing work order is skipped without error", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		message := agentQuestionMessage()
		message.OrderID = uuid.NewString()
		consume(t, newConsumer(emailService), message)

		assert.Empty(t, emailService.SentWorkOrderNotificationEmails())
	})

	t.Run("browser channel publishes title body and task path", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeAll,
		})
		enableNotifications(t, starter.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeNone,
		})

		emailService := services.NewNoopEmailService()
		consumer, published := capturingConsumer(newConsumer(emailService))
		consume(t, consumer, agentQuestionMessage())

		assert.Empty(t, emailService.SentWorkOrderNotificationEmails())
		require.NotEmpty(t, *published)
		var creatorAlert *messages.UserNotificationMessage
		for i := range *published {
			if (*published)[i].UserID == creator.ID.String() {
				creatorAlert = &(*published)[i]
				break
			}
		}
		require.NotNil(t, creatorAlert)
		assert.Contains(t, creatorAlert.Title, "The agent has a question")
		assert.Contains(t, creatorAlert.Body, factoryModel.WorkOrderKey(order.Number))
		assert.Equal(t, order.URLPath(factoryModel.Key), creatorAlert.URLPath)
		assert.Equal(t, factoryModel.WorkOrderKey(order.Number), creatorAlert.OrderKey)
		assert.Equal(t, factoryModel.Key, creatorAlert.FactoryKey)
		assert.Equal(t, models.NotificationTypeWorkOrderAgentQuestion, creatorAlert.EventType)
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
		message := statusMessage()
		message.ActorUserID = creator.ID.String()
		consume(t, consumer, message)

		require.Len(t, *published, 1)
		assert.Equal(t, owner.ID.String(), (*published)[0].UserID)
	})

	t.Run("email channel stays unaffected when browser is off", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeAll,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeNone,
		})
		enableNotifications(t, starter.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeNone,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeNone,
		})

		emailService := services.NewNoopEmailService()
		consumer, published := capturingConsumer(newConsumer(emailService))
		consume(t, consumer, agentQuestionMessage())

		sent := emailService.SentWorkOrderNotificationEmails()
		require.Len(t, sent, 1)
		assert.Equal(t, creator.GetEmail(), sent[0].ToEmail)
		assert.Empty(t, *published)
	})

	t.Run("browser channel off publishes nothing", func(t *testing.T) {
		enableNotifications(t, creator.ID, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeAll,
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeNone,
		})

		emailService := services.NewNoopEmailService()
		consumer, published := capturingConsumer(newConsumer(emailService))
		consume(t, consumer, agentQuestionMessage())

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
		)
		assert.Equal(t, "[SP-42] Task closed as completed", content.Subject)
		assert.NotContains(t, content.Subject, "Work order")
	})

	t.Run("agent question", func(t *testing.T) {
		content := buildWorkOrderNotificationContent(
			factoryModel,
			order,
			messages.FactoryWorkOrderNotificationMessage{
				EventType:      factoryevents.EventTypeOrderAgentQuestion,
				QuestionPrompt: "Which service owns retries?",
			},
			"An automation",
		)
		assert.Equal(t, "[SP-42] The agent has a question", content.Subject)
		assert.Equal(t, "The agent is waiting for an answer on SP-42.", content.Data.Summary)
		assert.Equal(t, "Which service owns retries?", content.Data.Detail)
	})

	t.Run("plan ready", func(t *testing.T) {
		content := buildWorkOrderNotificationContent(
			factoryModel,
			order,
			messages.FactoryWorkOrderNotificationMessage{EventType: factoryevents.EventTypeOrderPlanReady},
			"An automation",
		)
		assert.Equal(t, "[SP-42] Plan is ready", content.Subject)
		assert.Equal(t, "Refinement finished and the plan is ready for SP-42.", content.Data.Summary)
	})

	t.Run("unknown event type", func(t *testing.T) {
		content := buildWorkOrderNotificationContent(
			factoryModel,
			order,
			messages.FactoryWorkOrderNotificationMessage{EventType: "order.unknown"},
			"Ana",
		)
		assert.Equal(t, "[SP-42] Task update", content.Subject)
	})
}
