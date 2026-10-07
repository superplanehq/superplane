const MCP_PATH = "/mcp";

export type WorkspaceMCPClientTool = "cursor" | "claudeCode" | "vscode" | "codex" | "opencode";

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

/** Codex command that adds the SuperPlane workspace MCP server. */
export function workspaceMCPCodexCommand(origin: string): string {
  return `codex mcp add superplane --url ${workspaceMCPServerURL(origin)}`;
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

/** OpenCode config snippet for the SuperPlane workspace MCP URL. */
export function workspaceMCPOpenCodeConfig(origin: string): string {
  return JSON.stringify(
    {
      mcp: {
        superplane: {
          type: "remote",
          url: workspaceMCPServerURL(origin),
          enabled: true,
        },
      },
    },
    null,
    2,
  );
}

export function workspaceMCPClientSnippet(tool: WorkspaceMCPClientTool, origin: string): string {
  switch (tool) {
    case "claudeCode":
      return workspaceMCPClaudeCodeCommand(origin);
    case "codex":
      return workspaceMCPCodexCommand(origin);
    case "vscode":
      return workspaceMCPVSCodeConfig(origin);
    case "cursor":
      return workspaceMCPCursorConfig(origin);
    case "opencode":
      return workspaceMCPOpenCodeConfig(origin);
  }
}
