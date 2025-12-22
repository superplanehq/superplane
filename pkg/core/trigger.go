package core

import (
	"net/http"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
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
	Configuration() []configuration.Field

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
	Actions() []Action

	/*
	 * Execution a custom action - defined in Actions() for a trigger.
	 */
	HandleAction(ctx TriggerActionContext) error
}

type TriggerContext struct {
	Logger                 *log.Entry
	Configuration          any
	MetadataContext        MetadataContext
	RequestContext         RequestContext
	EventContext           EventContext
	WebhookContext         WebhookContext
	IntegrationContext     IntegrationContext
	AppInstallationContext AppInstallationContext
}

type WebhookSetupOptions struct {
	IntegrationID     *uuid.UUID
	AppInstallationID *uuid.UUID
	Resource          integrations.Resource
	Configuration     any
}

type EventContext interface {
	Emit(data any) error
}

type TriggerActionContext struct {
	Name                   string
	Parameters             map[string]any
	Configuration          any
	MetadataContext        MetadataContext
	RequestContext         RequestContext
	EventContext           EventContext
	WebhookContext         WebhookContext
	AppInstallationContext AppInstallationContext
}

type WebhookRequestContext struct {
	Body           []byte
	Headers        http.Header
	WorkflowID     string
	NodeID         string
	Configuration  any
	WebhookContext WebhookContext
	EventContext   EventContext

	//
	// Return an execution context for a given execution,
	// through a referencing key-value pair.
	//
	FindExecutionByKV func(key string, value string) (*ExecutionContext, error)
}

type WebhookContext interface {
	Setup(options *WebhookSetupOptions) error
	GetSecret() ([]byte, error)
}
