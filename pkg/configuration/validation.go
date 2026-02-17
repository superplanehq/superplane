package configuration

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/robfig/cron/v3"
)

var expressionPlaceholderRegex = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

func ValidateConfiguration(fields []Field, config map[string]any) error {
	for _, field := range fields {
		value, exists := config[field.Name]

		// Check if field is required (either always or conditionally)
		isRequired := field.Required
		if !isRequired && len(field.RequiredConditions) > 0 {
			isRequired = isRequiredByCondition(field, config)
		}

		if isRequired && (!exists || value == nil) {
			return fmt.Errorf("field '%s' is required", field.Name)
		}

		if !exists || value == nil {
			continue
		}

		err := validateFieldValue(field, value)
		if err != nil {
			return fmt.Errorf("field '%s': %w", field.Name, err)
		}

		// Validate field comparison rules
		err = validateFieldRules(field, value, config)
		if err != nil {
			return fmt.Errorf("field '%s': %w", field.Name, err)
		}
	}

	return nil
}

func validateNumber(field Field, value any) error {
	var num float64
	switch v := value.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	case int32:
		num = float64(v)
	case int64:
		num = float64(v)
	default:
		return fmt.Errorf("must be a number")
	}

	if field.TypeOptions == nil || field.TypeOptions.Number == nil {
		return nil
	}

	options := field.TypeOptions.Number
	if options.Min != nil && num < float64(*options.Min) {
		return fmt.Errorf("must be at least %d", *options.Min)
	}

	if options.Max != nil && num > float64(*options.Max) {
		return fmt.Errorf("must be at most %d", *options.Max)
	}

	return nil
}

func validateString(field Field, value any) error {
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if field.TypeOptions == nil || field.TypeOptions.String == nil {
		return nil
	}

	options := field.TypeOptions.String
	textLength := len(text)

	if options.MinLength != nil && textLength < *options.MinLength {
		return fmt.Errorf("must be at least %d characters", *options.MinLength)
	}

	if options.MaxLength != nil && textLength > *options.MaxLength {
		return fmt.Errorf("must be at most %d characters", *options.MaxLength)
	}

	return nil
}

func validateExpression(field Field, value any) error {
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be an expression string")
	}

	if field.TypeOptions == nil || field.TypeOptions.Expression == nil {
		return nil
	}

	options := field.TypeOptions.Expression
	textLength := len(text)

	if options.MinLength != nil && textLength < *options.MinLength {
		return fmt.Errorf("must be at least %d characters", *options.MinLength)
	}

	if options.MaxLength != nil && textLength > *options.MaxLength {
		return fmt.Errorf("must be at most %d characters", *options.MaxLength)
	}

	return nil
}

func validateText(field Field, value any) error {
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if field.TypeOptions == nil || field.TypeOptions.Text == nil {
		return nil
	}

	options := field.TypeOptions.Text
	textLength := len(text)

	if options.MinLength != nil && textLength < *options.MinLength {
		return fmt.Errorf("must be at least %d characters", *options.MinLength)
	}

	if options.MaxLength != nil && textLength > *options.MaxLength {
		return fmt.Errorf("must be at most %d characters", *options.MaxLength)
	}

	return nil
}

func validateSelect(field Field, value any) error {
	selected, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if field.TypeOptions == nil || field.TypeOptions.Select == nil {
		return nil
	}

	options := field.TypeOptions.Select
	if len(options.Options) == 0 {
		return nil
	}

	valid := slices.ContainsFunc(options.Options, func(opt FieldOption) bool {
		return opt.Value == selected
	})

	if !valid {
		validValues := make([]string, len(options.Options))
		for i, opt := range options.Options {
			validValues[i] = opt.Value
		}

		return fmt.Errorf("must be one of: %s", strings.Join(validValues, ", "))
	}

	return nil
}

func validateMultiSelect(field Field, value any) error {
	selectedValues, ok := value.([]any)
	if !ok {
		return fmt.Errorf("must be a list of values")
	}

	if field.TypeOptions == nil || field.TypeOptions.MultiSelect == nil {
		return nil
	}

	typeOptions := field.TypeOptions.MultiSelect
	if len(typeOptions.Options) == 0 {
		return nil
	}

	for _, selectedValue := range selectedValues {
		v, ok := selectedValue.(string)
		if !ok {
			return fmt.Errorf("all items must be strings")
		}

		valid := slices.ContainsFunc(typeOptions.Options, func(opt FieldOption) bool {
			return opt.Value == v
		})

		if valid {
			continue
		}

		validValues := make([]string, len(typeOptions.Options))
		for i, opt := range typeOptions.Options {
			validValues[i] = opt.Value
		}

		return fmt.Errorf("value '%s' must be one of: %s", v, strings.Join(validValues, ", "))
	}

	return nil
}

