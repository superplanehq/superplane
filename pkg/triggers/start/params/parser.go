package params

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// HasParams reports whether payload contains any param(...) leaf strings.
func HasParams(payload map[string]any) bool {
	return WalkPayload(payload, "", func(path string, value any) WalkControl {
		if s, ok := value.(string); ok && IsParamString(s) {
			return WalkStop
		}
		return WalkContinue
	}) == WalkStop
}

// ParseParams walks given payload and returns all param() definitions.
//
// Returns an error on the first invalid param() leaf, wrapped with its path:
// "<path>: <parse error>". Parse errors cover malformed param() syntax (missing type,
// unknown keys, bad quoting), invalid defaults, and select constraints such as
// missing values or a default outside the allowed options.
func ParseParams(payload map[string]any) ([]Definition, error) {
	var defs []Definition
	var err error

	WalkPayload(payload, "", func(path string, value any) WalkControl {
		if err != nil {
			return WalkStop
		}
		s, ok := value.(string)
		if !ok || !IsParamString(s) {
			// Not a param() expression, skip.
			return WalkContinue
		}
		def, parseErr := ParseParamString(path, s)
		if parseErr != nil {
			err = fmt.Errorf("%s: %w", path, parseErr)
			return WalkStop
		}
		defs = append(defs, def)
		return WalkContinue
	})

	if err == nil && len(defs) > 1 {
		sortDefinitions(defs)
	}
	return defs, err
}

var paramExprRe = regexp.MustCompile(`(?s)^param\((.*)\)$`)

// IsParamString reports whether s is a param(...) leaf value (trimmed).
func IsParamString(s string) bool {
	return paramExprRe.MatchString(strings.TrimSpace(s))
}

// ParseParamString parses a param(...) expression at path into a Definition.
func ParseParamString(path string, s string) (Definition, error) {
	s = strings.TrimSpace(s)
	m := paramExprRe.FindStringSubmatch(s)
	if m == nil {
		return Definition{}, fmt.Errorf("not a param() expression")
	}

	args, err := splitArgs(strings.TrimSpace(m[1]))
	if err != nil {
		return Definition{}, err
	}

	var (
		paramType    ParamType
		title        string
		defaultRaw   string
		defaultValue any
		required     bool
		values       []string
		order        int
	)
	for key, raw := range args {
		switch key {
		case "type":
			t, err := parseTypeName(raw)
			if err != nil {
				return Definition{}, err
			}
			paramType = t
		case "title":
			parsedTitle, err := parseQuotedString(raw)
			if err != nil {
				return Definition{}, fmt.Errorf("title: %w", err)
			}
			title = parsedTitle
		case "default":
			// default requires paramType, which may appear later than default in args map.
			// Postponing to after the loop.
			defaultRaw = raw
		case "required":
			parsedRequired, err := parseBoolToken(raw)
			if err != nil {
				return Definition{}, fmt.Errorf("required: %w", err)
			}
			required = parsedRequired
		case "values":
			parsedValues, err := parseSelectValues(raw)
			if err != nil {
				return Definition{}, fmt.Errorf("values: %w", err)
			}
			values = parsedValues
		case "order":
			parsedOrder, err := parseOrder(raw)
			if err != nil {
				return Definition{}, fmt.Errorf("order: %w", err)
			}
			order = parsedOrder
		default:
			return Definition{}, fmt.Errorf("unknown param() key %q", key)
		}
	}

	if defaultRaw != "" {
		defaultValue, err = parseDefaultValue(paramType, defaultRaw)
		if err != nil {
			return Definition{}, fmt.Errorf("default: %w", err)
		}
	}

	return NewDefinition(path, paramType, title, defaultValue, required, values, order)
}

func sortDefinitions(defs []Definition) {
	slices.SortFunc(defs, func(a, b Definition) int {
		if a.Order != b.Order {
			return cmp.Compare(a.Order, b.Order)
		}
		return strings.Compare(a.Path, b.Path)
	})
}

func splitArgs(inner string) (map[string]string, error) {
	if inner == "" {
		return nil, fmt.Errorf("param() has no arguments")
	}

	out := make(map[string]string)
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("param() has empty argument")
		}
		idx := strings.Index(part, ":")
		if idx < 0 {
			return nil, fmt.Errorf("invalid param() argument %q: missing ':'", part)
		}
		key := strings.TrimSpace(part[:idx])
		if key == "" {
			return nil, fmt.Errorf("invalid param() argument %q: empty key", part)
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("duplicate param() key %q", key)
		}
		out[key] = strings.TrimSpace(part[idx+1:])
	}
	return out, nil
}

func parseTypeName(raw string) (ParamType, error) {
	name := strings.TrimSpace(raw)
	switch ParamType(name) {
	case ParamTypeString, ParamTypeNumber, ParamTypeBoolean, ParamTypeSelect:
		return ParamType(name), nil
	default:
		return "", fmt.Errorf("unsupported type %q", name)
	}
}

func parseBoolToken(raw string) (bool, error) {
	switch strings.TrimSpace(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("expected true or false, got %q", raw)
	}
}

func parseQuotedString(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '\'' || raw[len(raw)-1] != '\'' {
		return "", fmt.Errorf("expected single-quoted string, got %q", raw)
	}
	content := raw[1 : len(raw)-1]
	if err := validateQuotedCharset(content); err != nil {
		return "", err
	}
	return content, nil
}

func validateQuotedCharset(s string) error {
	// Disallowing ', ", and comma in quoted strings for simpler parsing.
	if strings.ContainsAny(s, `'",`) {
		return fmt.Errorf("quoted value must not contain ', \", or comma")
	}
	return nil
}

func parseSelectValues(raw string) ([]string, error) {
	content, err := parseQuotedString(raw)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(content, "|")
	if len(parts) == 0 {
		return nil, fmt.Errorf("select values must not be empty")
	}

	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("select option must not be empty")
		}
		if strings.Contains(part, "|") {
			return nil, fmt.Errorf("select option must not contain |")
		}
		if err := validateQuotedCharset(part); err != nil {
			return nil, fmt.Errorf("select option %q: %w", part, err)
		}
		out = append(out, part)
	}
	return out, nil
}

func parseDefaultValue(paramType ParamType, raw string) (any, error) {
	raw = strings.TrimSpace(raw)
	switch paramType {
	case ParamTypeBoolean:
		return parseBoolToken(raw)
	case ParamTypeNumber:
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("expected number, got %q", raw)
		}
		return n, nil
	case ParamTypeString, ParamTypeSelect:
		return parseQuotedString(raw)
	default:
		return nil, fmt.Errorf("unsupported type %q", paramType)
	}
}

func parseOrder(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("expected non-negative integer, got %q", raw)
	}
	n, err := strconv.ParseInt(raw, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("expected non-negative integer, got %q", raw)
	}
	if n < 0 {
		return 0, fmt.Errorf("order must be non-negative, got %d", n)
	}
	return int(n), nil
}
