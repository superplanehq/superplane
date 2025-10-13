package components

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
	 * The output channels used by the component.
	 * If none is returned, the 'default' one is used.
	 */
	OutputChannels(configuration any) []OutputChannel

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
	Pass(outputs map[string][]any) error
	Fail(reason, message string) error
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

const (
	FieldTypeString      = "string"
	FieldTypeNumber      = "number"
	FieldTypeBool        = "boolean"
	FieldTypeSelect      = "select"
	FieldTypeMultiSelect = "multi_select"
	FieldTypeDate        = "date"
	FieldTypeURL         = "url"
	FieldTypeList        = "list"
	FieldTypeObject      = "object"
)

type ConfigurationField struct {
	/*
	 * Unique name identifier for the field
	 */
	Name string `json:"name"`

	/*
	 * Human-readable label for the field (displayed in forms)
	 */
	Label string `json:"label"`

	/*
	 * Type of the field. Supported types are:
	 * - string
	 * - number
	 * - boolean
	 * - select
	 * - multi_select
	 * - date
	 * - url
	 * - list
	 * - object
	 */
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     any    `json:"default"`

	/*
	 * Used for select / multi_select types
	 */
	Options []FieldOption `json:"options,omitempty"`

	/*
	 * Used for number type to specify minimum value
	 */
	Min *int `json:"min,omitempty"`

	/*
	 * Used for number type to specify maximum value
	 */
	Max *int `json:"max,omitempty"`

	/*
	 * Defines structures of items on a 'list' type.
	 */
	ListItem *ListItemDefinition `json:"list_item,omitempty"`

	/*
	 * Schema allows us to define nested object structures for 'object' type.
	 */
	Schema []ConfigurationField `json:"schema,omitempty"`
}

/*
 * FieldOption represents a selectable option for select / multi_select field types
 */
type FieldOption struct {
	Label string
	Value string
}

/*
 * ListItemDefinition defines the structure of items in an 'list' field
 */
type ListItemDefinition struct {
	Type   string
	Schema []ConfigurationField
}
