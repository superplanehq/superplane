package core

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
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
	HandleAction(ctx TriggerActionContext) (map[string]any, error)
}

type TriggerContext struct {
	Logger          *log.Entry
	Configuration   any
	HTTP            HTTPContext
	Metadata        MetadataContext
	Requests        RequestContext
	Events          EventContext
	Webhook         NodeWebhookContext
	AppInstallation AppInstallationContext
}

type EventContext interface {
	Emit(payloadType string, payload any) error
}

type TriggerActionContext struct {
	Name            string
	Parameters      map[string]any
	Configuration   any
	Logger          *log.Entry
	HTTP            HTTPContext
	Metadata        MetadataContext
	Requests        RequestContext
	Events          EventContext
	Webhook         NodeWebhookContext
	AppInstallation AppInstallationContext
}

type WebhookRequestContext struct {
	Body          []byte
	Headers       http.Header
	WorkflowID    string
	NodeID        string
	Configuration any
	Webhook       NodeWebhookContext
	Events        EventContext

	//
	// Return an execution context for a given execution,
	// through a referencing key-value pair.
	//
	FindExecutionByKV func(key string, value string) (*ExecutionContext, error)
}

type NodeWebhookContext interface {
	Setup() (string, error)
	GetSecret() ([]byte, error)
	ResetSecret() ([]byte, []byte, error)
	GetBaseURL() string
}