func validateDaysOfWeek(_ Field, value any) error {
	var selectedValues []string
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			dayValue, ok := item.(string)
			if !ok {
				return fmt.Errorf("all items must be strings")
			}
			selectedValues = append(selectedValues, dayValue)
		}
	case []string:
		selectedValues = append(selectedValues, v...)
	default:
		return fmt.Errorf("must be a list of values")
	}

	if len(selectedValues) == 0 {
		return fmt.Errorf("must contain at least one day")
	}

	validDays := []string{
		"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
	}

	for _, day := range selectedValues {
		if !slices.Contains(validDays, day) {
			return fmt.Errorf("invalid day '%s': must be one of monday, tuesday, wednesday, thursday, friday, saturday, sunday", day)
		}
	}

	return nil
}

func validateObject(field Field, value any) error {
	if text, ok := value.(string); ok {
		normalized := text
		hasExpressions := expressionPlaceholderRegex.MatchString(text)
		if hasExpressions {
			normalized = expressionPlaceholderRegex.ReplaceAllString(text, "{}")
		}

		var parsed any
		if err := json.Unmarshal([]byte(normalized), &parsed); err != nil {
			return fmt.Errorf("must be valid JSON")
		}

		if field.TypeOptions != nil && field.TypeOptions.Object != nil && len(field.TypeOptions.Object.Schema) > 0 {
			obj, ok := parsed.(map[string]any)
			if !ok {
				return fmt.Errorf("must be an object")
			}
			if hasExpressions {
				return nil
			}
			return ValidateConfiguration(field.TypeOptions.Object.Schema, obj)
		}

		switch parsed.(type) {
		case map[string]any:
			return nil
		case []any:
			return nil
		default:
			return fmt.Errorf("must be an object or array")
		}
	}

	if field.TypeOptions != nil && field.TypeOptions.Object != nil && len(field.TypeOptions.Object.Schema) > 0 {
		obj, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("must be an object")
		}

		return ValidateConfiguration(field.TypeOptions.Object.Schema, obj)
	}

	switch value.(type) {
	case map[string]any:
		return nil
	case []any:
		return nil
	default:
		return fmt.Errorf("must be an object or array")
	}
}

func validateList(field Field, value any) error {
	list, ok := value.([]any)
	if !ok {
		return fmt.Errorf("must be a list of values")
	}

	if field.Required && len(list) == 0 {
		return fmt.Errorf("must contain at least one item")
	}

	if field.TypeOptions.List == nil {
		return nil
	}

	listOptions := field.TypeOptions.List

	if listOptions.MaxItems != nil {
		if *listOptions.MaxItems <= 0 {
			return fmt.Errorf("invalid max_items configuration: must be greater than 0")
		}
		if len(list) > *listOptions.MaxItems {
			return fmt.Errorf("must contain at most %d items", *listOptions.MaxItems)
		}
	}

	if listOptions.ItemDefinition == nil {
		return nil
	}

	itemDef := listOptions.ItemDefinition
	for i, item := range list {
		if itemDef.Type == FieldTypeObject && len(itemDef.Schema) > 0 {
			itemMap, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("item at index %d must be an object", i)
			}

			err := ValidateConfiguration(itemDef.Schema, itemMap)
			if err != nil {
				return fmt.Errorf("item at index %d: %w", i, err)
			}
		} else if itemDef.Type != "" {
			if item == nil {
				return fmt.Errorf("item at index %d cannot be empty", i)
			}

			err := validateFieldValue(Field{Type: itemDef.Type, Required: true}, item)
			if err != nil {
				return fmt.Errorf("item at index %d: %w", i, err)
			}
		}
	}

	return nil
}

