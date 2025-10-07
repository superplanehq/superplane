package primitives

const DefaultBranchName = "default"

type ConfigurationField struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

type Primitive interface {
	Name() string
	Description() string
	Outputs(configuration any) []string
	Configuration() []ConfigurationField

	//
	// Executes the primitive.
	// Primitive execution has full control over the execution state,
	// so Execute() should finish the execution or move it to waiting state.
	// Execute() can be combined with HandleAction() to implement async primitives.
	//
	Execute(ctx ExecutionContext) error

	//
	// Allows primitives to define custom actions
	// that can be called for executions of the primitive.
	//
	Actions() []Action

	//
	// Handles custom actions for executions of the primitive.
	//
	HandleAction(ctx ActionContext) error
}

type ExecutionContext struct {
	Data          any
	Configuration any
	Metadata      MetadataContext
	State         ExecutionStateContext
}

// Metadata allows primitives to store/retrieve
// metadata about each execution.
type MetadataContext interface {
	Get(key string) (any, bool)
	Set(key string, value any)
	GetAll() map[string]any
}

// ExecutionState allows primitives to control execution lifecycle
type ExecutionStateContext interface {
	Wait() error
	Finish(outputs map[string][]any) error
	Fail(reason string) error
}

type Action struct {
	Name        string
	Description string
	Parameters  []ConfigurationField
}

type ActionContext struct {
	Name       string
	Parameters map[string]any
	Metadata   MetadataContext
	State      ExecutionStateContext
}

type Result struct {
	Branches map[string][]any
}
