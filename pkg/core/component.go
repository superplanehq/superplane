package core

import (
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
)

var DefaultOutputChannel = OutputChannel{Name: "default", Label: "Default"}

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
	ProcessQueueItem(ctx ProcessQueueContext) (*models.WorkflowNodeExecution, error)

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
	ID                     string
	WorkflowID             string
	Data                   any
	Configuration          any
	Logger                 *log.Entry
	MetadataContext        MetadataContext
	NodeMetadataContext    MetadataContext
	ExecutionStateContext  ExecutionStateContext
	RequestContext         RequestContext
	AuthContext            AuthContext
	IntegrationContext     IntegrationContext
	AppInstallationContext AppInstallationContext
}

/*
 * ExecutionContext allows the component
 * to control the state and metadata of each execution of it.
 */
type SetupContext struct {
	Logger                 *log.Entry
	Configuration          any
	MetadataContext        MetadataContext
	RequestContext         RequestContext
	AuthContext            AuthContext
	IntegrationContext     IntegrationContext
	AppInstallationContext AppInstallationContext
}

/*
 * IntegrationContext allows components to access integrations.
 */
type IntegrationContext interface {
	GetIntegration(ID string) (integrations.ResourceManager, error)
}

type Webhook struct {
	ID            uuid.UUID
	Configuration any
}

/*
 * MetadataContext allows components to store/retrieve
 * component-specific information about each execution.
 */
type MetadataContext interface {
	Get() any
	Set(any) error
}

/*
 * ExecutionStateContext allows components to control execution lifecycle.
 */
type ExecutionStateContext interface {
	IsFinished() bool
	Pass(outputs []Output) error
	Fail(reason, message string) error
	SetKV(key, value string) error
}

type Output struct {
	Channel  string
	Payloads []Payload
}

type Payload struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
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
	Name                   string
	Configuration          any
	Parameters             map[string]any
	Logger                 *log.Entry
	MetadataContext        MetadataContext
	ExecutionStateContext  ExecutionStateContext
	AuthContext            AuthContext
	RequestContext         RequestContext
	IntegrationContext     IntegrationContext
	AppInstallationContext AppInstallationContext
}

/*
 * ProcessQueueContext is provided to components to process a node's queue item.
 * It mirrors the data the queue worker would otherwise use to create executions.
 */
type ProcessQueueContext struct {
	// IDs and configuration
	WorkflowID    string
	NodeID        string
	Configuration any

	// Input event data and references
	RootEventID string
	EventID     string
	// SourceNodeID is the upstream node id that produced the event
	SourceNodeID string
	Input        any

	// CreateExecution creates a pending execution for this queue item.
	CreateExecution func() (uuid.UUID, error)

	// DequeueItem marks the queue item as processed.
	DequeueItem func() error

	// UpdateNodeState updates the state of the node.
	UpdateNodeState func(state string) error

	// DefaultProcessing performs the default processing for the queue item.
	// Convenience method to avoid boilerplate in components that just want default behavior,
	// where an execution is created and the item is dequeued.
	DefaultProcessing func() (*models.WorkflowNodeExecution, error)

	// GetExecutionMetadata retrieves the execution metadata for a given execution ID.
	GetExecutionMetadata func(uuid.UUID) (map[string]any, error)
	SetExecutionMetadata func(uuid.UUID, any) error

	// CountIncomingEdges returns the number of incoming edges for this node
	CountIncomingEdges func() (int, error)

	// CountDistinctIncomingSources returns the number of distinct upstream
	// source nodes connected to this node (ignoring multiple channels from the
	// same source)
	CountDistinctIncomingSources func() (int, error)

	// PassExecution marks the execution as passed with the provided outputs.
	PassExecution func(execID uuid.UUID, outputs map[string][]any) (*models.WorkflowNodeExecution, error)

	// FailExecution marks the execution as failed with the provided reason and message.
	FailExecution func(execID uuid.UUID, reason, message string) (*models.WorkflowNodeExecution, error)

	FindExecutionIDByKV func(key string, value string) (uuid.UUID, bool, error)
	SetExecutionKV      func(execID uuid.UUID, key string, value string) error
}

type AuthContext interface {
	AuthenticatedUser() *User
	GetUser(id uuid.UUID) (*User, error)
	HasRole(role string) (bool, error)
	InGroup(group string) (bool, error)
}

type User struct {
	ID    string `mapstructure:"id" json:"id"`
	Name  string `mapstructure:"name" json:"name"`
	Email string `mapstructure:"email" json:"email"`
}