func validateAnyPredicateList(field Field, value any) error {
	var predicates []Predicate
	err := mapstructure.Decode(value, &predicates)
	if err != nil {
		return fmt.Errorf("must be a list of predicates")
	}

	if field.Required && len(predicates) == 0 {
		return fmt.Errorf("must contain at least one predicate")
	}

	// Get valid operators if defined
	var validOperators []string
	if field.TypeOptions != nil && field.TypeOptions.AnyPredicateList != nil {
		for _, op := range field.TypeOptions.AnyPredicateList.Operators {
			validOperators = append(validOperators, op.Value)
		}
	}

	for i, predicate := range predicates {
		// Validate type field
		if predicate.Type == "" {
			return fmt.Errorf("predicate at index %d: 'type' must be a non-empty string", i)
		}

		// Validate operator is valid if operators are defined
		if len(validOperators) > 0 {
			valid := slices.Contains(validOperators, predicate.Type)
			if !valid {
				return fmt.Errorf("predicate at index %d: '%s' is not a valid operator. Must be one of: %s", i, predicate.Type, strings.Join(validOperators, ", "))
			}
		}

		// Validate value field
		if predicate.Value == "" {
			return fmt.Errorf("predicate at index %d: 'value' must be a non-empty string", i)
		}
	}

	return nil
}

func validateTime(field Field, value any) error {
	timeStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	// Default time format is HH:MM
	format := "15:04"
	if field.TypeOptions != nil && field.TypeOptions.Time != nil && field.TypeOptions.Time.Format != "" {
		format = field.TypeOptions.Time.Format
	}

	_, err := time.Parse(format, timeStr)
	if err != nil {
		return fmt.Errorf("must be a valid time in format %s", format)
	}

	return nil
}

func validateTimeRange(_ Field, value any) error {
	timeRangeStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if timeRangeStr == "" {
		return fmt.Errorf("time range cannot be empty")
	}

	startStr, endStr, err := parseTimeRange(timeRangeStr)
	if err != nil {
		return err
	}

	startTime, err := time.Parse("15:04", startStr)
	if err != nil {
		return fmt.Errorf("start time must be a valid time in format HH:MM")
	}

	endTime, err := time.Parse("15:04", endStr)
	if err != nil {
		return fmt.Errorf("end time must be a valid time in format HH:MM")
	}

	if !startTime.Before(endTime) {
		return fmt.Errorf("start time must be before end time")
	}

	return nil
}

func validateDate(field Field, value any) error {
	dateStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	// Default date format is YYYY-MM-DD
	format := "2006-01-02"
	if field.TypeOptions != nil && field.TypeOptions.Date != nil && field.TypeOptions.Date.Format != "" {
		format = field.TypeOptions.Date.Format
	}

	_, err := time.Parse(format, dateStr)
	if err != nil {
		return fmt.Errorf("must be a valid date in format %s", format)
	}

	return nil
}

func validateDateTime(field Field, value any) error {
	dateTimeStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	// Default datetime format is YYYY-MM-DDTHH:MM
	format := "2006-01-02T15:04"
	if field.TypeOptions != nil && field.TypeOptions.DateTime != nil && field.TypeOptions.DateTime.Format != "" {
		format = field.TypeOptions.DateTime.Format
	}

	_, err := time.Parse(format, dateTimeStr)
	if err != nil {
		return fmt.Errorf("must be a valid datetime in format %s", format)
	}

	return nil
}

func validateDayInYear(field Field, value any) error {
	dayStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	format := "MM/DD"
	if field.TypeOptions != nil && field.TypeOptions.DayInYear != nil && field.TypeOptions.DayInYear.Format != "" {
		format = field.TypeOptions.DayInYear.Format
	}

	var month, day int
	var extra string
	n, err := fmt.Sscanf(dayStr, "%d/%d%s", &month, &day, &extra)

	if n < 2 || n > 2 {
		return fmt.Errorf("must be a valid day in format %s (e.g., 12/25)", format)
	}

	if err != nil && n < 2 {
		return fmt.Errorf("must be a valid day in format %s (e.g., 12/25)", format)
	}

	if month < 1 || month > 12 || day < 1 || day > 31 {
		return fmt.Errorf("invalid day values: month must be 1-12, day must be 1-31")
	}

	daysInMonth := []int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if day > daysInMonth[month] {
		return fmt.Errorf("invalid day '%d' for month '%d'", day, month)
	}

	return nil
}

