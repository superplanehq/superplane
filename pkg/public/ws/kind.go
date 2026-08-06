package ws

import "strings"

const (
	KindCanvas  = "canvas"
	KindAgent   = "agent"
	KindFactory = "factory"

	// AgentSessionTopicPrefix is shared with eventdistributer topic keys.
	AgentSessionTopicPrefix = "agent-session:"
	// FactoryTopicPrefix is shared with eventdistributer topic keys.
	FactoryTopicPrefix = "factory:"
)

// KindFromTopic maps a hub subscription key to a low-cardinality kind label.
func KindFromTopic(topic string) string {
	if strings.HasPrefix(topic, AgentSessionTopicPrefix) {
		return KindAgent
	}
	if strings.HasPrefix(topic, FactoryTopicPrefix) {
		return KindFactory
	}
	return KindCanvas
}
