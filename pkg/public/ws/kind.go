package ws

import "strings"

const (
	KindCanvas  = "canvas"
	KindAgent   = "agent"
	KindFactory = "factory"
	KindUser    = "user"

	// AgentSessionTopicPrefix is shared with eventdistributer topic keys.
	AgentSessionTopicPrefix = "agent-session:"
	// FactoryTopicPrefix is shared with eventdistributer topic keys.
	FactoryTopicPrefix = "factory:"
	// UserTopicPrefix is shared with eventdistributer topic keys.
	UserTopicPrefix = "user:"
)

// KindFromTopic maps a hub subscription key to a low-cardinality kind label.
func KindFromTopic(topic string) string {
	if strings.HasPrefix(topic, AgentSessionTopicPrefix) {
		return KindAgent
	}
	if strings.HasPrefix(topic, FactoryTopicPrefix) {
		return KindFactory
	}
	if strings.HasPrefix(topic, UserTopicPrefix) {
		return KindUser
	}
	return KindCanvas
}
