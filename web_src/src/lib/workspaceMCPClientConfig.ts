const MCP_PATH = "/mcp";

export type WorkspaceMCPClientTool = "cursor" | "claudeCode" | "vscode";

/** Streamable HTTP URL for the SuperPlane workspace MCP server. */
export function workspaceMCPServerURL(origin: string): string {
  return `${origin.replace(/\/+$/, "")}${MCP_PATH}`;
}

/** Cursor mcp.json snippet for the SuperPlane workspace MCP URL. */
export function workspaceMCPCursorConfig(origin: string): string {
  return JSON.stringify(
    {
      mcpServers: {
        superplane: {
          url: workspaceMCPServerURL(origin),
        },
      },
    },
    null,
    2,
  );
}

/** Claude Code command that adds the SuperPlane workspace MCP server. */
export function workspaceMCPClaudeCodeCommand(origin: string): string {
  return `claude mcp add --transport http superplane ${workspaceMCPServerURL(origin)}`;
}

/** VS Code mcp.json snippet for the SuperPlane workspace MCP URL. */
export function workspaceMCPVSCodeConfig(origin: string): string {
  return JSON.stringify(
    {
      servers: {
        superplane: {
          type: "http",
          url: workspaceMCPServerURL(origin),
        },
      },
    },
    null,
    2,
  );
}

export function workspaceMCPClientSnippet(tool: WorkspaceMCPClientTool, origin: string): string {
  if (tool === "claudeCode") {
    return workspaceMCPClaudeCodeCommand(origin);
  }
  if (tool === "vscode") {
    return workspaceMCPVSCodeConfig(origin);
  }
  return workspaceMCPCursorConfig(origin);
}
