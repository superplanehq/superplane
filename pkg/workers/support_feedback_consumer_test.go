package workers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/services"
)

func Test__SupportFeedbackConsumer(t *testing.T) {
	t.Run("sends email and Discord message", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		discordCalls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			discordCalls++
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(server.Close)

		consumer := NewSupportFeedbackConsumer("amqp://localhost:5672", emailService, services.NewDiscordWebhookClient(server.URL))
		err := consumer.Consume(tackle.NewFakeDelivery(supportFeedbackPayload(t, messages.SupportFeedbackRequestedMessage{
			Category:         services.FeedbackCategoryBug,
			Details:          "The canvas did not load.",
			UserName:         "Ada Lovelace",
			UserEmail:        "ada@example.com",
			OrganizationID:   "org-1",
			OrganizationName: "Acme",
			PagePath:         "/acme/apps/deploy",
		})))
		require.NoError(t, err)

		sent := emailService.SentSupportFeedbackEmails()
		require.Len(t, sent, 1)
		assert.Equal(t, services.DefaultSupportFeedbackToEmail, sent[0].ToEmail)
		assert.Equal(t, services.FeedbackCategoryBug, sent[0].Feedback.Category)
		assert.Equal(t, "The canvas did not load.", sent[0].Feedback.Details)
		assert.Equal(t, 1, discordCalls)
	})

	t.Run("skips invalid messages", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consumer := NewSupportFeedbackConsumer("amqp://localhost:5672", emailService, services.NewDiscordWebhookClient(""))
		err := consumer.Consume(tackle.NewFakeDelivery(supportFeedbackPayload(t, messages.SupportFeedbackRequestedMessage{
			Category: "not-a-category",
			Details:  "hello",
		})))
		require.NoError(t, err)
		assert.Empty(t, emailService.SentSupportFeedbackEmails())
	})

	t.Run("returns email send errors", func(t *testing.T) {
		consumer := NewSupportFeedbackConsumer("amqp://localhost:5672", &failingEmailService{err: errors.New("smtp unavailable")}, services.NewDiscordWebhookClient(""))
		err := consumer.Consume(tackle.NewFakeDelivery(supportFeedbackPayload(t, messages.SupportFeedbackRequestedMessage{
			Category: services.FeedbackCategoryOther,
			Details:  "Need help with billing.",
		})))
		require.ErrorContains(t, err, "smtp unavailable")
	})
}

func supportFeedbackPayload(t *testing.T, message messages.SupportFeedbackRequestedMessage) []byte {
	t.Helper()

	payload, err := json.Marshal(message)
	require.NoError(t, err)
	return payload
}
