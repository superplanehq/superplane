package workers

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/services"
)

func Test__MagicCodeEmailConsumer(t *testing.T) {
	baseURL := "https://app.superplane.com"

	t.Run("sends readable code and magic link", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consumer := NewMagicCodeEmailConsumer("amqp://localhost:5672", emailService, baseURL)
		payload := magicCodeRequestedPayload(t, messages.MagicCodeRequestedMessage{
			Email:          "user@example.com",
			Code:           "123456",
			MagicLinkToken: "token with spaces",
			RedirectURL:    "https://example.com/welcome?tab=home",
			SignupIntent:   true,
		})

		err := consumer.Consume(tackle.NewFakeDelivery(payload))
		require.NoError(t, err)

		sentEmails := emailService.SentMagicCodeEmails()
		require.Len(t, sentEmails, 1)
		assert.Equal(t, "user@example.com", sentEmails[0].ToEmail)
		assert.Equal(t, "123 456", sentEmails[0].Code)
		assert.Equal(t, "https://app.superplane.com/auth/magic-code/verify?token=token+with+spaces&redirect=https%3A%2F%2Fexample.com%2Fwelcome%3Ftab%3Dhome&signup=true", sentEmails[0].MagicLink)
	})

	t.Run("skips incomplete messages", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consumer := NewMagicCodeEmailConsumer("amqp://localhost:5672", emailService, baseURL)
		payload := magicCodeRequestedPayload(t, messages.MagicCodeRequestedMessage{
			Email: "user@example.com",
		})

		err := consumer.Consume(tackle.NewFakeDelivery(payload))
		require.NoError(t, err)
		assert.Empty(t, emailService.SentMagicCodeEmails())
	})

	t.Run("returns send errors", func(t *testing.T) {
		emailService := &failingEmailService{err: errors.New("smtp unavailable")}
		consumer := NewMagicCodeEmailConsumer("amqp://localhost:5672", emailService, baseURL)
		payload := magicCodeRequestedPayload(t, messages.MagicCodeRequestedMessage{
			Email: "user@example.com",
			Code:  "123456",
		})

		err := consumer.Consume(tackle.NewFakeDelivery(payload))
		require.ErrorContains(t, err, "smtp unavailable")
	})
}

func magicCodeRequestedPayload(t *testing.T, message messages.MagicCodeRequestedMessage) []byte {
	t.Helper()

	payload, err := json.Marshal(message)
	require.NoError(t, err)
	return payload
}

type failingEmailService struct {
	err error
}

func (s *failingEmailService) SendInvitationEmail(_, _, _, _ string) error {
	return s.err
}

func (s *failingEmailService) SendMagicCodeEmail(_, _, _ string) error {
	return s.err
}
