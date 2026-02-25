package configuration

import "regexp"

var ExpressionPlaceholderRegex = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

func HasExpressionPlaceholder(value string) bool {
	return ExpressionPlaceholderRegex.MatchString(value)
}

func ReplaceExpressionPlaceholders(value string, replacement string) string {
	return ExpressionPlaceholderRegex.ReplaceAllString(value, replacement)
}
