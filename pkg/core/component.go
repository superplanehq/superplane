package core

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
)

var DefaultOutputChannel = OutputChannel{Name: "default", Label: "Default"}

var ErrSecretKeyNotFound = errors.New("secret or key not found")

type Component interface {

	/*
	 * The unique identifier for the component.
	 * This is how nodes reference it, and is used for registration.
	 */
	Name() string

	/*
	 * The label for the component.
	 * This is how nodes are displayed in the UI.
	 */
	Label() string

	/*
	 * A good description of what the component does.
	 * Helpful for documentation and user interfaces.
	 */
	Description() string

	/*
	 * Detailed markdown documentation explaining how to use the component.
	 * This should provide in-depth information about the component's purpose,
	 * configuration options, use cases, and examples.
	 */
	Documentation() string

	/*
	 * The icon for the component.
	 * This is used in the UI to represent the component.
	 */
	Icon() string

	/*
	 * The color for the component.
	 * This is used in the UI to represent the component.
	 */
	Color() string

	/*
	 * Example output data for the component.
	 */
	ExampleOutput() map[string]any

	/*
	 * The output channels used by the component.
	 * If none is returned, the 'default' one is used.
	 */
	OutputChannels(configuration any) []OutputChannel

	/*
	 * The configuration fields exposed by the component.
	 */
	Configuration() []configuration.Field

	/*
	 * Setup the component.
	 */
	Setup(ctx SetupContext) error

	/*
	 * ProcessQueueItem is called when a queue item for this component's node
	 * is ready to be processed. Implementations should create the appropriate
	 * execution or handle the item synchronously using the provided context.
	 */
	ProcessQueueItem(ctx ProcessQueueContext) (*uuid.UUID, error)

	/*
	 * Passes full execution control to the component.
	 *
	 * Component execution has full control over the execution state,
	 * so it is the responsibility of the component to control it.
	 *
	 * Components should finish the execution or move it to waiting state.
	 * Components can also implement async components by combining Execute() and HandleAction().
	 */
	Execute(ctx ExecutionContext) error

	/*
	 * Allows components to define custom actions
	 * that can be called on specific executions of the component.
	 */
	Actions() []Action

	/*
	 * Execution a custom action - defined in Actions() -
	 * on a specific execution of the component.
	 */
	HandleAction(ctx ActionContext) error

	/*
	 * Handler for webhooks.
	 */
	HandleWebhook(ctx WebhookRequestContext) (int, error)

	/*
	 * Cancel allows components to handle cancellation of executions.
	 * Default behavior does nothing. Components can override to perform
	 * cleanup or cancel external resources.
	 */
	Cancel(ctx ExecutionContext) error

	/*
	 * Cleanup allows components to clean up resources after being removed from a canvas.
	 * Default behavior does nothing. Components can override to perform cleanup.
	 */
	Cleanup(ctx SetupContext) error
}

type OutputChannel struct {
	Name        string
	Label       string
	Description string
}

/*
 * ExecutionContext allows the component
 * to control the state and metadata of each execution of it.
 */
type ExecutionContext struct {
	ID             uuid.UUID
	WorkflowID     string
	OrganizationID string
	NodeID         string
	SourceNodeID   string
	BaseURL        string
	Data           any
	Configuration  any
	ExpressionEnv  func(expression string) (map[string]any, error)
	Logger         *log.Entry
	HTTP           HTTPContext
	Metadata       MetadataContext
	NodeMetadata   MetadataContext
	ExecutionState ExecutionStateContext
	Requests       RequestContext
	Auth           AuthContext
	Integration    IntegrationContext
	Notifications  NotificationContext
	Secrets        SecretsContext
	CanvasMemory   CanvasMemoryContext
	Webhook        NodeWebhookContext
}

/*
 * Components / triggers / applications should always
 * use this context instead of the net/http directly for executing HTTP requests.
 *
 * This makes it easy for us to write unit tests for the implementations,
 * and also makes it easier to control HTTP timeouts for everything in one place.
 */
