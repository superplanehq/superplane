package configurationfields

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"

	_ "github.com/superplanehq/superplane/pkg/server" // register all components, triggers, and integrations
)

type Issue struct {
	OwnerKind string
	OwnerName string
	Path      string
	Field     string
}

func (i Issue) String() string {
	return fmt.Sprintf(
		"%s %q field %q at %s must use camelCase (e.g. %q)",
		i.OwnerKind,
		i.OwnerName,
		i.Field,
		i.Path,
		suggestCamelCase(i.Field),
	)
}

func (i Issue) Key() string {
	return fmt.Sprintf("%s:%s:%s:%s", i.OwnerKind, i.OwnerName, i.Path, i.Field)
}

// Run scans registered component, trigger, widget, and integration configuration
// field definitions for snake_case names.
func Run() ([]Issue, error) {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		return nil, fmt.Errorf("create registry: %w", err)
	}

	var issues []Issue

	for _, action := range reg.ListActions() {
		issues = append(issues, checkOwner("action", action.Name(), "configuration", action.Configuration())...)
	}

	for _, trigger := range reg.ListTriggers() {
		issues = append(issues, checkOwner("trigger", trigger.Name(), "configuration", trigger.Configuration())...)
	}

	for _, widget := range reg.ListWidgets() {
		issues = append(issues, checkOwner("widget", widget.Name(), "configuration", widget.Configuration())...)
	}

	for _, integration := range reg.ListIntegrations() {
		issues = append(issues, checkOwner("integration", integration.Name(), "configuration", integration.Configuration())...)

		for _, action := range integration.Actions() {
			issues = append(issues, checkOwner("action", action.Name(), "configuration", action.Configuration())...)
		}

		for _, trigger := range integration.Triggers() {
			issues = append(issues, checkOwner("trigger", trigger.Name(), "configuration", trigger.Configuration())...)
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].OwnerKind != issues[j].OwnerKind {
			return issues[i].OwnerKind < issues[j].OwnerKind
		}
		if issues[i].OwnerName != issues[j].OwnerName {
			return issues[i].OwnerName < issues[j].OwnerName
		}
		if issues[i].Path != issues[j].Path {
			return issues[i].Path < issues[j].Path
		}
		return issues[i].Field < issues[j].Field
	})

	return issues, nil
}

func checkOwner(ownerKind, ownerName, pathPrefix string, fields []configuration.Field) []Issue {
	var issues []Issue
	checkFields(ownerKind, ownerName, pathPrefix, fields, &issues)
	return issues
}

func checkFields(ownerKind, ownerName, pathPrefix string, fields []configuration.Field, issues *[]Issue) {
	for i, field := range fields {
		fieldPath := fmt.Sprintf("%s[%d]", pathPrefix, i)
		checkFieldName(ownerKind, ownerName, fieldPath, field.Name, issues)

		for j, condition := range field.VisibilityConditions {
			checkFieldName(ownerKind, ownerName, fmt.Sprintf("%s.visibilityConditions[%d]", fieldPath, j), condition.Field, issues)
		}

		for j, condition := range field.RequiredConditions {
			checkFieldName(ownerKind, ownerName, fmt.Sprintf("%s.requiredConditions[%d]", fieldPath, j), condition.Field, issues)
		}

		if field.TypeOptions == nil {
			continue
		}

		if field.TypeOptions.Object != nil {
			checkFields(ownerKind, ownerName, fieldPath+".schema", field.TypeOptions.Object.Schema, issues)
		}

		if field.TypeOptions.List != nil && field.TypeOptions.List.ItemDefinition != nil {
			checkFields(ownerKind, ownerName, fieldPath+".itemDefinition.schema", field.TypeOptions.List.ItemDefinition.Schema, issues)
		}
	}
}

func checkFieldName(ownerKind, ownerName, path, name string, issues *[]Issue) {
	if name == "" || isCamelCaseFieldName(name) {
		return
	}

	*issues = append(*issues, Issue{
		OwnerKind: ownerKind,
		OwnerName: ownerName,
		Path:      path,
		Field:     name,
	})
}

func isCamelCaseFieldName(name string) bool {
	if len(name) == 0 || !unicode.IsLower(rune(name[0])) {
		return false
	}

	return !strings.Contains(name, "_")
}

func suggestCamelCase(snake string) string {
	parts := strings.Split(snake, "_")
	if len(parts) == 0 {
		return snake
	}

	var builder strings.Builder
	builder.WriteString(strings.ToLower(parts[0]))
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		runes := []rune(part)
		builder.WriteRune(unicode.ToUpper(runes[0]))
		if len(runes) > 1 {
			builder.WriteString(strings.ToLower(string(runes[1:])))
		}
	}

	return builder.String()
}