func validateCron(_ Field, value any) error {
	cronStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if cronStr == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}

	// Validate allowed wildcards: * , - /
	validChars := "0123456789*,-/ abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, char := range cronStr {
		if !strings.ContainsRune(validChars, char) {
			return fmt.Errorf("cron expression contains invalid character '%c'. Valid wildcards are: * , - /", char)
		}
	}

	fields := strings.Fields(cronStr)

	var parser cron.Parser
	var formatDescription string

	switch len(fields) {
	case 5:
		// Standard 5-field cron: minute, hour, day of month, month, day of week
		parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		formatDescription = "5-field format (minute hour day month dayofweek)"
	case 6:
		// Extended 6-field cron: second, minute, hour, day of month, month, day of week
		parser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		formatDescription = "6-field format (second minute hour day month dayofweek)"
	default:
		return fmt.Errorf("cron expression must have either 5 fields (minute hour day month dayofweek) or 6 fields (second minute hour day month dayofweek), got %d fields", len(fields))
	}

	_, err := parser.Parse(cronStr)
	if err != nil {
		return fmt.Errorf("invalid cron expression for %s: %w", formatDescription, err)
	}

	return nil
}

func validateFieldValue(field Field, value any) error {
	switch field.Type {
	case FieldTypeString:
		return validateString(field, value)

	case FieldTypeExpression:
		return validateExpression(field, value)

	case FieldTypeText:
		return validateText(field, value)

	case FieldTypeNumber:
		return validateNumber(field, value)

	case FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("must be a boolean")
		}

	case FieldTypeSelect:
		return validateSelect(field, value)

	case FieldTypeMultiSelect:
		return validateMultiSelect(field, value)

	case FieldTypeDaysOfWeek:
		return validateDaysOfWeek(field, value)

	case FieldTypeIntegrationResource:
		// If Multi is true, validate as array of strings
		// Otherwise, validate as a single string
		if field.TypeOptions != nil && field.TypeOptions.Resource != nil && field.TypeOptions.Resource.Multi {
			selectedValues, ok := value.([]any)
			if !ok {
				return fmt.Errorf("must be a list of values")
			}
			// Validate all items are strings
			for _, selectedValue := range selectedValues {
				if _, ok := selectedValue.(string); !ok {
					return fmt.Errorf("all items must be strings")
				}
			}
			return nil
		}
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeGitRef:
		// Git reference is represented as a string (e.g., refs/heads/main, refs/tags/v1.0.0)
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeUser:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeRole:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeGroup:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}

	case FieldTypeList:
		return validateList(field, value)

	case FieldTypeAnyPredicateList:
		return validateAnyPredicateList(field, value)

	case FieldTypeObject:
		return validateObject(field, value)

	case FieldTypeTime:
		return validateTime(field, value)

	case FieldTypeTimeRange:
		return validateTimeRange(field, value)

	case FieldTypeDate:
		return validateDate(field, value)

	case FieldTypeDateTime:
		return validateDateTime(field, value)

	case FieldTypeDayInYear:
		return validateDayInYear(field, value)

	case FieldTypeCron:
		return validateCron(field, value)

	case FieldTypeTimezone:
		return validateTimezone(field, value)
	}

	return nil
}

// isRequiredByCondition checks if a field should be required based on RequiredConditions
func isRequiredByCondition(field Field, config map[string]any) bool {
	for _, condition := range field.RequiredConditions {
		conditionValue, exists := config[condition.Field]
		if !exists {
			continue
		}

		conditionValueStr := fmt.Sprintf("%v", conditionValue)
		for _, requiredValue := range condition.Values {
			if conditionValueStr == requiredValue {
				return true
			}
		}
	}
	return false
}

// validateFieldRules validates comparison rules between fields
func validateFieldRules(field Field, value any, config map[string]any) error {
	for _, rule := range field.ValidationRules {
		compareValue, exists := config[rule.CompareWith]
		if !exists || compareValue == nil {
			continue // Skip validation if comparison field doesn't exist
		}

		err := validateComparisonRule(field, value, compareValue, rule)
		if err != nil {
			if rule.Message != "" {
				return fmt.Errorf("%s", rule.Message)
			}
			return err
		}
	}
	return nil
}

// validateComparisonRule validates a single comparison rule
func validateComparisonRule(field Field, value any, compareValue any, rule ValidationRule) error {
	switch field.Type {
	case FieldTypeTime:
		return validateTimeComparison(value, compareValue, rule)
	case FieldTypeDateTime:
		return validateDateTimeComparison(value, compareValue, rule)
	case FieldTypeDate:
		return validateDateComparison(value, compareValue, rule)
	case FieldTypeDayInYear:
		return validateDayInYearComparison(value, compareValue, rule)
	case FieldTypeNumber:
		return validateNumberComparison(value, compareValue, rule)
	default:
		return validateStringComparison(value, compareValue, rule)
	}
}

