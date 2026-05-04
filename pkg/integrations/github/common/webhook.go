package common

type WebhookConfiguration struct {
	EventType  string   `json:"eventType"`
	EventTypes []string `json:"eventTypes"` // Multiple event types (takes precedence over EventType if set)
	Repository string   `json:"repository"`
}
