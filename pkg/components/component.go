package components

const DefaultBranchName = "default"

type ConfigurationField struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

type Component interface {

	/*
	 * The unique identifier for the component.
	 * This is how nodes reference it, and is used for registration.
	 */
	Name() string

	/*
	 * A good description of what the component does.
	 * Helpful for documentation and user interfaces.
	 */
	Description() string

	/*
	 * The output branches used by the component.
	 * If none is returned, the 'default' one is used.
	 */
	OutputBranches(configuration any) []string

	/*
	 * The configuration fields exposed by the component.
	 */
	Configuration() []ConfigurationField

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

/*
 * ExecutionContext allows the component
 * to control the state and metadata of each execution of it.
 */
type ExecutionContext struct {
	Data                  any
	Configuration         any
	MetadataContext       MetadataContext
	ExecutionStateContext ExecutionStateContext
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
	Wait() error
	Finish(outputs map[string][]any) error
	Fail(reason string) error
}

/*
 * Custom action definition for a component.
 */
type Action struct {
	Name        string
	Description string
	Parameters  []ConfigurationField
}

/*
 * ActionContext allows the component to execute a custom action,
 * and control the state and metadata of each execution of it.
 */
type ActionContext struct {
	Name                  string
	Parameters            map[string]any
	MetadataContext       MetadataContext
	ExecutionStateContext ExecutionStateContext
}
