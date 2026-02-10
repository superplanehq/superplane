package dash0

import "encoding/json"

// OnAlertEventConfiguration stores trigger settings for alert event filtering.
type OnAlertEventConfiguration struct {
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

// AlertEventPayload is the normalized event emitted by the On Alert Event trigger.
type AlertEventPayload struct {
	EventType   string         `json:"eventType"`
	CheckID     string         `json:"checkId"`
	CheckName   string         `json:"checkName,omitempty"`
	Severity    string         `json:"severity,omitempty"`
	Labels      map[string]any `json:"labels,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	Timestamp   string         `json:"timestamp"`
	Event       map[string]any `json:"event"`
}

// AlertWebhookPayload captures raw Dash0 webhook bodies before normalization.
type AlertWebhookPayload struct {
	Data map[string]any
}

// UnmarshalJSON preserves the webhook payload as a generic JSON map.
func (p *AlertWebhookPayload) UnmarshalJSON(data []byte) error {
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	p.Data = decoded
	return nil
}
