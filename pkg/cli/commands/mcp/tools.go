package mcp

import (
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// toolSpec pairs a generated MCP tool with the CLI command path it maps to.
type toolSpec struct {
	tool *mcpsdk.Tool
	path []string
}

// commands we never expose as tools.
var skippedCommands = map[string]bool{
	"mcp":        true, // don't expose the server itself
	"help":       true,
	"completion": true,
}

// global/persistent flags we hide from every tool (managed by the server, not
// the agent).
var hiddenFlags = map[string]bool{
	"output":  true, // forced to json by the executor
	"verbose": true,
	"config":  true,
	"help":    true,
}

// collectTools walks the command tree and returns one tool per runnable leaf.
func collectTools(root *cobra.Command, readOnly bool) []toolSpec {
	var tools []toolSpec

	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		for _, sub := range c.Commands() {
			if sub.Hidden || skippedCommands[sub.Name()] {
				continue
			}

			childPath := append(append([]string{}, path...), sub.Name())

			if sub.Runnable() && (!readOnly || isReadOnly(sub.Name())) {
				tools = append(tools, buildTool(sub, childPath))
			}

			if sub.HasAvailableSubCommands() {
				walk(sub, childPath)
			}
		}
	}
	walk(root, nil)

	return tools
}

func isReadOnly(name string) bool {
	switch name {
	case "list", "get", "describe", "show", "tree", "whoami", "version", "active":
		return true
	}
	return strings.HasPrefix(name, "list")
}

func buildTool(cmd *cobra.Command, path []string) toolSpec {
	props := map[string]*jsonschema.Schema{}
	seen := map[string]bool{}

	addFlag := func(f *pflag.Flag) {
		if f.Hidden || seen[f.Name] || hiddenFlags[f.Name] {
			return
		}
		seen[f.Name] = true
		props[f.Name] = schemaForFlag(f)
	}
	cmd.LocalFlags().VisitAll(addFlag)
	cmd.InheritedFlags().VisitAll(addFlag)

	// Expose positional args when the command takes any.
	if strings.Contains(cmd.Use, "<") || strings.Contains(cmd.Use, "[") || cmd.Args != nil {
		props["args"] = &jsonschema.Schema{
			Type:        "array",
			Description: "Positional arguments for the command, in order (e.g. an ID or name).",
			Items:       &jsonschema.Schema{Type: "string"},
		}
	}

	desc := cmd.Short
	if cmd.Long != "" {
		desc = cmd.Long
	}

	return toolSpec{
		tool: &mcpsdk.Tool{
			Name:        toolName(path),
			Description: desc,
			InputSchema: &jsonschema.Schema{Type: "object", Properties: props},
		},
		path: path,
	}
}

// toolName joins the command path with underscores and sanitizes it to the
// characters MCP tool names allow.
func toolName(path []string) string {
	raw := strings.Join(path, "_")
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func schemaForFlag(f *pflag.Flag) *jsonschema.Schema {
	s := &jsonschema.Schema{Description: f.Usage}
	switch f.Value.Type() {
	case "bool":
		s.Type = "boolean"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "count":
		s.Type = "integer"
	case "float32", "float64":
		s.Type = "number"
	case "stringArray", "stringSlice", "intSlice", "boolSlice":
		s.Type = "array"
		s.Items = &jsonschema.Schema{Type: "string"}
	default:
		s.Type = "string"
	}
	return s
}
