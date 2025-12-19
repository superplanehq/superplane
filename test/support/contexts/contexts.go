package contexts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

type EventContext struct {
	EmittedEvents []any
}

func (e *EventContext) Emit(event any) error {
	e.EmittedEvents = append(e.EmittedEvents, event)
	return nil
}

func (e *EventContext) Count() int {
	return len(e.EmittedEvents)
}

type WebhookContext struct {
	Secret string
}

func (w *WebhookContext) GetSecret() ([]byte, error) {
	return []byte(w.Secret), nil
}

func (w *WebhookContext) Setup(options *core.WebhookSetupOptions) error {
	return nil
}

type MetadataContext struct {
	Metadata any
}

func (m *MetadataContext) Get() any {
	return m.Metadata
}

func (m *MetadataContext) Set(metadata any) {
	m.Metadata = metadata
}

type AppInstallationContext struct {
	Metadata         any
	State            string
	StateDescription string
	BrowserAction    *core.BrowserAction
	Webhooks         []core.Webhook
	Secrets          map[string]core.InstallationSecret
	WebhookRequests  []any
}

func (c *AppInstallationContext) ID() uuid.UUID {
	return uuid.New()
}

func (c *AppInstallationContext) GetMetadata() any {
	return c.Metadata
}

func (c *AppInstallationContext) SetMetadata(metadata any) {
	c.Metadata = metadata
}

func (c *AppInstallationContext) GetConfig(name string) ([]byte, error) {
	return nil, nil
}

func (c *AppInstallationContext) GetState() string {
	return ""
}

func (c *AppInstallationContext) SetState(state, stateDescription string) {
	c.State = state
	c.StateDescription = stateDescription
}

func (c *AppInstallationContext) NewBrowserAction(action core.BrowserAction) {
	c.BrowserAction = &action
}

func (c *AppInstallationContext) RemoveBrowserAction() {
	c.BrowserAction = nil
}

func (c *AppInstallationContext) SetSecret(name string, value []byte) error {
	c.Secrets[name] = core.InstallationSecret{Name: name, Value: value}
	return nil
}

func (c *AppInstallationContext) GetSecrets() ([]core.InstallationSecret, error) {
	secrets := make([]core.InstallationSecret, 0, len(c.Secrets))
	for _, secret := range c.Secrets {
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

func (c *AppInstallationContext) ListWebhooks() ([]core.Webhook, error) {
	return []core.Webhook{}, nil
}

func (c *AppInstallationContext) CreateWebhook(configuration any) error {
	c.Webhooks = append(c.Webhooks, core.Webhook{ID: uuid.New(), Configuration: configuration})
	return nil
}

func (c *AppInstallationContext) RequestWebhook(configuration any) error {
	c.WebhookRequests = append(c.WebhookRequests, configuration)
	return nil
}

func (c *AppInstallationContext) AssociateWebhook(webhookID uuid.UUID) {
	// TODO: I don't like this method
}