// validateTimeComparison validates time field comparisons
func validateTimeComparison(value any, compareValue any, rule ValidationRule) error {
	valueStr, ok1 := value.(string)
	compareStr, ok2 := compareValue.(string)
	if !ok1 || !ok2 {
		return fmt.Errorf("time values must be strings")
	}

	valueTime, err1 := time.Parse("15:04", valueStr)
	compareTime, err2 := time.Parse("15:04", compareStr)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("invalid time format")
	}

	return compareTimeValues(valueTime, compareTime, rule)
}

// validateDateTimeComparison validates datetime field comparisons
func validateDateTimeComparison(value any, compareValue any, rule ValidationRule) error {
	valueStr, ok1 := value.(string)
	compareStr, ok2 := compareValue.(string)
	if !ok1 || !ok2 {
		return fmt.Errorf("datetime values must be strings")
	}

	valueTime, err1 := time.Parse("2006-01-02T15:04", valueStr)
	compareTime, err2 := time.Parse("2006-01-02T15:04", compareStr)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("invalid datetime format")
	}

	return compareTimeValues(valueTime, compareTime, rule)
}

// validateDateComparison validates date field comparisons
func validateDateComparison(value any, compareValue any, rule ValidationRule) error {
	valueStr, ok1 := value.(string)
	compareStr, ok2 := compareValue.(string)
	if !ok1 || !ok2 {
		return fmt.Errorf("date values must be strings")
	}

	valueTime, err1 := time.Parse("2006-01-02", valueStr)
	compareTime, err2 := time.Parse("2006-01-02", compareStr)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("invalid date format")
	}

	return compareTimeValues(valueTime, compareTime, rule)
}

func validateDayInYearComparison(value any, compareValue any, rule ValidationRule) error {
	valueStr, ok1 := value.(string)
	compareStr, ok2 := compareValue.(string)
	if !ok1 || !ok2 {
		return fmt.Errorf("day-in-year values must be strings")
	}

	var valueMonth, valueDay int
	var compareMonth, compareDay int

	n1, err1 := fmt.Sscanf(valueStr, "%d/%d", &valueMonth, &valueDay)
	n2, err2 := fmt.Sscanf(compareStr, "%d/%d", &compareMonth, &compareDay)

	if n1 != 2 || n2 != 2 || err1 != nil || err2 != nil {
		return fmt.Errorf("invalid day-in-year format")
	}

	valueDayOfYear := (valueMonth-1)*31 + valueDay
	compareDayOfYear := (compareMonth-1)*31 + compareDay

	if valueMonth > compareMonth {
		if rule.Type == ValidationRuleLessThan {
			return nil
		}
	}

	return compareDayInYearValues(valueDayOfYear, compareDayOfYear, valueStr, compareStr, rule)
}

// validateNumberComparison validates number field comparisons
func validateNumberComparison(value any, compareValue any, rule ValidationRule) error {
	var num1, num2 float64

	switch v := value.(type) {
	case float64:
		num1 = v
	case int:
		num1 = float64(v)
	default:
		return fmt.Errorf("value must be a number")
	}

	switch v := compareValue.(type) {
	case float64:
		num2 = v
	case int:
		num2 = float64(v)
	default:
		return fmt.Errorf("comparison value must be a number")
	}

	return compareNumericValues(num1, num2, rule)
}

// validateStringComparison validates string field comparisons
func validateStringComparison(value any, compareValue any, rule ValidationRule) error {
	str1, ok1 := value.(string)
	str2, ok2 := compareValue.(string)
	if !ok1 || !ok2 {
		return fmt.Errorf("values must be strings")
	}

	return compareStringValues(str1, str2, rule)
}

// compareTimeValues compares time values based on the rule type
func compareTimeValues(value, compareValue time.Time, rule ValidationRule) error {
	switch rule.Type {
	case ValidationRuleLessThan:
		if !value.Before(compareValue) {
			return fmt.Errorf("must be before %v", compareValue.Format("15:04"))
		}
	case ValidationRuleGreaterThan:
		if !value.After(compareValue) {
			return fmt.Errorf("must be after %v", compareValue.Format("15:04"))
		}
	case ValidationRuleEqual:
		if !value.Equal(compareValue) {
			return fmt.Errorf("must be equal to %v", compareValue.Format("15:04"))
		}
	case ValidationRuleNotEqual:
		if value.Equal(compareValue) {
			return fmt.Errorf("must not be equal to %v", compareValue.Format("15:04"))
		}
	default:
		return fmt.Errorf("unknown validation rule type: %s", rule.Type)
	}
	return nil
}

