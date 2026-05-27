package params

import (
	"fmt"
	"strconv"
	"strings"
)

const paramPrefix = "param("

// Type identifies a manual-run parameter field.
type Type string

const (
	TypeString  Type = "string"
	TypeNumber  Type = "number"
	TypeBoolean Type = "boolean"
	TypeSelect  Type = "select"
)

// Definition describes a param(...) placeholder in a template payload.
type Definition struct {
	Type     Type
	Title    string
	Default  any
	Required bool
	Values   []string // select only
}

// Field is a parameterized leaf in a template payload tree.
type Field struct {
	Path string
	Def  Definition
}

// ParseParamString parses s when it is a param(...) placeholder.
func ParseParamString(s string) (Definition, bool, error) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, paramPrefix) || !strings.HasSuffix(s, ")") {
		return Definition{}, false, nil
	}

	body := strings.TrimSpace(s[len(paramPrefix) : len(s)-1])
	opts, err := parseOptionPairs(body)
	if err != nil {
		return Definition{}, true, err
	}

	def := Definition{Required: false}
	typeSet := false

	for key, rawVal := range opts {
		switch key {
		case "type":
			t := Type(strings.TrimSpace(rawVal))
			switch t {
			case TypeString, TypeNumber, TypeBoolean, TypeSelect:
				def.Type = t
				typeSet = true
			default:
				return Definition{}, true, fmt.Errorf("unknown param type %q", rawVal)
			}
		case "title":
			def.Title = rawVal
		case "default":
			def.Default = rawVal
		case "required":
			required, err := strconv.ParseBool(strings.TrimSpace(rawVal))
			if err != nil {
				return Definition{}, true, fmt.Errorf("invalid required value %q", rawVal)
			}
			def.Required = required
		case "values":
			parts := strings.Split(rawVal, "|")
			def.Values = make([]string, 0, len(parts))
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					def.Values = append(def.Values, part)
				}
			}
		default:
			return Definition{}, true, fmt.Errorf("unknown param option %q", key)
		}
	}

	if !typeSet {
		return Definition{}, true, fmt.Errorf("param() missing type")
	}

	if def.Type == TypeSelect && len(def.Values) == 0 {
		return Definition{}, true, fmt.Errorf("select param requires values")
	}

	def.Default = coerceDefault(def.Type, def.Default)

	return def, true, nil
}

