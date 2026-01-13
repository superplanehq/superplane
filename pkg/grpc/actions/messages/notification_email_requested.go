package messages

import (
	pb "github.com/superplanehq/superplane/pkg/protos/components"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const NotificationEmailRequestedRoutingKey = "notification-email-requested"

type NotificationEmailRequestedMessage struct {
	message *pb.NotificationEmailRequested
}

func NewNotificationEmailRequestedMessage(
	organizationID string,
	title string,
	body string,
	url string,
	urlLabel string,
	emails []string,
	groups []string,
	roles []string,
) NotificationEmailRequestedMessage {
	return NotificationEmailRequestedMessage{
		message: &pb.NotificationEmailRequested{
			OrganizationId: organizationID,
			Title:          title,
			Body:           body,
			Url:            url,
			UrlLabel:       urlLabel,
			Emails:         emails,
			Groups:         groups,
			Roles:          roles,
			Timestamp:      timestamppb.Now(),
		},
	}
}

func (m NotificationEmailRequestedMessage) Publish() error {
	return Publish(WorkflowExchange, NotificationEmailRequestedRoutingKey, toBytes(m.message))
}
