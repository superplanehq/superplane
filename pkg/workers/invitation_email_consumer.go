package workers

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/services"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

const InvitationEmailServiceName = "superplane" + "." + messages.WorkflowExchange + "." + messages.InvitationCreatedRoutingKey + ".worker-consumer"
const InvitationEmailConnectionName = "superplane"

type InvitationEmailConsumer struct {
	Consumer     *tackle.Consumer
	RabbitMQURL  string
	EmailService services.EmailService
	BaseURL      string
}

func NewInvitationEmailConsumer(rabbitMQURL string, emailService services.EmailService, baseURL string) *InvitationEmailConsumer {
	logger := logging.NewTackleLogger(log.StandardLogger().WithFields(log.Fields{
		"consumer": "invitation_email",
	}))

	consumer := tackle.NewConsumer()
	consumer.SetLogger(logger)

	return &InvitationEmailConsumer{
		RabbitMQURL:  rabbitMQURL,
		Consumer:     consumer,
		EmailService: emailService,
		BaseURL:      baseURL,
	}
}

func (c *InvitationEmailConsumer) Start() error {
	options := tackle.Options{
		URL:            c.RabbitMQURL,
		ConnectionName: InvitationEmailConnectionName,
		Service:        InvitationEmailServiceName,
		RemoteExchange: messages.WorkflowExchange,
		RoutingKey:     messages.InvitationCreatedRoutingKey,
	}

	for {
		log.Infof("Connecting to RabbitMQ queue for %s events", messages.InvitationCreatedRoutingKey)

		err := c.Consumer.Start(&options, c.Consume)
		if err != nil {
			log.Errorf("Error consuming messages from %s: %v", messages.InvitationCreatedRoutingKey, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Warnf("Connection to RabbitMQ closed for %s, reconnecting...", messages.InvitationCreatedRoutingKey)
		time.Sleep(5 * time.Second)
	}
}

func (c *InvitationEmailConsumer) Stop() {
	c.Consumer.Stop()
}

func (c *InvitationEmailConsumer) Consume(delivery tackle.Delivery) error {
	data := &protos.InvitationCreated{}
	err := proto.Unmarshal(delivery.Body(), data)
	if err != nil {
		log.Errorf("Error unmarshaling invitation created message: %v", err)
		return err
	}

	invitationID, err := uuid.Parse(data.InvitationId)
	if err != nil {
		log.Errorf("Invalid invitation ID %s: %v", data.InvitationId, err)
		return nil
	}

	invitation, err := models.FindInvitationByID(invitationID.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warnf("Invitation %s not found", invitationID)
			return nil
		}

		log.Errorf("Error finding invitation %s: %v", invitationID, err)
		return err
	}

	org, err := models.FindOrganizationByID(invitation.OrganizationID.String())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warnf("Organization %s not found for invitation %s", invitation.OrganizationID, invitationID)
			return nil
		}

		log.Errorf("Error finding organization %s: %v", invitation.OrganizationID, err)
		return err
	}

	if invitation.State != models.InvitationStatePending {
		log.Infof("Invitation %s is not pending (state: %s), skipping email", invitationID, invitation.State)
		return nil
	}

	inviter, err := models.FindUnscopedUserByID(invitation.InvitedBy.String())
	if err != nil {
		log.Errorf("Error finding inviter %s: %v", invitation.InvitedBy, err)
		return err
	}

	err = c.EmailService.SendInvitationEmail(
		invitation.Email,
		org.Name,
		c.BaseURL+"/login",
		inviter.GetEmail(),
	)

	if err != nil {
		log.Errorf("Failed to send invitation email for %s: %v", invitationID, err)
		return err
	}

	log.Infof("Successfully sent invitation email for %s to %s", invitationID, invitation.Email)
	return nil
}
