package components

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/integrations"
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
	ProcessQueueItem(ctx ProcessQueueContext) error

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
	ID                    string
	WorkflowID            string
	Data                  any
	Configuration         any
	MetadataContext       MetadataContext
	ExecutionStateContext ExecutionStateContext
	RequestContext        RequestContext
	AuthContext           AuthContext
	IntegrationContext    IntegrationContext
}

/*
 * ExecutionContext allows the component
 * to control the state and metadata of each execution of it.
 */
type SetupContext struct {
	Configuration      any
	MetadataContext    MetadataContext
	RequestContext     RequestContext
	AuthContext        AuthContext
	IntegrationContext IntegrationContext
}

/*
 * IntegrationContext allows components to access integrations.
 */
type IntegrationContext interface {
	GetIntegration(ID string) (integrations.ResourceManager, error)
}

/*
 * MetadataContext allows components to store/retrieve
 * component-specific information about each execution.
 */
type MetadataContext interface {
	Get() any
	Set(any)
}

/*
 * ExecutionStateContext allows components to control execution lifecycle.
 */
type ExecutionStateContext interface {
	Pass(outputs map[string][]any) error
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
	Name                  string
	Configuration         any
	Parameters            map[string]any
	MetadataContext       MetadataContext
	ExecutionStateContext ExecutionStateContext
	AuthContext           AuthContext
	RequestContext        RequestContext
	IntegrationContext    IntegrationContext
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
	Input       any
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
