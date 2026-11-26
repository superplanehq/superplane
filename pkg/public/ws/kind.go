package ws

import (
	"strings"

	"github.com/google/uuid"
)

const (
	KindCanvas       = "canvas"
	KindFactory      = "factory"
	KindUser         = "user"
	KindPublic       = "public"
	KindWebhookSetup = "webhook_setup"

	// FactoryTopicPrefix is shared with eventdistributer topic keys.
	FactoryTopicPrefix = "factory:"
	// UserTopicPrefix is shared with eventdistributer topic keys.
	UserTopicPrefix = "user:"
	// PublicLineTopicPrefix is the guest board socket. It carries no order payload.
	PublicLineTopicPrefix   = "public-line:"
	webhookSetupTopicPrefix = "webhook-setup:"
)

func WebhookSetupTopic(organizationID uuid.UUID) string {
	return webhookSetupTopicPrefix + organizationID.String()
}

// KindFromTopic maps a hub subscription key to a low-cardinality kind label.
func KindFromTopic(topic string) string {
	if strings.HasPrefix(topic, FactoryTopicPrefix) {
		return KindFactory
	}
	if strings.HasPrefix(topic, PublicLineTopicPrefix) {
		return KindPublic
	}
	if strings.HasPrefix(topic, UserTopicPrefix) {
		return KindUser
	}
	if strings.HasPrefix(topic, webhookSetupTopicPrefix) {
		return KindWebhookSetup
	}
	return KindCanvas
}
