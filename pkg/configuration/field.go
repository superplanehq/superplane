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
	Type               string `json:"type"`
	Description        string `json:"description"`
	Required           bool   `json:"required"`
	Default            any    `json:"default"`
	Togglable          bool   `json:"togglable"`
	DisallowExpression bool   `json:"disallow_expression"`

	/*
	 * Whether the field is sensitive (e.g., password, API token)
	 */
	Sensitive bool `json:"sensitive"`

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

	/*
	 * Used for controlling when the field is required based on other field values.
	 * If specified, the field is only required when these conditions are met.
	 */
	RequiredConditions []RequiredCondition `json:"required_conditions,omitempty"`

	/*
	 * Used for defining validation rules that compare this field with other fields.
	 * For example, ensuring startTime < endTime or startDateTime < endDateTime.
	 */
	ValidationRules []ValidationRule `json:"validation_rules,omitempty"`
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
	MultiSelect      *MultiSelectTypeOptions      `json:"multi_select,omitempty"`
	Resource         *ResourceTypeOptions         `json:"resource,omitempty"`
	List             *ListTypeOptions             `json:"list,omitempty"`
	AnyPredicateList *AnyPredicateListTypeOptions `json:"any_predicate_list,omitempty"`
	Object           *ObjectTypeOptions           `json:"object,omitempty"`
	Time             *TimeTypeOptions             `json:"time,omitempty"`
	Date             *DateTypeOptions             `json:"date,omitempty"`
	DateTime         *DateTimeTypeOptions         `json:"datetime,omitempty"`
	DayInYear        *DayInYearTypeOptions        `json:"day_in_year,omitempty"`
	Cron             *CronTypeOptions             `json:"cron,omitempty"`
	Timezone         *TimezoneTypeOptions         `json:"timezone,omitempty"`
}

/*
 * ResourceTypeOptions specifies which resource type to display
 */
type ResourceTypeOptions struct {
	Type           string `json:"type"`
	UseNameAsValue bool   `json:"use_name_as_value,omitempty"`

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
	ValueFrom *ParameterValueFrom `json:"value_from"`
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
	MinLength *int `json:"min_length,omitempty"`
	MaxLength *int `json:"max_length,omitempty"`
}

type ExpressionTypeOptions struct {
	MinLength *int `json:"min_length,omitempty"`
	MaxLength *int `json:"max_length,omitempty"`
}

type TextTypeOptions struct {
	MinLength *int `json:"min_length,omitempty"`
	MaxLength *int `json:"max_length,omitempty"`
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
 * DayInYearTypeOptions specifies format and constraints for day-in-year fields
 */
type DayInYearTypeOptions struct {
	Format string `json:"format,omitempty"` // Expected format, defaults to "MM/DD", e.g., "12/25"
}

/*
 * CronTypeOptions specifies constraints for cron expression fields
 */
type CronTypeOptions struct {
	AllowedFields []string `json:"allowed_fields,omitempty"` // Optional: limit which cron fields are allowed
}

/*
 * TimezoneTypeOptions specifies constraints for timezone fields
 */
type TimezoneTypeOptions struct {
	// Could add supported timezones list here if needed in the future
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
	ItemDefinition *ListItemDefinition `json:"item_definition"`
	ItemLabel      string              `json:"item_label,omitempty"`
	MaxItems       *int                `json:"max_items,omitempty"`
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

type RequiredCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}

const (
	ValidationRuleLessThan    = "less_than"
	ValidationRuleGreaterThan = "greater_than"
	ValidationRuleEqual       = "equal"
	ValidationRuleNotEqual    = "not_equal"
	ValidationRuleMaxLength   = "max_length"
	ValidationRuleMinLength   = "min_length"
)

type ValidationRule struct {
	Type        string `json:"type"`         // less_than, greater_than, equal, not_equal, max_length, min_length
	CompareWith string `json:"compare_with"` // field name to compare with (for field comparisons)
	Value       any    `json:"value"`        // static value to compare with (for direct validation)
	Message     string `json:"message"`      // custom error message
}
