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
	 * Detailed markdown documentation explaining how to use the trigger.
	 * This should provide in-depth information about the trigger's purpose,
	 * configuration options, use cases, and examples.
	 */
	Documentation() string

	/*
	 * The icon for the trigger.
	 */
	Icon() string

	/*
	 * The color for the trigger.
	 */
	Color() string

	/*
	 * Example input data for the trigger.
	 */
	ExampleData() map[string]any

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

	/*
	 * Cleanup allows triggers to clean up resources after being removed from a canvas.
	 * Default behavior does nothing. Triggers can override to perform cleanup.
	 */
	Cleanup(ctx TriggerContext) error
}

type TriggerContext struct {
	Logger        *log.Entry
	Configuration any
	HTTP          HTTPContext
	Metadata      MetadataContext
	Requests      RequestContext
	Events        EventContext
	Webhook       NodeWebhookContext
	Integration   IntegrationContext
}

type EventContext interface {
	Emit(payloadType string, payload any) error
}

type TriggerActionContext struct {
	Name          string
	Parameters    map[string]any
	Configuration any
	Logger        *log.Entry
	HTTP          HTTPContext
	Metadata      MetadataContext
	Requests      RequestContext
	Events        EventContext
	Webhook       NodeWebhookContext
	Integration   IntegrationContext
}

type WebhookRequestContext struct {
	Body          []byte
	Headers       http.Header
	WorkflowID    string
	NodeID        string
	Configuration any
	Metadata      MetadataContext
	Logger        *log.Entry
	Webhook       NodeWebhookContext
	Events        EventContext
	Integration   IntegrationContext

	//
	// Return an execution context for a given execution,
	// through a referencing key-value pair.
	//
	FindExecutionByKV func(key string, value string) (*ExecutionContext, error)

	// Do not make HTTP calls as part of handling the webhook. This is useful for
	// retrieving more data that is not part of the webhook payload.
	HTTP HTTPContext
}

type NodeWebhookContext interface {
	Setup() (string, error)
	GetSecret() ([]byte, error)
	ResetSecret() ([]byte, []byte, error)
	GetBaseURL() string
}
