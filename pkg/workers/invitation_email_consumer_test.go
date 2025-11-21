package workers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

type TestEmailService struct {
	sentEmails []SentEmail
}

type SentEmail struct {
	ToEmail          string
	OrganizationName string
	InvitationLink   string
	InviterEmail     string
}

func NewTestEmailService() *TestEmailService {
	return &TestEmailService{
		sentEmails: make([]SentEmail, 0),
	}
}

func (s *TestEmailService) SendInvitationEmail(toEmail, organizationName, invitationLink, inviterEmail string) error {
	s.sentEmails = append(s.sentEmails, SentEmail{
		ToEmail:          toEmail,
		OrganizationName: organizationName,
		InvitationLink:   invitationLink,
		InviterEmail:     inviterEmail,
	})
	return nil
}

func (s *TestEmailService) GetSentEmails() []SentEmail {
	return s.sentEmails
}

func (s *TestEmailService) Reset() {
	s.sentEmails = make([]SentEmail, 0)
}

func Test__InvitationEmailConsumer(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	testEmailService := NewTestEmailService()
	amqpURL := "amqp://guest:guest@rabbitmq:5672"
	baseURL := "https://app.superplane.com"

	consumer := NewInvitationEmailConsumer(amqpURL, testEmailService, baseURL)

	go consumer.Start()
	defer consumer.Stop()

	time.Sleep(100 * time.Millisecond)

	t.Run("should send email for pending invitation", func(t *testing.T) {
		testEmailService.Reset()

		invitation, err := models.CreateInvitation(
			r.Organization.ID,
			r.User,
			"test@example.com",
			models.InvitationStatePending,
		)
		require.NoError(t, err)

		message := messages.NewInvitationCreatedMessage(invitation)
		err = message.Publish()
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			return len(testEmailService.GetSentEmails()) > 0
		}, time.Second*5, 100*time.Millisecond)

		sentEmails := testEmailService.GetSentEmails()
		require.Len(t, sentEmails, 1)

		sentEmail := sentEmails[0]
		assert.Equal(t, "test@example.com", sentEmail.ToEmail)
		assert.Equal(t, r.Organization.Name, sentEmail.OrganizationName)
		assert.Equal(t, baseURL+"/login", sentEmail.InvitationLink)
		assert.Equal(t, r.Account.Email, sentEmail.InviterEmail)
	})

	t.Run("should not send email for accepted invitation", func(t *testing.T) {
		testEmailService.Reset()

		invitation, err := models.CreateInvitation(
			r.Organization.ID,
			r.User,
			"accepted@example.com",
			models.InvitationStateAccepted,
		)
		require.NoError(t, err)

		message := messages.NewInvitationCreatedMessage(invitation)
		err = message.Publish()
		require.NoError(t, err)

		require.Never(t, func() bool {
			return len(testEmailService.GetSentEmails()) > 0
		}, time.Second*2, 100*time.Millisecond)
	})
}

func TestNewInvitationEmailConsumer(t *testing.T) {
	testEmailService := &TestEmailService{}
	rabbitMQURL := "amqp://localhost:5672"
	baseURL := "https://app.superplane.com"

	consumer := NewInvitationEmailConsumer(rabbitMQURL, testEmailService, baseURL)

	assert.NotNil(t, consumer)
	assert.Equal(t, rabbitMQURL, consumer.RabbitMQURL)
	assert.Equal(t, testEmailService, consumer.EmailService)
	assert.Equal(t, baseURL, consumer.BaseURL)
	assert.NotNil(t, consumer.Consumer)
}
