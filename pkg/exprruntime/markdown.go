package exprruntime

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/expr-lang/expr"
)

// MarkdownFunctionOptions adds escaping functions for text and angle-bracket link destinations.
func MarkdownFunctionOptions() []expr.Option {
	return []expr.Option{
		markdownFunction("markdownText", markdownText),
		markdownFunction("markdownLink", markdownLink),
	}
}

func markdownFunction(name string, escape func(string) string) expr.Option {
	return expr.Function(name, func(params ...any) (any, error) {
		if len(params) != 1 {
			return nil, fmt.Errorf("%s() expects one string argument", name)
		}
		value, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("%s() expects a string", name)
		}
		return escape(value), nil
	}, new(func(string) string))
}

func markdownText(value string) string {
	var escaped strings.Builder
	for _, r := range value {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			escaped.WriteByte(' ')
			continue
		}
		if strings.ContainsRune("&<>|\\`*_{}[]()!#~", r) {
			fmt.Fprintf(&escaped, "&#%d;", r)
			continue
		}
		escaped.WriteRune(r)
	}
	return escaped.String()
}

func markdownLink(value string) string {
	return strings.NewReplacer("<", "%3C", ">", "%3E", "\n", "%0A", "\r", "%0D", " ", "%20").Replace(value)
}
