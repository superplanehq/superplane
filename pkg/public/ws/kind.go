package ws

import "strings"

const (
	KindCanvas  = "canvas"
	KindFactory = "factory"
	KindUser    = "user"
	KindPublic  = "public"

	// FactoryTopicPrefix is shared with eventdistributer topic keys.
	FactoryTopicPrefix = "factory:"
	// UserTopicPrefix is shared with eventdistributer topic keys.
	UserTopicPrefix = "user:"
	// PublicLineTopicPrefix is the guest board socket. It carries no order payload.
	PublicLineTopicPrefix = "public-line:"
)

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
	return KindCanvas
}
