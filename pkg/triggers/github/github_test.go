package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/triggers"
)

// Helper function to create a signed webhook request
func createWebhookRequest(body []byte, eventType string, secret string, config Configuration) triggers.WebhookRequestContext {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	signature := fmt.Sprintf("sha256=%x", h.Sum(nil))

	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", signature)
	headers.Set("X-GitHub-Event", eventType)

	return triggers.WebhookRequestContext{
		Body:           body,
		Headers:        headers,
		Configuration:  config,
		WebhookContext: &DummyWebhookContext{Secret: secret},
		EventContext:   &DummyEventContext{},
	}
}

func Test__HandleWebhook(t *testing.T) {
	trigger := &GitHub{}

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(triggers.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("no X-GitHub-Event -> 400", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")

		code, err := trigger.HandleWebhook(triggers.WebhookRequestContext{
			Headers:        headers,
			EventContext:   &DummyEventContext{},
			WebhookContext: &DummyWebhookContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-GitHub-Event header")
	})

	t.Run("event not in configuration is ignored", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", "pull_request")

		eventContext := &DummyEventContext{}
		code, err := trigger.HandleWebhook(triggers.WebhookRequestContext{
			Headers:        headers,
			Configuration:  Configuration{EventType: "push"},
			EventContext:   eventContext,
			WebhookContext: &DummyWebhookContext{},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", "push")

		code, err := trigger.HandleWebhook(triggers.WebhookRequestContext{
			Body:           []byte(`{"ref":"refs/heads/main"}`),
			Headers:        headers,
			Configuration:  Configuration{EventType: "push"},
			WebhookContext: &DummyWebhookContext{Secret: secret},
			EventContext:   &DummyEventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("branch deletion push is ignored", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main","deleted":true}`)
		ctx := createWebhookRequest(body, "push", "test-secret", Configuration{EventType: "push"})

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("push event - event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main"}`)
		ctx := createWebhookRequest(body, "push", "test-secret", Configuration{EventType: "push"})

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("push event with exact branch match - event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main"}`)
		config := Configuration{
			EventType: "push",
			Refs: []*RefFilter{
				{Type: FilterTypeExactMatch, Value: "main"},
			},
		}
		ctx := createWebhookRequest(body, "push", "test-secret", config)

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("push event with exact branch no match - event is ignored", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/develop"}`)
		config := Configuration{
			EventType: "push",
			Refs: []*RefFilter{
				{Type: FilterTypeExactMatch, Value: "main"},
			},
		}
		ctx := createWebhookRequest(body, "push", "test-secret", config)

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("push event with regex branch match - event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/feature/my-feature"}`)
		config := Configuration{
			EventType: "push",
			Refs: []*RefFilter{
				{Type: FilterTypeRegex, Value: "^feature/.*"},
			},
		}
		ctx := createWebhookRequest(body, "push", "test-secret", config)

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("push event with regex branch no match - event is ignored", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/bugfix/my-fix"}`)
		config := Configuration{
			EventType: "push",
			Refs: []*RefFilter{
				{Type: FilterTypeRegex, Value: "^feature/.*"},
			},
		}
		ctx := createWebhookRequest(body, "push", "test-secret", config)

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("push event with multiple branch filters - one matches", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/develop"}`)
		config := Configuration{
			EventType: "push",
			Refs: []*RefFilter{
				{Type: FilterTypeExactMatch, Value: "main"},
				{Type: FilterTypeExactMatch, Value: "develop"},
				{Type: FilterTypeRegex, Value: "^feature/.*"},
			},
		}
		ctx := createWebhookRequest(body, "push", "test-secret", config)

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("push event with invalid regex - returns 500", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main"}`)
		config := Configuration{
			EventType: "push",
			Refs: []*RefFilter{
				{Type: "regex", Value: "[invalid("},
			},
		}
		ctx := createWebhookRequest(body, "push", "test-secret", config)

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusInternalServerError, code)
		assert.ErrorContains(t, err, "error checking branch filter")
		assert.Zero(t, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("push event missing ref field - returns 400", func(t *testing.T) {
		body := []byte(`{}`)
		ctx := createWebhookRequest(body, "push", "test-secret", Configuration{EventType: "push"})

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "failed to extract branch")
		assert.Zero(t, ctx.EventContext.(*DummyEventContext).Count())
	})

	t.Run("pull_request event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"opened","number":123}`)
		ctx := createWebhookRequest(body, "pull_request", "test-secret", Configuration{EventType: "pull_request"})

		code, err := trigger.HandleWebhook(ctx)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, ctx.EventContext.(*DummyEventContext).Count())
	})
}

type DummyWebhookContext struct {
	Secret string
}

func (w *DummyWebhookContext) GetSecret() ([]byte, error) {
	return []byte(w.Secret), nil
}

func (w *DummyWebhookContext) Setup(options *triggers.WebhookSetupOptions) error {
	return nil
}

type DummyEventContext struct {
	EmittedEvents []any
}

func (e *DummyEventContext) Emit(event any) error {
	e.EmittedEvents = append(e.EmittedEvents, event)
	return nil
}

func (e *DummyEventContext) Count() int {
	return len(e.EmittedEvents)
}

func Test__Setup(t *testing.T) {
	trigger := &GitHub{}

	t.Run("valid exact-match branch filter", func(t *testing.T) {
		ctx := triggers.TriggerContext{
			Configuration: map[string]any{
				"integration": uuid.NewString(),
				"repository":  "test-repo-id",
				"eventType":   "push",
				"branches": []map[string]any{
					{
						"type":  "exact-match",
						"value": "main",
					},
				},
			},
			MetadataContext:    &DummyMetadataContext{},
			IntegrationContext: &DummyIntegrationContext{},
			WebhookContext:     &DummyWebhookContext{},
		}

		err := trigger.Setup(ctx)
		assert.NoError(t, err)
	})

	t.Run("valid regex branch filter", func(t *testing.T) {
		ctx := triggers.TriggerContext{
			Configuration: map[string]any{
				"integration": uuid.NewString(),
				"repository":  "test-repo-id",
				"eventType":   "push",
				"branches": []map[string]any{
					{
						"type":  "regex",
						"value": "^feature/.*",
					},
				},
			},
			MetadataContext:    &DummyMetadataContext{},
			IntegrationContext: &DummyIntegrationContext{},
			WebhookContext:     &DummyWebhookContext{},
		}

		err := trigger.Setup(ctx)
		assert.NoError(t, err)
	})

	t.Run("invalid regex branch filter", func(t *testing.T) {
		ctx := triggers.TriggerContext{
			Configuration: map[string]any{
				"integration": uuid.NewString(),
				"repository":  "test-repo-id",
				"eventType":   "push",
				"branches": []map[string]any{
					{
						"type":  "regex",
						"value": "[invalid(",
					},
				},
			},
			MetadataContext:    &DummyMetadataContext{},
			IntegrationContext: &DummyIntegrationContext{},
			WebhookContext:     &DummyWebhookContext{},
		}

		err := trigger.Setup(ctx)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "invalid regex pattern")
	})
}

type DummyMetadataContext struct {
	data any
}

func (m *DummyMetadataContext) Get() any {
	if m.data == nil {
		return map[string]any{}
	}
	return m.data
}

func (m *DummyMetadataContext) Set(data any) {
	m.data = data
}

type DummyIntegrationContext struct{}

func (i *DummyIntegrationContext) GetIntegration(id string) (integrations.ResourceManager, error) {
	return &DummyResourceManager{}, nil
}

type DummyResourceManager struct{}

func (r *DummyResourceManager) Get(resourceType string, resourceID string) (integrations.Resource, error) {
	return &DummyResource{}, nil
}

func (r *DummyResourceManager) List(resourceType string) ([]integrations.Resource, error) {
	return []integrations.Resource{}, nil
}

func (r *DummyResourceManager) Status(resourceType, id string, parentResource integrations.Resource) (integrations.StatefulResource, error) {
	return nil, nil
}

func (r *DummyResourceManager) Cancel(resourceType, id string, parentResource integrations.Resource) error {
	return nil
}

func (r *DummyResourceManager) SetupWebhook(options integrations.WebhookOptions) (any, error) {
	return nil, nil
}

func (r *DummyResourceManager) CleanupWebhook(options integrations.WebhookOptions) error {
	return nil
}

type DummyResource struct{}

func (r *DummyResource) Id() string {
	return "test-resource-id"
}

func (r *DummyResource) Name() string {
	return "test-resource-name"
}

func (r *DummyResource) Type() string {
	return "repository"
}

func (r *DummyResource) URL() string {
	return "https://github.com/test/repo"
}
