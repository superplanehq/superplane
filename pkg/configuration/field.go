package configuration

const (
	/*
	 * Basic field types
	 */
	FieldTypeString      = "string"
	FieldTypeText        = "text"
	FieldTypeExpression  = "expression"
	FieldTypeXML         = "xml"
	FieldTypeNumber      = "number"
	FieldTypeBool        = "boolean"
	FieldTypeSelect      = "select"
	FieldTypeMultiSelect = "multi-select"
	FieldTypeList        = "list"
	FieldTypeObject      = "object"
	FieldTypeTime        = "time"
	FieldTypeDate        = "date"
	FieldTypeDateTime    = "datetime"
	FieldTypeTimezone    = "timezone"
	FieldTypeDaysOfWeek  = "days-of-week"
	FieldTypeTimeRange   = "time-range"

	/*
	 * Special field types
	 */
	FieldTypeDayInYear           = "day-in-year"
	FieldTypeCron                = "cron"
	FieldTypeUser                = "user"
	FieldTypeRole                = "role"
	FieldTypeGroup               = "group"
	FieldTypeIntegrationResource = "integration-resource"
	FieldTypeAnyPredicateList    = "any-predicate-list"
	FieldTypeGitRef              = "git-ref"
	FieldTypeSecretKey           = "secret-key"
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
	 * Optional placeholder shown in the UI input for this field
	 */
	Placeholder string `json:"placeholder,omitempty"`

	/*
	 * Type of the field. Supported types are defined by FieldType* constants above.
	 */
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     any    `json:"default"`
	Togglable   bool   `json:"togglable"`

	/*
	 * Whether the field is sensitive (e.g., password, API token)
	 */
	Sensitive bool `json:"sensitive"`

	/*
	 * Type-specific options for fields.
	 * The structure depends on the field type.
	 */
	TypeOptions *TypeOptions `json:"typeOptions,omitempty"`

	/*
	 * Used for controlling when the field is visible.
	 * No visibility conditions - always visible.
	 */
	VisibilityConditions []VisibilityCondition `json:"visibilityConditions,omitempty"`

	/*
	 * Used for controlling when the field is required based on other field values.
	 * If specified, the field is only required when these conditions are met.
	 */
	RequiredConditions []RequiredCondition `json:"requiredConditions,omitempty"`
}

/*
 * TypeOptions contains type-specific configuration for fields.
 */
type TypeOptions struct {
	Number           *NumberTypeOptions           `json:"number,omitempty"`
	String           *StringTypeOptions           `json:"string,omitempty"`
	Text             *TextTypeOptions             `json:"text,omitempty"`
	Expression       *ExpressionTypeOptions       `json:"expression,omitempty"`
	Select           *SelectTypeOptions           `json:"select,omitempty"`
	MultiSelect      *MultiSelectTypeOptions      `json:"multiSelect,omitempty"`
	Resource         *ResourceTypeOptions         `json:"resource,omitempty"`
	List             *ListTypeOptions             `json:"list,omitempty"`
	AnyPredicateList *AnyPredicateListTypeOptions `json:"anyPredicateList,omitempty"`
	Object           *ObjectTypeOptions           `json:"object,omitempty"`
	Time             *TimeTypeOptions             `json:"time,omitempty"`
	Date             *DateTypeOptions             `json:"date,omitempty"`
	DateTime         *DateTimeTypeOptions         `json:"dateTime,omitempty"`
}

/*
 * ResourceTypeOptions specifies which resource type to display
 */
type ResourceTypeOptions struct {
	Type           string `json:"type"`
	UseNameAsValue bool   `json:"useNameAsValue,omitempty"`

	//
	// If true, render as multi-select instead of single select
	//
	Multi bool `json:"multi,omitempty"`

	//
	// Additional parameters to be sent as query parameters to the /resources endpoint.
	// They can be static or come from values of other fields.
	//
	Parameters []ParameterRef `json:"parameters,omitempty"`
}

type ParameterRef struct {
	Name      string              `json:"name"`
	Value     *string             `json:"value"`
	ValueFrom *ParameterValueFrom `json:"valueFrom"`
}

type ParameterValueFrom struct {
	Field string `json:"field"`
}

/*
 * NumberTypeOptions specifies constraints for number fields
 */
type NumberTypeOptions struct {
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
}

/*
 * StringTypeOptions specifies constraints for string fields
 */
type StringTypeOptions struct {
	MinLength *int `json:"minLength,omitempty"`
	MaxLength *int `json:"maxLength,omitempty"`
}

type ExpressionTypeOptions struct {
	MinLength *int `json:"minLength,omitempty"`
	MaxLength *int `json:"maxLength,omitempty"`
}

type TextTypeOptions struct {
	MinLength *int `json:"minLength,omitempty"`
	MaxLength *int `json:"maxLength,omitempty"`
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
 * ListTypeOptions defines the structure of list items
 */
type ListTypeOptions struct {
	ItemDefinition *ListItemDefinition `json:"itemDefinition"`
	ItemLabel      string              `json:"itemLabel,omitempty"`
	MaxItems       *int                `json:"maxItems,omitempty"`
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
	Label string `json:"label"`
	Value string `json:"value"`
}

/*
 * ListItemDefinition defines the structure of items in an 'list' field
 */
type ListItemDefinition struct {
	Type   string  `json:"type"`
	Schema []Field `json:"schema,omitempty"`
}

type VisibilityCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}

type RequiredCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}