func coerceDefault(paramType Type, raw any) any {
	if raw == nil {
		return nil
	}
	s, ok := raw.(string)
	if !ok {
		return raw
	}
	switch paramType {
	case TypeNumber:
		if s == "" {
			return nil
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
		return s
	case TypeBoolean:
		if s == "" {
			return nil
		}
		b, err := strconv.ParseBool(s)
		if err != nil {
			return s
		}
		return b
	default:
		return s
	}
}

// HasParams reports whether payload contains any param(...) placeholders.
func HasParams(payload map[string]any) bool {
	return len(ExtractFields(payload)) > 0
}

// ExtractFields walks payload and returns parameterized fields in stable path order.
func ExtractFields(payload map[string]any) []Field {
	var fields []Field
	walkPayload(payload, "", &fields)
	return fields
}

func walkPayload(value any, prefix string, fields *[]Field) {
	switch v := value.(type) {
	case map[string]any:
		keys := sortedMapKeys(v)
		for _, key := range keys {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			walkPayload(v[key], path, fields)
		}
	case []any:
		for i, item := range v {
			path := fmt.Sprintf("%s[%d]", prefix, i)
			if prefix == "" {
				path = fmt.Sprintf("[%d]", i)
			}
			walkPayload(item, path, fields)
		}
	case string:
		def, ok, err := ParseParamString(v)
		if err != nil {
			return
		}
		if ok {
			*fields = append(*fields, Field{Path: prefix, Def: def})
		}
	}
}

// ValidatePayload checks that every param(...) placeholder in payload is well-formed.
func ValidatePayload(payload map[string]any) ([]Field, error) {
	var fields []Field
	if err := walkPayloadValidate(payload, "", &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func walkPayloadValidate(value any, prefix string, fields *[]Field) error {
	switch v := value.(type) {
	case map[string]any:
		keys := sortedMapKeys(v)
		for _, key := range keys {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			if err := walkPayloadValidate(v[key], path, fields); err != nil {
				return err
			}
		}
	case []any:
		for i, item := range v {
			path := fmt.Sprintf("%s[%d]", prefix, i)
			if prefix == "" {
				path = fmt.Sprintf("[%d]", i)
			}
			if err := walkPayloadValidate(item, path, fields); err != nil {
				return err
			}
		}
	case string:
		if !strings.HasPrefix(strings.TrimSpace(v), paramPrefix) {
			return nil
		}
		if prefix == "" {
			return fmt.Errorf("param() must be a string value on an object field")
		}
		def, ok, err := ParseParamString(v)
		if err != nil {
			return fmt.Errorf("field %q: %w", prefix, err)
		}
		if !ok {
			return fmt.Errorf("field %q: invalid param() syntax", prefix)
		}
		if _, err := validateDefinition(def); err != nil {
			return fmt.Errorf("field %q: %w", prefix, err)
		}
		*fields = append(*fields, Field{Path: prefix, Def: def})
	}
	return nil
}

func validateDefinition(def Definition) (Definition, error) {
	if def.Title == "" {
		return def, fmt.Errorf("param() missing title")
	}
	return def, nil
}

// ContainsUnresolvedParams reports whether any string value is still a param(...) placeholder.
func ContainsUnresolvedParams(payload map[string]any) bool {
	return hasParamString(payload)
}

func hasParamString(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for _, child := range v {
			if hasParamString(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if hasParamString(child) {
				return true
			}
		}
	case string:
		if strings.HasPrefix(strings.TrimSpace(v), paramPrefix) {
			return true
		}
	}
	return false
}

// Merge substitutes param placeholders in template with values keyed by dot-path.
func Merge(template map[string]any, values map[string]any) (map[string]any, error) {
	fields := ExtractFields(template)
	for _, field := range fields {
		if field.Def.Required {
			if _, ok := values[field.Path]; !ok {
				return nil, fmt.Errorf("missing required parameter %q", field.Path)
			}
		}
	}

	out, err := cloneValue(template)
	if err != nil {
		return nil, err
	}
	root, ok := out.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("template payload must be an object")
	}

	for _, field := range fields {
		raw, provided := values[field.Path]
		var value any
		if provided {
			value, err = coerceValue(field.Def, raw)
			if err != nil {
				return nil, fmt.Errorf("field %q: %w", field.Path, err)
			}
		} else if field.Def.Default != nil {
			value = field.Def.Default
		} else if field.Def.Required {
			return nil, fmt.Errorf("missing required parameter %q", field.Path)
		} else {
			continue
		}

		if err := setAtPath(root, field.Path, value); err != nil {
			return nil, err
		}
	}

	return root, nil
}

func coerceValue(def Definition, raw any) (any, error) {
	switch def.Type {
	case TypeString:
		switch v := raw.(type) {
		case string:
			return v, nil
		case float64, int, int64, bool:
			return fmt.Sprint(v), nil
		default:
			return nil, fmt.Errorf("expected string")
		}
	case TypeNumber:
		switch v := raw.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case string:
			if v == "" {
				return nil, fmt.Errorf("expected number")
			}
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return float64(i), nil
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("expected number")
			}
			return f, nil
		default:
			return nil, fmt.Errorf("expected number")
		}
	case TypeBoolean:
		switch v := raw.(type) {
		case bool:
			return v, nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("expected boolean")
			}
			return b, nil
		default:
			return nil, fmt.Errorf("expected boolean")
		}
	case TypeSelect:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string")
		}
		for _, allowed := range def.Values {
			if s == allowed {
				return s, nil
			}
		}
		return nil, fmt.Errorf("value %q is not one of the allowed options", s)
	default:
		return nil, fmt.Errorf("unknown param type")
	}
}

func parseOptionPairs(body string) (map[string]string, error) {
	parts := splitOutsideQuotes(body, ',')
	opts := make(map[string]string, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, err := splitKeyValue(part)
		if err != nil {
			return nil, err
		}
		opts[key] = unquote(val)
	}
	return opts, nil
}

func splitKeyValue(part string) (string, string, error) {
	inQuote := false
	for i := 0; i < len(part); i++ {
		if part[i] == '\'' {
			inQuote = !inQuote
			continue
		}
		if part[i] == ':' && !inQuote {
			key := strings.TrimSpace(part[:i])
			val := strings.TrimSpace(part[i+1:])
			if key == "" {
				return "", "", fmt.Errorf("invalid param option %q", part)
			}
			return key, val, nil
		}
	}
	return "", "", fmt.Errorf("invalid param option %q", part)
}

func splitOutsideQuotes(s string, sep byte) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' {
			inQuote = !inQuote
			current.WriteByte(c)
			continue
		}
		if c == sep && !inQuote {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(c)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func cloneValue(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, child := range t {
			cloned, err := cloneValue(child)
			if err != nil {
				return nil, err
			}
			out[k] = cloned
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, child := range t {
			cloned, err := cloneValue(child)
			if err != nil {
				return nil, err
			}
			out[i] = cloned
		}
		return out, nil
	default:
		return v, nil
	}
}

func setAtPath(root map[string]any, path string, value any) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	segments := strings.Split(path, ".")
	current := any(root)
	for i, segment := range segments {
		isLast := i == len(segments)-1
		switch node := current.(type) {
		case map[string]any:
			if isLast {
				node[segment] = value
				return nil
			}
			child, ok := node[segment]
			if !ok {
				return fmt.Errorf("path %q not found at segment %q", path, segment)
			}
			current = child
		default:
			return fmt.Errorf("path %q not found at segment %q", path, segment)
		}
	}
	return fmt.Errorf("path %q not found", path)
}
