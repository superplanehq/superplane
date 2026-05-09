# SuperPlane MCP Server

The SuperPlane MCP (Model Context Protocol) server provides AI assistants like Claude with direct access to SuperPlane workflows through a standardized tool interface.

## What is MCP?

[Model Context Protocol](https://modelcontextprotocol.io) is an open protocol that allows AI assistants to securely interact with external systems through a defined set of tools. The SuperPlane MCP server exposes SuperPlane's canvas, event, execution, integration, and secret management capabilities as MCP tools that Claude Code, Cursor, and other MCP clients can call.

## Why Use the MCP Server?

- **Direct API Access**: Claude can list canvases, emit events, check execution status, and manage workflows without needing to generate API code
- **Interactive Workflow Development**: Build and test canvases interactively with Claude's help
- **Debugging**: Query event history and execution details to troubleshoot workflows
- **Integration Management**: List and configure integrations and secrets programmatically

## Installation

The MCP server is built into the SuperPlane CLI. Make sure you have the latest version:

```bash
make build.cli
```

The `superplane mcp` command will be available in your `bin/` directory.

## Configuration

The MCP server requires authentication to access your SuperPlane instance. Configure it using one of these methods:

### Method 1: Environment Variables

```bash
export SUPERPLANE_API_TOKEN="your-api-token"
export SUPERPLANE_API_URL="https://your-instance.superplane.io"  # Optional, defaults to localhost:8000
```

### Method 2: SuperPlane Config File

If you've already run `superplane connect`, your credentials are stored in `~/.superplane.yaml`:

```yaml
currentContext: "https://your-instance.superplane.io/your-org"
contexts:
  - url: "https://your-instance.superplane.io"
    organization: "your-org"
    apiToken: "your-api-token"
```

The MCP server will automatically use your current context.

## Usage

### With Claude Code

Add this configuration to your `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "superplane": {
      "command": "/path/to/superplane",
      "args": ["mcp"],
      "env": {
        "SUPERPLANE_API_TOKEN": "your-api-token",
        "SUPERPLANE_API_URL": "https://your-instance.superplane.io"
      }
    }
  }
}
```

Or use your existing config file:

```json
{
  "mcpServers": {
    "superplane": {
      "command": "/path/to/superplane",
      "args": ["mcp"]
    }
  }
}
```

Restart Claude Code and verify the connection by asking: "List my SuperPlane canvases"

### With Cursor

Add to your Cursor settings (`.cursor/settings.json`):

```json
{
  "mcpServers": {
    "superplane": {
      "command": "/path/to/superplane",
      "args": ["mcp"],
      "env": {
        "SUPERPLANE_API_TOKEN": "your-api-token",
        "SUPERPLANE_API_URL": "https://your-instance.superplane.io"
      }
    }
  }
}
```

### Manual Testing

You can test the server manually using `stdio`:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./bin/superplane mcp
```

## Available Tools

### Canvas Tools

- **`list_canvases`**: List all canvases in the organization
  - Returns: Canvas ID, name, and creation timestamp
  
- **`describe_canvas`**: Get full details of a specific canvas
  - Input: `canvas_id` (string)
  - Returns: Canvas metadata and YAML spec

- **`create_canvas`**: Create a new canvas from YAML
  - Input: `name` (string), `spec_yaml` (string)
  - Returns: Created canvas ID and metadata

- **`update_canvas`**: Update an existing canvas
  - Input: `canvas_id` (string), `spec_yaml` (string)
  - Returns: Updated canvas metadata

### Event Tools

- **`emit_event`**: Emit an output event for a canvas node
  - Input: `canvas_id` (string), `node_id` (string), `data` (object)
  - Returns: Event ID
  - Note: This triggers execution flow for downstream nodes

- **`list_events`**: List recent root events for a canvas
  - Input: `canvas_id` (string)
  - Returns: Event ID, node ID, channel, custom name, timestamp

### Execution Tools

- **`list_executions`**: List workflow executions for a canvas node
  - Input: `canvas_id` (string), `node_id` (string)
  - Returns: Execution ID, state, result, timestamps

- **`describe_execution`**: Get detailed information about a specific execution
  - Input: `canvas_id` (string), `execution_id` (string)
  - Returns: Execution details with child executions

### Integration Tools

- **`list_integrations`**: List all configured integrations
  - Returns: Integration ID, service, name, configuration

### Secret Tools

- **`list_secrets`**: List all secrets in the organization
  - Returns: Secret ID, name, creation timestamp

## Development

### Running Tests

Run all MCP tests:

```bash
go test ./pkg/mcp/...
```

Run with verbose output:

```bash
go test -v ./pkg/mcp/...
```

Run integration tests only:

```bash
go test -v ./pkg/mcp -run Integration
```

### Adding New Tools

To add a new MCP tool:

1. Create or update the appropriate file in `pkg/mcp/tools/`
2. Implement the handler function that takes `context.Context`, `*openapi_client.APIClient`, and any arguments
3. Register the tool in the appropriate `Register*Tools` function
4. Add unit tests for the handler
5. Update the integration test to verify the tool is registered
6. Update this documentation

Example:

```go
func RegisterMyTools(ctx context.Context, s *mcp.Server, apiClient *openapi_client.APIClient) error {
    myToolHandler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var args struct {
            MyArg string `json:"my_arg"`
        }
        if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
            return nil, fmt.Errorf("failed to parse arguments: %w", err)
        }
        return handleMyTool(ctx, apiClient, args.MyArg)
    }

    s.AddTool(&mcp.Tool{
        Name:        "my_tool",
        Description: "Does something useful",
        InputSchema: json.RawMessage(`{"type":"object","properties":{"my_arg":{"type":"string","description":"Description of argument"}},"required":["my_arg"]}`),
    }, myToolHandler)

    return nil
}
```

### Architecture

The MCP server architecture:

```
cmd/mcp/main.go
    └─> pkg/mcp/server.go (StartServer)
          ├─> auth.go (LoadConfig, NewAPIClient)
          └─> registerTools()
                ├─> tools/canvases.go
                ├─> tools/canvas_mutations.go
                ├─> tools/events.go
                ├─> tools/executions.go
                ├─> tools/integrations.go
                └─> tools/secrets.go
```

The server uses:
- **Transport**: stdio (standard input/output)
- **Protocol**: JSON-RPC 2.0 over MCP
- **Authentication**: Bearer token in HTTP headers
- **API Client**: OpenAPI-generated client (`pkg/openapi_client`)

## Troubleshooting

### Server doesn't start

- Verify the `superplane` binary is in your PATH or use absolute path
- Check that your API token is valid: `superplane canvas list`
- Review the MCP server logs (stderr output from the command)

### Tools not appearing

- Restart your Claude Code or Cursor client
- Verify the MCP server configuration is in the correct settings file
- Check for error messages in the client's MCP logs

### API calls failing

- Verify `SUPERPLANE_API_URL` points to the correct instance
- Ensure your API token has the necessary permissions
- Test the API directly: `curl -H "Authorization: Bearer $SUPERPLANE_API_TOKEN" $SUPERPLANE_API_URL/api/v1/canvases`

## Security Notes

- **Never commit API tokens**: Use environment variables or secure config files
- **Token scope**: The MCP server has full access to your SuperPlane account with the provided token
- **Local only**: By default, the MCP server only runs locally and doesn't expose any network ports
- **Audit logs**: All API actions are logged in SuperPlane's audit trail

## See Also

- [Model Context Protocol Documentation](https://modelcontextprotocol.io)
- [SuperPlane API Documentation](https://docs.superplane.io)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
