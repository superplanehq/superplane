package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// CommandSpec is one shell directive in a command_list task.
// JSON accepts either a plain string or {"name","command"}.
type CommandSpec struct {
	Name    string `json:"name,omitempty"`
	Command string `json:"command"`
}

// DisplayText is the live-log accordion title (name when set, otherwise the shell).
func (c CommandSpec) DisplayText() string {
	if name := strings.TrimSpace(c.Name); name != "" {
		return name
	}
	return strings.TrimSpace(c.Command)
}

// ShellLine is the directive sourced/executed by the runner.
func (c CommandSpec) ShellLine() string {
	return strings.TrimSpace(c.Command)
}

// CommandList unmarshals create-task / stored commands that may be strings or objects.
type CommandList []CommandSpec

// NewCommandList builds unnamed command specs from shell lines (tests and adapters).
func NewCommandList(lines ...string) CommandList {
	out := make(CommandList, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, CommandSpec{Command: line})
	}
	return out
}

func (c CommandList) ShellLines() []string {
	out := make([]string, 0, len(c))
	for _, spec := range c {
		if line := spec.ShellLine(); line != "" {
			out = append(out, line)
		}
	}
	return out
}

func (c *CommandList) UnmarshalJSON(data []byte) error {
	data = bytesTrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*c = nil
		return nil
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return fmt.Errorf("commands must be an array: %w", err)
	}

	out := make(CommandList, 0, len(rawItems))
	for i, item := range rawItems {
		spec, err := parseCommandSpecJSON(item)
		if err != nil {
			return fmt.Errorf("commands[%d]: %w", i, err)
		}
		if spec.ShellLine() == "" {
			continue
		}
		out = append(out, spec)
	}
	*c = out
	return nil
}

func parseCommandSpecJSON(item json.RawMessage) (CommandSpec, error) {
	item = bytesTrimSpace(item)
	if len(item) == 0 || string(item) == "null" {
		return CommandSpec{}, fmt.Errorf("command entry is empty")
	}

	if item[0] == '"' {
		var line string
		if err := json.Unmarshal(item, &line); err != nil {
			return CommandSpec{}, err
		}
		return CommandSpec{Command: strings.TrimSpace(line)}, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(item, &raw); err != nil {
		return CommandSpec{}, err
	}
	if _, ok := raw["command"]; !ok {
		return CommandSpec{}, fmt.Errorf("command is required")
	}
	var spec CommandSpec
	if err := json.Unmarshal(item, &spec); err != nil {
		return CommandSpec{}, err
	}
	spec.Name = strings.TrimSpace(spec.Name)
	spec.Command = strings.TrimSpace(spec.Command)
	return spec, nil
}

func bytesTrimSpace(b []byte) []byte {
	return bytes.TrimSpace(b)
}
