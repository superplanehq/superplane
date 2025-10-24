package triggers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/integrations"
)

type Trigger interface {

	/*
	 * The unique identifier for the trigger.
	 * This is how nodes reference it, and is used for registration.
	 */
	Name() string

	/*
	 * The label for the trigger.
	 * This is how nodes are displayed in the UI.
	 */
	Label() string

	/*
	 * A good description of what the trigger does.
	 * Helpful for documentation and user interfaces.
	 */
	Description() string

	/*
	 * The icon for the trigger.
	 */
	Icon() string

	/*
	 * The color for the trigger.
	 */
	Color() string

	/*
	 * The configuration fields exposed by the trigger.
	 */
	Configuration() []components.ConfigurationField

	/*
	 * Handler for webhooks
	 */
	HandleWebhook(ctx WebhookRequestContext) (int, error)

	/*
	 * Setup the trigger.
	 */
	Setup(ctx TriggerContext) error

	/*
	 * Allows triggers to define custom actions.
	 */
	Actions() []components.Action

	/*
	 * Execution a custom action - defined in Actions() for a trigger.
	 */
	HandleAction(ctx TriggerActionContext) error
}

type TriggerContext struct {
	Configuration      any
	MetadataContext    components.MetadataContext
	RequestContext     components.RequestContext
	EventContext       EventContext
	WebhookContext     WebhookContext
	IntegrationContext IntegrationContext
}

type IntegrationContext interface {
	GetIntegration(ID string) (integrations.ResourceManager, error)
}

type WebhookContext interface {
	Setup(options *WebhookSetupOptions) error
	GetSecret() ([]byte, error)
}

type WebhookSetupOptions struct {
	IntegrationID *uuid.UUID
	Resource      integrations.Resource
	Configuration any
}

type EventContext interface {
	Emit(data any) error
}

type TriggerActionContext struct {
	Name            string
	Parameters      map[string]any
	Configuration   any
	MetadataContext components.MetadataContext
	RequestContext  components.RequestContext
	EventContext    EventContext
	WebhookContext  WebhookContext
}

type WebhookRequestContext struct {
	Body           []byte
	Headers        http.Header
	Configuration  any
	WebhookContext WebhookContext
	EventContext   EventContext
}
