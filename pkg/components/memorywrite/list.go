// Package memorywrite holds shared list-mode helpers used by the memory
// write components (addMemory, updateMemory, upsertMemory).
//
// When list mode is enabled, the component evaluates a list expression at
// execute time and writes one memory row per element, exposing each element
// as a configurable iteration variable for the per-item value expressions.
package memorywrite

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const DefaultItemVariable = "item"

var (
	reservedItemVariables = map[string]struct{}{
		"$":        {},
		"memory":   {},
		"config":   {},
		"root":     {},
		"previous": {},
		"ctx":      {},
	}

	itemVariablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ListMode captures the optional list-mode configuration shared by every
// memory write component.
type ListMode struct {
	IterateList  bool   `mapstructure:"iterateList" json:"iterateList,omitempty"`
	ListSource   string `mapstructure:"listSource" json:"listSource,omitempty"`
	ItemVariable string `mapstructure:"itemVariable" json:"itemVariable,omitempty"`
}

// Normalize trims whitespace and applies the default iteration variable.
func (m ListMode) Normalize() ListMode {
	m.ListSource = strings.TrimSpace(m.ListSource)
	m.ItemVariable = strings.TrimSpace(m.ItemVariable)
	if m.IterateList && m.ItemVariable == "" {
		m.ItemVariable = DefaultItemVariable
	}
	return m
}

// Validate ensures list-mode configuration is internally consistent.
// It only validates when IterateList is true.
func (m ListMode) Validate() error {
	if !m.IterateList {
		return nil
	}
	if m.ListSource == "" {
		return fmt.Errorf("listSource is required when iterateList is true")
	}
	if !itemVariablePattern.MatchString(m.ItemVariable) {
		return fmt.Errorf("itemVariable %q must match %s", m.ItemVariable, itemVariablePattern.String())
	}
	if _, ok := reservedItemVariables[m.ItemVariable]; ok {
		return fmt.Errorf("itemVariable %q is reserved", m.ItemVariable)
	}
	return nil
}

// EvaluateList resolves the configured list expression to a slice of items.
// It accepts []any directly or coerces common slice/array shapes.
func (m ListMode) EvaluateList(expressions core.ExpressionContext) ([]any, error) {
	if !m.IterateList {
		return nil, fmt.Errorf("list mode is not enabled")
	}
	if expressions == nil {
		return nil, fmt.Errorf("expression context is not available")
	}

	value, err := expressions.Run(m.ListSource)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate listSource: %w", err)
	}

	return coerceToList(value)
}

func coerceToList(value any) ([]any, error) {
	switch v := value.(type) {
	case nil:
		return []any{}, nil
	case []any:
		return v, nil
	case []map[string]any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	case []string:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = item
		}
		return out, nil
	default:
		return nil, fmt.Errorf("listSource must evaluate to a list, got %T", value)
	}
}

// Scope builds the per-item expression scope (just the iteration variable
// for now; future enhancements can add index/totalCount).
func (m ListMode) Scope(item any) map[string]any {
	return map[string]any{m.ItemVariable: item}
}

// ResolveValue evaluates a single value field for a per-item iteration.
//
// String values are evaluated as bare expr expressions with the iteration
// scope merged in. Anything else (already resolved at Build time, for
// example via a {{ }} template) is returned unchanged so that the same
// resolved value is written for every list element.
func ResolveValue(value any, scope map[string]any, expressions core.ExpressionContext) (any, error) {
	raw, ok := value.(string)
	if !ok {
		return value, nil
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, nil
	}

	resolved, err := expressions.RunWithScope(trimmed, scope)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}
