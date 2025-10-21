package triggers

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/components"
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
	 * The configuration fields exposed by the trigger.
	 */
	Configuration() []components.ConfigurationField

	/*
	 * Setup the trigger.
	 */
	Setup(ctx SetupContext) error

	/*
	 * Starts the trigger
	 */
	Start(ctx TriggerContext) error

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
	Configuration   any
	MetadataContext components.MetadataContext
	RequestContext  components.RequestContext
	EventContext    EventContext
	WebhookContext  WebhookContext
}

type WebhookContext interface {
	RegisterActionCall(actionName string) error
	Create() error
}

type SetupContext struct {
	Configuration   any
	MetadataContext components.MetadataContext
	WebhookContext  WebhookContext
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
	WebhookContext  WebhookRequestContext
}

type WebhookRequestContext struct {
	Request  *http.Request
	Response http.ResponseWriter
}
