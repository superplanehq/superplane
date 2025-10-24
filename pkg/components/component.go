package components

import "time"

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
	RequestContext        RequestContext
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
	Parameters     []ConfigurationField
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
}

const (
	FieldTypeString              = "string"
	FieldTypeNumber              = "number"
	FieldTypeBool                = "boolean"
	FieldTypeSelect              = "select"
	FieldTypeMultiSelect         = "multi-select"
	FieldTypeIntegration         = "integration"
	FieldTypeIntegrationResource = "integration-resource"
	FieldTypeURL                 = "url"
	FieldTypeList                = "list"
	FieldTypeObject              = "object"
	FieldTypeTime                = "time"
	FieldTypeUser                = "user"
	FieldTypeRole                = "role"
	FieldTypeGroup               = "group"
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
	 * - integration
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
	 * Type-specific options for fields.
	 * The structure depends on the field type.
	 */
	TypeOptions *TypeOptions `json:"type_options,omitempty"`

	/*
	 * Used for controlling when the field is visible.
	 * No visibility conditions - always visible.
	 */
	VisibilityConditions []VisibilityCondition `json:"visibility_conditions,omitempty"`
}

/*
 * TypeOptions contains type-specific configuration for fields.
 */
type TypeOptions struct {
	Number      *NumberTypeOptions      `json:"number,omitempty"`
	Select      *SelectTypeOptions      `json:"select,omitempty"`
	MultiSelect *MultiSelectTypeOptions `json:"multi_select,omitempty"`
	Integration *IntegrationTypeOptions `json:"integration,omitempty"`
	Resource    *ResourceTypeOptions    `json:"resource,omitempty"`
	List        *ListTypeOptions        `json:"list,omitempty"`
	Object      *ObjectTypeOptions      `json:"object,omitempty"`
	Time        *TimeTypeOptions        `json:"time,omitempty"`
}

/*
 * ResourceTypeOptions specifies which resource type to display
 */
type ResourceTypeOptions struct {
	Type string `json:"type"`
}

/*
 * NumberTypeOptions specifies constraints for number fields
 */
type NumberTypeOptions struct {
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
}

/*
 * TimeTypeOptions specifies format and constraints for time fields
 */
type TimeTypeOptions struct {
	Format string `json:"format,omitempty"` // Expected format, e.g., "HH:MM", "HH:MM:SS"
}

/*
 * SelectTypeOptions specifies options for select fields
 */
type SelectTypeOptions struct {
	Options []FieldOption `json:"options"`
}

/*
 * MultiSelectTypeOptions specifies options for multi_select fields
 */
type MultiSelectTypeOptions struct {
	Options []FieldOption `json:"options"`
}

/*
 * IntegrationTypeOptions specifies which integration type to display
 */
type IntegrationTypeOptions struct {
	Type string `json:"type"`
}

/*
 * ListTypeOptions defines the structure of list items
 */
type ListTypeOptions struct {
	ItemDefinition *ListItemDefinition `json:"item_definition"`
}

/*
 * ObjectTypeOptions defines the schema for object fields
 */
type ObjectTypeOptions struct {
	Schema []ConfigurationField `json:"schema"`
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

type VisibilityCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}
