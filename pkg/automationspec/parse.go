package automationspec

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Document struct {
	Definition
	root *yaml.Node
}

type Diagnostic struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s at %d:%d: %s", d.Path, d.Line, d.Column, d.Message)
}

// Diagnostic locates a public field, or its containing mapping if it is missing.
func (d *Document) Diagnostic(path, message string) *Diagnostic {
	if path == "$" {
		return at(path, d.root, message)
	}
	segments := strings.FieldsFunc(path, func(r rune) bool { return r == '.' || r == '[' || r == ']' })
	_, location := d.sourceAt(segments)
	return at(path, location.value, message)
}

func at(path string, node *yaml.Node, message string) *Diagnostic {
	location := &Diagnostic{Path: path, Line: 1, Column: 1, Message: message}
	if node != nil && node.Line > 0 && node.Column > 0 {
		location.Line = node.Line
		location.Column = node.Column
	}
	return location
}

func Parse(source string) (*Document, error) {
	decoder := yaml.NewDecoder(strings.NewReader(source))
	var parsed yaml.Node
	if err := decoder.Decode(&parsed); errors.Is(err, io.EOF) {
		return nil, at("$", nil, "Enter one automation definition.")
	} else if err != nil {
		return nil, syntaxDiagnostic(source, err)
	}

	var extra yaml.Node
	if err := decoder.Decode(&extra); err == nil {
		return nil, at("$", &extra, "Keep one YAML document in each automation definition.")
	} else if !errors.Is(err, io.EOF) {
		return nil, syntaxDiagnostic(source, err)
	}

	root := parsed.Content[0]
	document := &Document{root: root}
	value, err := document.readValue(nil, root)
	if err != nil {
		return nil, err
	}
	if err := document.validateSchema(value); err != nil {
		return nil, err
	}
	if err := root.Decode(&document.Definition); err != nil {
		return nil, syntaxDiagnostic(source, err)
	}
	if err := document.validateSemantics(); err != nil {
		return nil, err
	}

	if document.Icon == "" {
		document.Icon = DefaultIcon
	}
	if document.TimeoutSeconds == 0 {
		document.TimeoutSeconds = DefaultTimeoutSeconds
	}
	return document, nil
}

var yamlErrorLine = regexp.MustCompile(`line ([0-9]+): (.+)`)

func syntaxDiagnostic(source string, err error) *Diagnostic {
	diagnostic := &Diagnostic{Path: "$", Line: 1, Column: 1, Message: "Correct the YAML syntax. " + strings.TrimPrefix(err.Error(), "yaml: ")}
	if position := yamlErrorLine.FindStringSubmatch(err.Error()); position != nil {
		diagnostic.Line, _ = strconv.Atoi(position[1])
		diagnostic.Message = "Correct the YAML syntax. " + position[2]
		if lines := strings.Split(source, "\n"); diagnostic.Line > 0 && diagnostic.Line <= len(lines) {
			line := lines[diagnostic.Line-1]
			diagnostic.Column += len(line) - len(strings.TrimLeft(line, " \t"))
		}
	}
	return diagnostic
}

// readValue converts YAML to JSON-compatible values without resolving aliases.
func (d *Document) readValue(segments []string, node *yaml.Node) (any, error) {
	if node.Kind == yaml.AliasNode {
		path, _ := d.sourceAt(segments)
		return nil, at(path, node, "Enter the value directly. YAML aliases are unsupported.")
	}
	switch node.ShortTag() {
	case "!!map":
		if node.Kind == yaml.MappingNode {
			return d.readMapping(segments, node)
		}
	case "!!seq":
		if node.Kind == yaml.SequenceNode {
			return d.readList(segments, node)
		}
	case "!!str", "!!bool", "!!int", "!!float", "!!null":
		if node.Kind == yaml.ScalarNode {
			var value any
			if err := node.Decode(&value); err == nil {
				return value, nil
			}
		}
	}
	path, _ := d.sourceAt(segments)
	return nil, at(path, node, "Use a YAML value that can be represented in JSON.")
}

func (d *Document) readMapping(segments []string, node *yaml.Node) (map[string]any, error) {
	values := make(map[string]any, len(node.Content)/2)
	for index := 0; index < len(node.Content); index += 2 {
		key, value := node.Content[index], node.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			path, _ := d.sourceAt(segments)
			return nil, at(path, key, "Use a string for each field name.")
		}
		child := append(segments, key.Value)
		if _, exists := values[key.Value]; exists {
			path, _ := d.sourceAt(child)
			return nil, at(path, key, "Enter this field once.")
		}
		decoded, err := d.readValue(child, value)
		if err != nil {
			return nil, err
		}
		values[key.Value] = decoded
	}
	return values, nil
}

func (d *Document) readList(segments []string, node *yaml.Node) ([]any, error) {
	values := make([]any, len(node.Content))
	for index, value := range node.Content {
		decoded, err := d.readValue(append(segments, strconv.Itoa(index)), value)
		if err != nil {
			return nil, err
		}
		values[index] = decoded
	}
	return values, nil
}

type sourceLocation struct {
	key   *yaml.Node
	value *yaml.Node
}

// sourceAt maps JSON Schema location segments to YAML positions and public paths.
func (d *Document) sourceAt(segments []string) (string, sourceLocation) {
	var path strings.Builder
	location := sourceLocation{value: d.root}
	for _, segment := range segments {
		if location.value == nil {
			break
		}
		if location.value.Kind == yaml.SequenceNode {
			fmt.Fprintf(&path, "[%s]", segment)
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(location.value.Content) {
				break
			}
			location = sourceLocation{value: location.value.Content[index]}
			continue
		}

		if segment == "" || strings.ContainsAny(segment, ".$[]") {
			fmt.Fprintf(&path, "[%q]", segment)
		} else {
			if path.Len() > 0 {
				path.WriteByte('.')
			}
			path.WriteString(segment)
		}
		property := mappingProperty(location.value, segment)
		if property.value == nil {
			break
		}
		location = property
	}
	if path.Len() == 0 {
		return "$", location
	}
	return path.String(), location
}

func mappingProperty(node *yaml.Node, name string) sourceLocation {
	if node.Kind == yaml.MappingNode {
		for index := 0; index < len(node.Content); index += 2 {
			if key := node.Content[index]; key.Value == name {
				return sourceLocation{key: key, value: node.Content[index+1]}
			}
		}
	}
	return sourceLocation{}
}

func itemPath(path string, index int) string {
	return fmt.Sprintf("%s[%d]", path, index)
}
