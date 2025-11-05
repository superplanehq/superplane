package configuration

const (
	FieldTypeString              = "string"
	FieldTypeNumber              = "number"
	FieldTypeBool                = "boolean"
	FieldTypeSelect              = "select"
	FieldTypeMultiSelect         = "multi-select"
	FieldTypeIntegration         = "integration"
	FieldTypeIntegrationResource = "integration-resource"
	FieldTypeList                = "list"
	FieldTypeObject              = "object"
	FieldTypeTime                = "time"
	FieldTypeDate                = "date"
	FieldTypeDateTime            = "datetime"
	FieldTypeUser                = "user"
	FieldTypeRole                = "role"
	FieldTypeGroup               = "group"
)

type Field struct {
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
	Date        *DateTypeOptions        `json:"date,omitempty"`
	DateTime    *DateTimeTypeOptions    `json:"datetime,omitempty"`
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
 * DateTypeOptions specifies format and constraints for date fields
 */
type DateTypeOptions struct {
	Format string `json:"format,omitempty"` // Expected format, e.g., "YYYY-MM-DD", "MM/DD/YYYY"
}

/*
 * DateTimeTypeOptions specifies format and constraints for datetime fields
 */
type DateTimeTypeOptions struct {
	Format string `json:"format,omitempty"` // Expected format, e.g., "2006-01-02T15:04", "YYYY-MM-DDTHH:MM"
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
	Schema []Field `json:"schema"`
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
	Schema []Field
}

type VisibilityCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}
