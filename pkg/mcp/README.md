# SuperPlane MCP Server

This package implements a Model Context Protocol (MCP) server for SuperPlane as an HTTP handler using JSON-RPC 2.0 over the Streamable HTTP transport.

## Overview

The MCP server provides programmatic access to SuperPlane resources through three tools:
- **canvas_get**: Retrieve a canvas (workflow) in YAML or JSON format
- **canvas_list_versions**: List all versions of a canvas
- **integrations_list**: List all integrations for an organization

## Architecture

- **handler.go**: Main HTTP handler implementing JSON-RPC 2.0 protocol
- **tools.go**: Implementation of the three MCP tools
- **handler_test.go**: Comprehensive test suite

## Endpoint

```
POST /mcp
```

## Authentication

The MCP server uses JWT Bearer token authentication via the `Authorization` header:

```
Authorization: Bearer <jwt-token>
```

The JWT token is validated using the same signer as the rest of the SuperPlane API. Claims from the token (org_id, user_id) are extracted and used for authorization.

## JSON-RPC 2.0 Protocol

### Request Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "canvas_get",
    "arguments": {
      "canvas_id": "uuid",
      "org_id": "uuid",
      "format": "yaml"
    }
  }
}
```

### Response Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "...",
        "mimeType": "application/x-yaml"
      }
    ]
  }
}
```

### Error Format

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32600,
    "message": "Invalid Request",
    "data": null
  }
}
```

## Supported Methods

### initialize

Handshake to establish protocol version and capabilities.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "superplane-mcp",
      "version": "1.0.0"
    }
  }
}
```

### notifications/initialized

Client notification that initialization is complete. No response is sent.

### tools/list

List all available tools.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "canvas_get",
        "description": "Retrieve a canvas (workflow) by ID in YAML or JSON format",
        "inputSchema": {
          "type": "object",
          "properties": {
            "canvas_id": {
              "type": "string",
              "description": "The ID of the canvas to retrieve"
            },
            "org_id": {
              "type": "string",
              "description": "The organization ID that owns the canvas"
            },
            "format": {
              "type": "string",
              "description": "Output format: 'yaml' or 'json'",
              "enum": ["yaml", "json"],
              "default": "yaml"
            }
          },
          "required": ["canvas_id", "org_id"]
        }
      }
      // ... other tools
    ]
  }
}
```

### tools/call

Execute a specific tool.

## Tools

### canvas_get

Retrieves a canvas with its live version.

**Arguments:**
- `canvas_id` (string, required): UUID of the canvas
- `org_id` (string, required): UUID of the organization
- `format` (string, optional): Output format - "yaml" (default) or "json"

**Example:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "canvas_get",
    "arguments": {
      "canvas_id": "123e4567-e89b-12d3-a456-426614174000",
      "org_id": "123e4567-e89b-12d3-a456-426614174001",
      "format": "yaml"
    }
  }
}
```

### canvas_list_versions

Lists all versions of a canvas.

**Arguments:**
- `canvas_id` (string, required): UUID of the canvas
- `org_id` (string, required): UUID of the organization

**Example:**
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "canvas_list_versions",
    "arguments": {
      "canvas_id": "123e4567-e89b-12d3-a456-426614174000",
      "org_id": "123e4567-e89b-12d3-a456-426614174001"
    }
  }
}
```

### integrations_list

Lists all integrations for an organization.

**Arguments:**
- `org_id` (string, required): UUID of the organization

**Example:**
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "integrations_list",
    "arguments": {
      "org_id": "123e4567-e89b-12d3-a456-426614174001"
    }
  }
}
```

## Error Codes

The server uses standard JSON-RPC 2.0 error codes:

- `-32700`: Parse error - Invalid JSON
- `-32600`: Invalid Request - Missing required fields or invalid format
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error

## Testing

Run the test suite:

```bash
go test ./pkg/mcp/... -v
```

## Integration

The MCP handler is registered in `pkg/public/server.go` at the `/mcp` endpoint. It validates JWT tokens but does not use the standard middleware stack, as authentication is handled internally.

## Future Enhancements

Potential improvements:
- Add more tools (canvas create, update, delete)
- Support for canvas nodes and edges queries
- Webhook and trigger management tools
- Real-time updates via streaming
