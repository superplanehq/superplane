package canvasyaml

import (
	"fmt"
	"strings"
	"unicode"

	goyaml "gopkg.in/yaml.v3"
)

type Issue struct {
	Line     int
	Path     string
	Field    string
	NodeID   string
	NodeName string
}

func (i Issue) String() string {
	suggestion := suggestCamelCase(i.Field)
	location := i.Path
	if i.Line > 0 {
		location = fmt.Sprintf("%s:%d", i.Path, i.Line)
	}

	nodeRef := ""
	if i.NodeID != "" || i.NodeName != "" {
		nodeRef = fmt.Sprintf(" (node id=%q name=%q)", i.NodeID, i.NodeName)
	}

	return fmt.Sprintf(
		"%s: configuration field %q uses snake_case%s; use camelCase (e.g. %q)",
		location,
		i.Field,
		nodeRef,
		suggestion,
	)
}

// LintConfigurationFieldNames reports snake_case keys under spec.nodes[].configuration.
func LintConfigurationFieldNames(data []byte) ([]Issue, error) {
	var doc goyaml.Node
	if err := goyaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse canvas yaml: %w", err)
	}

	root := &doc
	if root.Kind == goyaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != goyaml.MappingNode {
		return nil, nil
	}

	specNode := mappingValue(root, "spec")
	if specNode == nil {
		return nil, nil
	}

	nodesNode := mappingValue(specNode, "nodes")
	if nodesNode == nil || nodesNode.Kind != goyaml.SequenceNode {
		return nil, nil
	}

	var issues []Issue
	for i, nodeNode := range nodesNode.Content {
		if nodeNode.Kind != goyaml.MappingNode {
			continue
		}

		nodeID := scalarMappingValue(nodeNode, "id")
		nodeName := scalarMappingValue(nodeNode, "name")
		configNode := mappingValue(nodeNode, "configuration")
		if configNode == nil {
			continue
		}

		pathPrefix := fmt.Sprintf("spec.nodes[%d].configuration", i)
		if nodeID != "" {
			pathPrefix = fmt.Sprintf("spec.nodes[%d] (id=%q).configuration", i, nodeID)
		}

		walkConfigurationKeys(configNode, pathPrefix, nodeID, nodeName, &issues)
	}

	return issues, nil
}

// FormatIssues returns a multi-line error summary for CLI/API responses.
func FormatIssues(issues []Issue) string {
	if len(issues) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("canvas configuration fields must use camelCase:\n")
	for _, issue := range issues {
		builder.WriteString("  - ")
		builder.WriteString(issue.String())
		builder.WriteByte('\n')
	}

	return strings.TrimRight(builder.String(), "\n")
}

func walkConfigurationKeys(node *goyaml.Node, path, nodeID, nodeName string, issues *[]Issue) {
	switch node.Kind {
	case goyaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			key := keyNode.Value
			childPath := path + "." + key

			if isSnakeCaseFieldName(key) {
				*issues = append(*issues, Issue{
					Line:     keyNode.Line,
					Path:     childPath,
					Field:    key,
					NodeID:   nodeID,
					NodeName: nodeName,
				})
			}

			walkConfigurationKeys(valueNode, childPath, nodeID, nodeName, issues)
		}
	case goyaml.SequenceNode:
		for i, item := range node.Content {
			walkConfigurationKeys(item, fmt.Sprintf("%s[%d]", path, i), nodeID, nodeName, issues)
		}
	}
}

func isSnakeCaseFieldName(name string) bool {
	if !strings.Contains(name, "_") {
		return false
	}

	for _, r := range name {
		if r == '_' {
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}

	// Allow SCREAMING_SNAKE constants if they ever appear as keys.
	if strings.ToUpper(name) == name {
		return false
	}

	return true
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

func mappingValue(mapping *goyaml.Node, key string) *goyaml.Node {
	if mapping == nil || mapping.Kind != goyaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		if keyNode.Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

func scalarMappingValue(mapping *goyaml.Node, key string) string {
	valueNode := mappingValue(mapping, key)
	if valueNode == nil || valueNode.Kind != goyaml.ScalarNode {
		return ""
	}

	return valueNode.Value
}
