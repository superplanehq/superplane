package firehydrant

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIncident__HandleWebhook(t *testing.T) {
	trigger := &OnIncident{}

	signatureFor := func(secret string, body []byte) string {
		return computeHMACSHA256([]byte(secret), body)
	}

	t.Run("missing fh-signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"data":{"incident":{"id":"inc-123"}},"event":{"operation":"CREATED","resource_type":"incident"}}`)
		headers := http.Header{}
		headers.Set("fh-signature", "invalid-hex-signature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("no secret configured -> skip verification", func(t *testing.T) {
		body := []byte(`{"data":{"incident":{"id":"inc-123","name":"Test"}},"event":{"operation":"CREATED","resource_type":"incident"}}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: ""},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte("invalid json")
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("non-incident resource_type -> no emit", func(t *testing.T) {
		body := []byte(`{"data":{},"event":{"operation":"CREATED","resource_type":"change_event"}}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("UPDATED operation without milestone filter -> no emit", func(t *testing.T) {
		body := []byte(`{"data":{"incident":{"id":"inc-123"}},"event":{"operation":"UPDATED","resource_type":"incident"}}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("CREATED incident -> event emitted with normalized fields", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "04d9fd1a-ba9c-417d-b396-58a6e2c374de",
					"name": "API Outage",
					"number": 42,
					"severity": {"slug": "SEV1", "description": "Critical"},
					"priority": {"slug": "P1", "description": "High"},
					"current_milestone": "started"
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "firehydrant.incident.created", payload.Type)

		data := payload.Data.(map[string]any)
		assert.Equal(t, "incident.created", data["event"])
		assert.Equal(t, "CREATED", data["operation"])

		incident := data["incident"].(map[string]any)
		assert.Equal(t, "SEV1", incident["severity"], "severity should be flattened to slug string")
		assert.Equal(t, "P1", incident["priority"], "priority should be flattened to slug string")
		assert.Equal(t, "API Outage", incident["name"], "scalar fields should be preserved")
		assert.Equal(t, "started", incident["current_milestone"], "scalar fields should be preserved")
	})

	t.Run("severity filter match -> event emitted", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-123",
					"name": "DB Outage",
					"severity": {"slug": "SEV1", "description": "Critical"}
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severities": []any{"SEV1", "SEV0"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("severity filter mismatch -> no emit", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-123",
					"name": "Minor Issue",
					"severity": {"slug": "SEV3", "description": "Minor"}
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severities": []any{"SEV1"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("no severity on incident with filter -> no emit", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-456",
					"name": "Unknown Severity Incident"
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severities": []any{"SEV1"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("UPDATED with matching milestone filter -> event emitted", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-789",
					"name": "Mitigated Outage",
					"current_milestone": "mitigated",
					"milestones": [{"type": "started"}, {"type": "mitigated"}],
					"severity": {"slug": "SEV1", "description": "Critical"}
				}
			},
			"event": {
				"operation": "UPDATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"current_milestone": []any{"mitigated", "resolved"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "firehydrant.incident.updated", payload.Type)

		data := payload.Data.(map[string]any)
		assert.Equal(t, "incident.updated", data["event"])
		assert.Equal(t, "UPDATED", data["operation"])
	})

	t.Run("UPDATED with non-matching milestone filter -> no emit", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-789",
					"name": "Started Outage",
					"current_milestone": "started"
				}
			},
			"event": {
				"operation": "UPDATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"current_milestone": []any{"mitigated", "resolved"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("CREATED with milestone filter configured but no match -> no emit", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-100",
					"name": "New Incident"
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"current_milestone": []any{"mitigated"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("CREATED with matching milestone filter -> event emitted", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-101",
					"name": "New Incident With Milestone",
					"current_milestone": "started",
					"severity": {"slug": "SEV1", "description": "Critical"}
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"current_milestone": []any{"started"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "firehydrant.incident.created", payload.Type)

		data := payload.Data.(map[string]any)
		assert.Equal(t, "incident.created", data["event"])
		assert.Equal(t, "CREATED", data["operation"])
	})

	t.Run("UPDATED with milestone and severity filter both matching -> event emitted", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-200",
					"name": "Resolved Critical",
					"current_milestone": "resolved",
					"milestones": [{"type": "started"}, {"type": "resolved"}],
					"severity": {"slug": "SEV1", "description": "Critical"}
				}
			},
			"event": {
				"operation": "UPDATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"current_milestone": []any{"resolved"},
				"severities":        []any{"SEV1"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("UPDATED with milestone match but severity mismatch -> no emit", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-201",
					"name": "Resolved Minor",
					"current_milestone": "resolved",
					"milestones": [{"type": "started"}, {"type": "resolved"}],
					"severity": {"slug": "SEV3", "description": "Minor"}
				}
			},
			"event": {
				"operation": "UPDATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"current_milestone": []any{"resolved"},
				"severities":        []any{"SEV1"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})
}

func Test__OnIncident__Setup(t *testing.T) {
	trigger := &OnIncident{}

	t.Run("valid configuration -> webhook requested with incidents subscription", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{},
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"incidents"}, webhookConfig.Subscriptions)
	})

	t.Run("configuration with severities -> webhook requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Configuration: map[string]any{
				"severities": []any{"SEV1", "SEV2"},
			},
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"incidents"}, webhookConfig.Subscriptions)
	})
}

func Test__verifyWebhookSignature(t *testing.T) {
	t.Run("empty secret -> skip verification", func(t *testing.T) {
		err := verifyWebhookSignature("", []byte("body"), []byte{})
		require.NoError(t, err)
	})

	t.Run("missing signature with secret -> error", func(t *testing.T) {
		err := verifyWebhookSignature("", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "missing signature")
	})

	t.Run("signature mismatch -> error", func(t *testing.T) {
		err := verifyWebhookSignature("invalid-hex", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "signature mismatch")
	})

	t.Run("valid signature -> no error", func(t *testing.T) {
		body := []byte("test body")
		secret := []byte("test secret")
		sig := computeHMACSHA256(secret, body)

		err := verifyWebhookSignature(sig, body, secret)
		require.NoError(t, err)
	})
}

func Test__buildIncidentPayload(t *testing.T) {
	t.Run("nested severity and priority are flattened to slugs", func(t *testing.T) {
		payload := WebhookPayload{
			Data: WebhookData{
				Incident: map[string]any{
					"id":       "inc-1",
					"name":     "Outage",
					"severity": map[string]any{"slug": "SEV1", "description": "Critical"},
					"priority": map[string]any{"slug": "P0", "description": "Urgent"},
				},
			},
			Event: WebhookEvent{Operation: "CREATED", ResourceType: "incident"},
		}

		result := buildIncidentPayload(payload)
		incident := result["incident"].(map[string]any)

		assert.Equal(t, "SEV1", incident["severity"])
		assert.Equal(t, "P0", incident["priority"])
		assert.Equal(t, "inc-1", incident["id"])
		assert.Equal(t, "Outage", incident["name"])
	})

	t.Run("nil severity and priority are left unchanged", func(t *testing.T) {
		payload := WebhookPayload{
			Data: WebhookData{
				Incident: map[string]any{
					"id":   "inc-2",
					"name": "No Severity",
				},
			},
			Event: WebhookEvent{Operation: "CREATED", ResourceType: "incident"},
		}

		result := buildIncidentPayload(payload)
		incident := result["incident"].(map[string]any)

		_, hasSeverity := incident["severity"]
		_, hasPriority := incident["priority"]
		assert.False(t, hasSeverity)
		assert.False(t, hasPriority)
		assert.Equal(t, "inc-2", incident["id"])
	})

	t.Run("severity already a string is preserved", func(t *testing.T) {
		payload := WebhookPayload{
			Data: WebhookData{
				Incident: map[string]any{
					"id":       "inc-3",
					"severity": "SEV2",
					"priority": "P1",
				},
			},
			Event: WebhookEvent{Operation: "CREATED", ResourceType: "incident"},
		}

		result := buildIncidentPayload(payload)
		incident := result["incident"].(map[string]any)

		assert.Equal(t, "SEV2", incident["severity"])
		assert.Equal(t, "P1", incident["priority"])
	})

	t.Run("nested object without slug key is left unchanged", func(t *testing.T) {
		payload := WebhookPayload{
			Data: WebhookData{
				Incident: map[string]any{
					"id":       "inc-4",
					"severity": map[string]any{"description": "Critical"},
					"priority": map[string]any{"level": 1},
				},
			},
			Event: WebhookEvent{Operation: "CREATED", ResourceType: "incident"},
		}

		result := buildIncidentPayload(payload)
		incident := result["incident"].(map[string]any)

		assert.Equal(t, map[string]any{"description": "Critical"}, incident["severity"])
		assert.Equal(t, map[string]any{"level": 1}, incident["priority"])
	})

	t.Run("nil incident data -> no incident key", func(t *testing.T) {
		payload := WebhookPayload{
			Data:  WebhookData{Incident: nil},
			Event: WebhookEvent{Operation: "CREATED", ResourceType: "incident"},
		}

		result := buildIncidentPayload(payload)

		_, hasIncident := result["incident"]
		assert.False(t, hasIncident)
		assert.Equal(t, "incident.created", result["event"])
	})

	t.Run("does not mutate the original incident map", func(t *testing.T) {
		original := map[string]any{
			"id":       "inc-5",
			"severity": map[string]any{"slug": "SEV1", "description": "Critical"},
		}
		payload := WebhookPayload{
			Data:  WebhookData{Incident: original},
			Event: WebhookEvent{Operation: "CREATED", ResourceType: "incident"},
		}

		_ = buildIncidentPayload(payload)

		// The original map should still have the nested object
		_, isMap := original["severity"].(map[string]any)
		assert.True(t, isMap, "original incident map should not be mutated")
	})
}
