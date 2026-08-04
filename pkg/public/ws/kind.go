package ws

import "strings"

const (
	KindCanvas = "canvas"
	KindAgent  = "agent"

	// AgentSessionTopicPrefix is shared with eventdistributer topic keys.
	AgentSessionTopicPrefix = "agent-session:"
)

// KindFromTopic maps a hub subscription key to a low-cardinality kind label.
func KindFromTopic(topic string) string {
	if strings.HasPrefix(topic, AgentSessionTopicPrefix) {
		return KindAgent
	}
	return KindCanvas
}
