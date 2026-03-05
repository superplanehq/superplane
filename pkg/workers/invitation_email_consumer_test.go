package workers

import (
	"testing"

	"github.com/renderedtext/go-tackle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
)

func Test__InvitationEmailConsumer(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})

	testEmailService := services.NewNoopEmailService()
	amqpURL := "amqp://guest:guest@rabbitmq:5672"
	baseURL := "https://app.superplane.com"

	consumer := NewInvitationEmailConsumer(amqpURL, testEmailService, baseURL)

	t.Run("should send email for pending invitation", func(t *testing.T) {
		testEmailService.Reset()

		invitation, err := models.CreateInvitation(
			r.Organization.ID,
			r.User,
			"test@example.com",
			models.InvitationStatePending,
		)
		require.NoError(t, err)

		payload, err := proto.Marshal(&protos.InvitationCreated{InvitationId: invitation.ID.String()})
		require.NoError(t, err)

		err = consumer.Consume(tackle.NewFakeDelivery(payload))
		require.NoError(t, err)

		sentEmails := testEmailService.SentInvitationEmails()
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

		payload, err := proto.Marshal(&protos.InvitationCreated{InvitationId: invitation.ID.String()})
		require.NoError(t, err)

		err = consumer.Consume(tackle.NewFakeDelivery(payload))
		require.NoError(t, err)
		assert.Len(t, testEmailService.SentInvitationEmails(), 0)
	})
}

func TestNewInvitationEmailConsumer(t *testing.T) {
	testEmailService := services.NewNoopEmailService()
	rabbitMQURL := "amqp://localhost:5672"
	baseURL := "https://app.superplane.com"

	consumer := NewInvitationEmailConsumer(rabbitMQURL, testEmailService, baseURL)

	assert.NotNil(t, consumer)
	assert.Equal(t, rabbitMQURL, consumer.RabbitMQURL)
	assert.Equal(t, testEmailService, consumer.EmailService)
	assert.Equal(t, baseURL, consumer.BaseURL)
	assert.NotNil(t, consumer.Consumer)
}