type HTTPContext interface {
	Do(*http.Request) (*http.Response, error)
}

/*
 * ExecutionContext allows the component
 * to control the state and metadata of each execution of it.
 */
type SetupContext struct {
	Logger        *log.Entry
	Configuration any
	HTTP          HTTPContext
	Metadata      MetadataContext
	Requests      RequestContext
	Auth          AuthContext
	Integration   IntegrationContext
	Webhook       NodeWebhookContext
}

/*
 * MetadataContext allows components to store/retrieve
 * component-specific information about each execution.
 */
type MetadataContext interface {
	Get() any
	Set(any) error
}

type CanvasMemoryContext interface {
	Add(namespace string, values any) error
}

/*
 * ExecutionStateContext allows components to control execution lifecycle.
 */
type ExecutionStateContext interface {
	IsFinished() bool
	SetKV(key, value string) error

	/*
	 * Pass the execution, emitting a payload to the specified channel.
	 */
	Emit(channel, payloadType string, payloads []any) error

	/*
	 * Pass the execution, without emitting any payloads from it.
	 */
	Pass() error

	/*
	 * Fails the execution.
	 * No payloads are emitted.
	 */
	Fail(reason, message string) error
}

/*
 * RequestContext allows the execution to schedule
 * work with the processing engine.
 */
type RequestContext interface {

	//
	// Allows the scheduling of a certain component action at a later time
	//
	ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error
}

/*
 * Custom action definition for a component.
 */
type Action struct {
	Name           string
	Description    string
	UserAccessible bool
	Parameters     []configuration.Field
}

/*
 * ActionContext allows the component to execute a custom action,
 * and control the state and metadata of each execution of it.
 */
type ActionContext struct {
	Name           string
	Configuration  any
	Parameters     map[string]any
	Logger         *log.Entry
	HTTP           HTTPContext
	Metadata       MetadataContext
	ExecutionState ExecutionStateContext
	Auth           AuthContext
	Requests       RequestContext
	Integration    IntegrationContext
	Notifications  NotificationContext
	Secrets        SecretsContext
}

/*
 * ProcessQueueContext is provided to components to process a node's queue item.
 * It mirrors the data the queue worker would otherwise use to create executions.
 */
type ProcessQueueContext struct {
	WorkflowID    string
	NodeID        string
	RootEventID   string
	EventID       string
	SourceNodeID  string
	Configuration any
	Input         any
	ExpressionEnv func(expression string) (map[string]any, error)

	//
	// Deletes the queue item
	//
	DequeueItem func() error

	//
	// Updates the state of the node
	//
	UpdateNodeState func(state string) error

	//
	// Creates a pending execution for this queue item.
	//
	CreateExecution func() (*ExecutionContext, error)

	//
	// Finds an execution by a key-value pair.
	// Returns an ExecutionContext.
	//
	FindExecutionByKV func(key string, value string) (*ExecutionContext, error)

	//
	// DefaultProcessing performs the default processing for the queue item.
	// Convenience method to avoid boilerplate in components that just want default behavior,
	// where an execution is created and the item is dequeued.
	//
	DefaultProcessing func() (*uuid.UUID, error)

	//
	// CountDistinctIncomingSources returns the number of distinct upstream
	// source nodes connected to this node (ignoring multiple channels from the
	// same source)
	//
	CountDistinctIncomingSources func() (int, error)
}

type AuthContext interface {
	AuthenticatedUser() *User
	GetUser(id uuid.UUID) (*User, error)
	HasRole(role string) (bool, error)
	InGroup(group string) (bool, error)
}

type NotificationReceivers struct {
	Emails []string
	Groups []string
	Roles  []string
}

type NotificationContext interface {
	Send(title, body, url, urlLabel string, receivers NotificationReceivers) error
}

type SecretsContext interface {
	GetKey(secretName, keyName string) ([]byte, error)
}

type User struct {
	ID    string `mapstructure:"id" json:"id"`
	Name  string `mapstructure:"name" json:"name"`
	Email string `mapstructure:"email" json:"email"`
}