// compareNumericValues compares numeric values based on the rule type
func compareNumericValues(value, compareValue float64, rule ValidationRule) error {
	switch rule.Type {
	case ValidationRuleLessThan:
		if value >= compareValue {
			return fmt.Errorf("must be less than %v", compareValue)
		}
	case ValidationRuleGreaterThan:
		if value <= compareValue {
			return fmt.Errorf("must be greater than %v", compareValue)
		}
	case ValidationRuleEqual:
		if value != compareValue {
			return fmt.Errorf("must be equal to %v", compareValue)
		}
	case ValidationRuleNotEqual:
		if value == compareValue {
			return fmt.Errorf("must not be equal to %v", compareValue)
		}
	default:
		return fmt.Errorf("unknown validation rule type: %s", rule.Type)
	}
	return nil
}

// compareStringValues compares string values based on the rule type
func compareStringValues(value, compareValue string, rule ValidationRule) error {
	switch rule.Type {
	case ValidationRuleLessThan:
		if value >= compareValue {
			return fmt.Errorf("must be less than %s", compareValue)
		}
	case ValidationRuleGreaterThan:
		if value <= compareValue {
			return fmt.Errorf("must be greater than %s", compareValue)
		}
	case ValidationRuleEqual:
		if value != compareValue {
			return fmt.Errorf("must be equal to %s", compareValue)
		}
	case ValidationRuleNotEqual:
		if value == compareValue {
			return fmt.Errorf("must not be equal to %s", compareValue)
		}
	default:
		return fmt.Errorf("unknown validation rule type: %s", rule.Type)
	}
	return nil
}

func compareDayInYearValues(valueDayOfYear, compareDayOfYear int, _, compareStr string, rule ValidationRule) error {
	switch rule.Type {
	case ValidationRuleLessThan:
		if valueDayOfYear >= compareDayOfYear {
			return fmt.Errorf("must be before %s", compareStr)
		}
	case ValidationRuleGreaterThan:
		if valueDayOfYear <= compareDayOfYear {
			return fmt.Errorf("must be after %s", compareStr)
		}
	case ValidationRuleEqual:
		if valueDayOfYear != compareDayOfYear {
			return fmt.Errorf("must be equal to %s", compareStr)
		}
	case ValidationRuleNotEqual:
		if valueDayOfYear == compareDayOfYear {
			return fmt.Errorf("must not be equal to %s", compareStr)
		}
	default:
		return fmt.Errorf("unknown validation rule type: %s", rule.Type)
	}
	return nil
}

func validateTimezone(_ Field, value any) error {
	timezoneStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if timezoneStr == "" {
		return fmt.Errorf("timezone cannot be empty")
	}

	if timezoneStr == "current" {
		return fmt.Errorf("timezone value 'current' should be replaced with actual timezone offset before submission")
	}

	var offsetHours float64
	var err error

	// Handle cases with or without + prefix
	cleanTz := strings.TrimPrefix(timezoneStr, "+")
	offsetHours, err = strconv.ParseFloat(cleanTz, 64)
	if err != nil {
		return fmt.Errorf("invalid timezone format: must be a numeric offset like '-5', '0', '5.5', or '+8'")
	}

	// Check valid range: UTC-12 to UTC+14
	if offsetHours < -12 || offsetHours > 14 {
		return fmt.Errorf("timezone offset must be between -12 and +14 hours, got: %g", offsetHours)
	}

	// Additional validation: only allow .5 decimal for half-hour timezones
	if offsetHours != float64(int(offsetHours)) && offsetHours != float64(int(offsetHours))+0.5 {
		return fmt.Errorf("timezone offset must be a whole number or half hour (e.g., 5.5), got: %g", offsetHours)
	}

	return nil
}

func parseTimeRange(timeRangeStr string) (string, string, error) {
	parts := strings.Split(timeRangeStr, "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("time range must be in format HH:MM-HH:MM")
	}

	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])
	if startStr == "" || endStr == "" {
		return "", "", fmt.Errorf("time range must be in format HH:MM-HH:MM")
	}

	return startStr, endStr, nil
}
