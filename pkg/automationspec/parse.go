package automationspec

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Document struct {
	Definition
	locations map[string]*yaml.Node
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
	for candidate := path; candidate != ""; {
		if node := d.locations[candidate]; node != nil {
			return at(path, node, message)
		}
		end := strings.LastIndexAny(candidate, ".[")
		if end < 0 {
			break
		}
		candidate = candidate[:end]
	}
	return at(path, d.locations["$"], message)
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
	document := &Document{locations: make(map[string]*yaml.Node)}
	if err := document.validateShape("$", root, objectField(definitionFields)); err != nil {
		return nil, err
	}
	if err := root.Decode(&document.Definition); err != nil {
		return nil, syntaxDiagnostic(source, err)
	}
	if err := document.validateDefinition(); err != nil {
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

type fieldShape struct {
	types  []string
	entry  *fieldShape
	fields map[string]fieldShape
}

func scalarField(types ...string) fieldShape {
	return fieldShape{types: types}
}

func objectField(fields map[string]fieldShape) fieldShape {
	return fieldShape{fields: fields}
}

func listField(entry fieldShape) fieldShape {
	return fieldShape{entry: &entry}
}

var (
	textField        = scalarField("!!str")
	definitionFields = map[string]fieldShape{
		"apiVersion":      textField,
		"kind":            textField,
		"name":            textField,
		"icon":            textField,
		"timeout_seconds": scalarField("!!int"),
		"agent": objectField(map[string]fieldShape{
			"harness": textField,
			"model":   textField,
			"credentials": objectField(map[string]fieldShape{
				"source": textField,
				"name":   textField,
			}),
		}),
		"triggers": listField(objectField(map[string]fieldShape{
			"id":  textField,
			"use": textField,
			"config": objectField(map[string]fieldShape{
				"cron":       textField,
				"utc_offset": scalarField("!!int", "!!float"),
			}),
		})),
		"steps": listField(objectField(map[string]fieldShape{
			"name":              textField,
			"bash":              textField,
			"prompt":            textField,
			"working_directory": textField,
			"secrets":           listField(textField),
		})),
	}
)

func (d *Document) validateShape(path string, node *yaml.Node, shape fieldShape) error {
	d.locations[path] = node
	switch {
	case node.ShortTag() == "!!null":
		return at(path, node, "Replace null with a value.")
	case node.Kind == yaml.AliasNode:
		return at(path, node, "Enter the value directly. YAML aliases are unsupported.")
	case shape.fields != nil:
		return d.validateMapping(path, node, shape.fields)
	case shape.entry != nil:
		return d.validateList(path, node, *shape.entry)
	default:
		return validateScalar(path, node, shape.types)
	}
}

func (d *Document) validateMapping(path string, node *yaml.Node, fields map[string]fieldShape) error {
	if node.Kind != yaml.MappingNode || node.ShortTag() != "!!map" {
		return at(path, node, "Enter a mapping with named fields.")
	}
	for index := 0; index < len(node.Content); index += 2 {
		key, value := node.Content[index], node.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" {
			return at(path, key, "Use a string for each field name.")
		}
		childPath := key.Value
		if path != "$" {
			childPath = path + "." + key.Value
		}
		if _, exists := d.locations[childPath]; exists {
			return at(childPath, key, "Enter this field once.")
		}
		field, exists := fields[key.Value]
		if !exists {
			return at(childPath, key, "Remove this unknown field or correct its name.")
		}
		if err := d.validateShape(childPath, value, field); err != nil {
			return err
		}
	}
	return nil
}

func (d *Document) validateList(path string, node *yaml.Node, entry fieldShape) error {
	if node.Kind != yaml.SequenceNode || node.ShortTag() != "!!seq" {
		return at(path, node, "Enter a list.")
	}
	for index, value := range node.Content {
		if err := d.validateShape(itemPath(path, index), value, entry); err != nil {
			return err
		}
	}
	return nil
}

func validateScalar(path string, node *yaml.Node, tags []string) error {
	matches := node.Kind == yaml.ScalarNode && slices.Contains(tags, node.ShortTag())
	switch {
	case len(tags) > 1:
		var number float64
		if matches && node.Decode(&number) == nil {
			return nil
		}
		return at(path, node, "Enter a number.")
	case tags[0] == "!!int":
		var number int
		if matches && node.Decode(&number) == nil {
			return nil
		}
		return at(path, node, "Enter an integer.")
	case !matches:
		return at(path, node, "Enter a string.")
	default:
		return nil
	}
}

func itemPath(path string, index int) string {
	return fmt.Sprintf("%s[%d]", path, index)
}
