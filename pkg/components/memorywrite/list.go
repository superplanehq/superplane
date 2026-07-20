// Package memorywrite holds shared list-mode helpers used by the memory
// write components (addMemory, updateMemory, upsertMemory).
//
// When list mode is enabled, the component evaluates a list expression at
// execute time and writes one memory row per element, exposing each element
// as a configurable iteration variable for the per-item value expressions.
package memorywrite

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"
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
		"run":      {},
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
	}

	// Fall back to reflection so any typed slice/array (for example []int,
	// []float64, []MyStruct) is accepted, not just the explicit fast paths above.
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out, nil
	}

	return nil, fmt.Errorf("listSource must evaluate to a list, got %T", value)
}

// Variables builds the per-item set of extra expression variables (just the
// iteration variable for now; future enhancements can add index/totalCount).
func (m ListMode) Variables(item any) map[string]any {
	return map[string]any{m.ItemVariable: item}
}

// ResolveValue evaluates a single value field for a per-item iteration.
//
// String values are evaluated as bare expr expressions with the iteration
// variables merged in. Anything else (already resolved at Build time, for
// example via a {{ }} template) is returned unchanged so that the same
// resolved value is written for every list element.
func ResolveValue(value any, variables map[string]any, expressions core.ExpressionContext) (any, error) {
	raw, ok := value.(string)
	if !ok {
		return value, nil
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw, nil
	}

	resolved, err := expressions.RunWithExtraVariables(trimmed, variables)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

// NameValuePair is the common shape for configured name/value field lists.
type NameValuePair struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// FieldNames returns the trimmed, deduplicated field names from a list of
// pairs in declaration order. Pairs with empty names are skipped.
func FieldNames(pairs []NameValuePair) []string {
	fields := make([]string, 0, len(pairs))
	seen := map[string]struct{}{}
	for _, pair := range pairs {
		name := strings.TrimSpace(pair.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		fields = append(fields, name)
	}
	return fields
}

// ResolvePairs evaluates each pair's value with the provided variables and
// returns a map keyed by trimmed name. Pairs with empty names are skipped.
func ResolvePairs(pairs []NameValuePair, variables map[string]any, expressions core.ExpressionContext) (map[string]any, error) {
	values := make(map[string]any, len(pairs))
	for _, pair := range pairs {
		name := strings.TrimSpace(pair.Name)
		if name == "" {
			continue
		}
		resolved, err := ResolveValue(pair.Value, variables, expressions)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", name, err)
		}
		values[name] = resolved
	}
	return values, nil
}

// ResolveAllItemValues evaluates valueList for every list element before any
// memory writes run, so expression failures do not leave partial writes.
func ResolveAllItemValues(
	items []any,
	mode ListMode,
	pairs []NameValuePair,
	expressions core.ExpressionContext,
) ([]map[string]any, error) {
	resolved := make([]map[string]any, 0, len(items))
	for i, item := range items {
		values, err := ResolvePairs(pairs, mode.Variables(item), expressions)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve values for list item %d: %w", i, err)
		}
		resolved = append(resolved, values)
	}
	return resolved, nil
}

// ResolveAllItemMatches evaluates matchList for every list element using the
// same per-item iteration variable rules as ResolveAllItemValues, so bare
// expressions like item.uuid resolve to each iteration's actual value.
func ResolveAllItemMatches(
	items []any,
	mode ListMode,
	pairs []NameValuePair,
	expressions core.ExpressionContext,
) ([]map[string]any, error) {
	resolved := make([]map[string]any, 0, len(items))
	for i, item := range items {
		matches, err := ResolvePairs(pairs, mode.Variables(item), expressions)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve matches for list item %d: %w", i, err)
		}
		resolved = append(resolved, matches)
	}
	return resolved, nil
}

// AppendUniqueRecords appends records by physical memory ID. If a record is
// seen again, its latest values replace the previous entry without increasing
// the reported count.
func AppendUniqueRecords(
	records []core.CanvasMemoryRecord,
	positions map[uuid.UUID]int,
	next []core.CanvasMemoryRecord,
) []core.CanvasMemoryRecord {
	for _, record := range next {
		if record.ID == uuid.Nil {
			records = append(records, record)
			continue
		}

		if position, ok := positions[record.ID]; ok {
			records[position] = record
			continue
		}

		positions[record.ID] = len(records)
		records = append(records, record)
	}

	return records
}

// RecordValues returns the value payloads used by component outputs while
// keeping record IDs internal to the execution context.
func RecordValues(records []core.CanvasMemoryRecord) []any {
	values := make([]any, 0, len(records))
	for _, record := range records {
		values = append(values, record.Values)
	}
	return values
}
