package workers

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/services"
)

func TestOrganizationMemberJoinedEmailConsumer(t *testing.T) {
	consumer := NewOrganizationMemberJoinedEmailConsumer("amqp://localhost:5672", services.NewNoopEmailService(), "https://app.superplane.test/")

	t.Run("sends the owner a member management link", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consumer.EmailService = emailService

		require.NoError(t, consumer.Consume(tackle.NewFakeDelivery(memberJoinedPayload(t, messages.OrganizationMemberJoinedMessage{
			ToEmail: "owner@example.com", OrganizationID: "org-1", OrganizationName: "Example Organization", MemberEmail: "alex@example.com", MemberName: "Alex",
		}))))

		emails := emailService.SentOrganizationMemberJoinedEmails()
		require.Len(t, emails, 1)
		assert.Equal(t, "owner@example.com", emails[0].ToEmail)
		assert.Equal(t, "https://app.superplane.test/org-1/settings/members", emails[0].SettingsURL)
	})

	t.Run("uses the member email when the member name is blank", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consumer.EmailService = emailService

		require.NoError(t, consumer.Consume(tackle.NewFakeDelivery(memberJoinedPayload(t, messages.OrganizationMemberJoinedMessage{
			ToEmail:          "owner@example.com",
			OrganizationID:   "org-1",
			OrganizationName: "Example Organization",
			MemberEmail:      "alex@example.com",
			MemberName:       "   ",
		}))))

		emails := emailService.SentOrganizationMemberJoinedEmails()
		require.Len(t, emails, 1)
		assert.Equal(t, "", emails[0].MemberName)
		assert.Equal(t, "alex@example.com", emails[0].MemberEmail)
	})

	t.Run("discards malformed messages", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consumer.EmailService = emailService

		require.NoError(t, consumer.Consume(tackle.NewFakeDelivery([]byte("{"))))
		assert.Empty(t, emailService.SentOrganizationMemberJoinedEmails())
	})

	t.Run("discards incomplete messages", func(t *testing.T) {
		emailService := services.NewNoopEmailService()
		consumer.EmailService = emailService
		require.NoError(t, consumer.Consume(tackle.NewFakeDelivery(memberJoinedPayload(t, messages.OrganizationMemberJoinedMessage{
			ToEmail:          "   ",
			OrganizationID:   "org-1",
			OrganizationName: "Example Organization",
			MemberEmail:      "alex@example.com",
		}))))
		assert.Empty(t, emailService.SentOrganizationMemberJoinedEmails())
	})

	t.Run("returns email delivery errors for retry", func(t *testing.T) {
		consumer.EmailService = &failingMemberJoinedEmailService{err: errors.New("smtp unavailable")}
		err := consumer.Consume(tackle.NewFakeDelivery(memberJoinedPayload(t, messages.OrganizationMemberJoinedMessage{
			ToEmail: "owner@example.com", OrganizationID: "org-1", OrganizationName: "Example Organization", MemberEmail: "alex@example.com",
		})))
		require.ErrorContains(t, err, "smtp unavailable")
	})
}

func TestOrganizationMemberJoinedEmailPipeline(t *testing.T) {
	rabbitMQURL := "amqp://guest:guest@rabbitmq:5672/superplane_test"
	t.Setenv("RABBITMQ_URL", rabbitMQURL)

	emailService := services.NewNoopEmailService()
	consumer := NewOrganizationMemberJoinedEmailConsumer(rabbitMQURL, emailService, "https://app.superplane.test")
	go consumer.Start()
	t.Cleanup(consumer.Stop)

	require.Eventually(t, func() bool {
		return consumer.Consumer.State == tackle.StateListening
	}, 10*time.Second, 100*time.Millisecond)

	require.NoError(t, (messages.OrganizationMemberJoinedMessage{
		ToEmail:          "owner@example.com",
		OrganizationID:   "org-1",
		OrganizationName: "Example Organization",
		MemberEmail:      "alex@example.com",
		MemberName:       "Alex",
	}).Publish())

	require.Eventually(t, func() bool {
		return len(emailService.SentOrganizationMemberJoinedEmails()) == 1
	}, 10*time.Second, 100*time.Millisecond)
}

func memberJoinedPayload(t *testing.T, message messages.OrganizationMemberJoinedMessage) []byte {
	t.Helper()
	payload, err := json.Marshal(message)
	require.NoError(t, err)
	return payload
}

type failingMemberJoinedEmailService struct{ err error }

func (s *failingMemberJoinedEmailService) SendMagicCodeEmail(_, _, _ string) error { return s.err }

func (s *failingMemberJoinedEmailService) SendOrganizationMemberJoinedEmail(_, _, _, _, _ string) error {
	return s.err
}
