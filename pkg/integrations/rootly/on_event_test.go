package rootly

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnEvent_Name(t *testing.T) {
	trigger := &OnEvent{}
	assert.Equal(t, "rootly.onEvent", trigger.Name())
}

func TestOnEvent_Label(t *testing.T) {
	trigger := &OnEvent{}
	assert.Equal(t, "On Event", trigger.Label())
}

func TestOnEvent_Description(t *testing.T) {
	trigger := &OnEvent{}
	assert.Equal(t, "Listen to incident timeline events", trigger.Description())
}

func TestOnEvent_Configuration(t *testing.T) {
	trigger := &OnEvent{}
	config := trigger.Configuration()

	assert.Len(t, config, 3)

	// Events field
	assert.Equal(t, "events", config[0].Name)
	assert.True(t, config[0].Required)

	// EventKinds field
	assert.Equal(t, "eventKinds", config[1].Name)
	assert.False(t, config[1].Required)

	// Visibility field
	assert.Equal(t, "visibility", config[2].Name)
	assert.False(t, config[2].Required)
}

func TestBuildTimelineEventPayload(t *testing.T) {
	webhook := TimelineEventWebhookPayload{
		Event: WebhookEvent{
			ID:       "evt_123",
			Type:     "timeline_event.created",
			IssuedAt: "2026-02-08T10:00:00Z",
		},
		Data: map[string]any{
			"id":                "te_456",
			"kind":              "note",
			"body":              "This is a test note",
			"occurred_at":       "2026-02-08T09:55:00Z",
			"created_at":        "2026-02-08T10:00:00Z",
			"user_display_name": "John Doe",
			"visibility":        "internal",
			"incident": map[string]any{
				"id":    "inc_789",
				"title": "Test Incident",
			},
		},
	}

	payload := buildTimelineEventPayload(webhook)

	assert.Equal(t, "timeline_event.created", payload["event"])
	assert.Equal(t, "evt_123", payload["event_id"])
	assert.Equal(t, "2026-02-08T10:00:00Z", payload["issued_at"])
	assert.Equal(t, "te_456", payload["id"])
	assert.Equal(t, "note", payload["kind"])
	assert.Equal(t, "This is a test note", payload["body"])
	assert.Equal(t, "John Doe", payload["user_display_name"])
	assert.Equal(t, "internal", payload["visibility"])
	assert.NotNil(t, payload["incident"])
}

func TestBuildTimelineEventPayload_MinimalData(t *testing.T) {
	webhook := TimelineEventWebhookPayload{
		Event: WebhookEvent{
			ID:       "evt_123",
			Type:     "timeline_event.created",
			IssuedAt: "2026-02-08T10:00:00Z",
		},
		Data: nil,
	}

	payload := buildTimelineEventPayload(webhook)

	assert.Equal(t, "timeline_event.created", payload["event"])
	assert.Equal(t, "evt_123", payload["event_id"])
	_, hasKind := payload["kind"]
	assert.False(t, hasKind)
}

func TestTimelineEventWebhookPayload_Unmarshal(t *testing.T) {
	jsonData := `{
		"event": {
			"id": "evt_123",
			"type": "timeline_event.created",
			"issued_at": "2026-02-08T10:00:00Z"
		},
		"data": {
			"id": "te_456",
			"kind": "note",
			"body": "Test note content"
		}
	}`

	var payload TimelineEventWebhookPayload
	err := json.Unmarshal([]byte(jsonData), &payload)

	assert.NoError(t, err)
	assert.Equal(t, "evt_123", payload.Event.ID)
	assert.Equal(t, "timeline_event.created", payload.Event.Type)
	assert.Equal(t, "te_456", payload.Data["id"])
	assert.Equal(t, "note", payload.Data["kind"])
}
