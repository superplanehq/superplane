package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// makeHandler returns an MCP tool handler that executes the mapped CLI command
// by shelling out to this same binary with `--output json`.
func makeHandler(self string, path []string) mcpsdk.ToolHandler {
	return func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args := map[string]any{}
		if raw := req.Params.Arguments; len(raw) > 0 {
			if err := json.Unmarshal(raw, &args); err != nil {
				return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
			}
		}

		cmdArgs := append([]string{}, path...)

		// Positional args come right after the command path.
		if pos, ok := args["args"]; ok {
			if list, ok := pos.([]any); ok {
				for _, item := range list {
					cmdArgs = append(cmdArgs, fmt.Sprint(item))
				}
			}
		}

		// Remaining keys map to flags.
		for k, v := range args {
			if k == "args" {
				continue
			}
			switch vv := v.(type) {
			case bool:
				if vv {
					cmdArgs = append(cmdArgs, "--"+k)
				}
			case []any:
				for _, item := range vv {
					cmdArgs = append(cmdArgs, "--"+k, fmt.Sprint(item))
				}
			default:
				cmdArgs = append(cmdArgs, "--"+k, fmt.Sprint(vv))
			}
		}

		// Always return structured output to the agent.
		cmdArgs = append(cmdArgs, "--output", "json")

		var stdout, stderr bytes.Buffer
		c := exec.CommandContext(ctx, self, cmdArgs...)
		c.Stdout = &stdout
		c.Stderr = &stderr

		if err := c.Run(); err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return errorResult(msg), nil
		}

		out := stdout.String()
		if strings.TrimSpace(out) == "" {
			out = "(command completed successfully with no output)"
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: out}},
		}, nil
	}
}

// errorResult reports a failed tool call back to the agent without failing the
// MCP request itself.
func errorResult(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
	}
}
