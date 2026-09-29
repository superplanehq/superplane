const MCP_PATH = "/mcp";

/** Cursor mcp.json snippet for the SuperPlane workspace MCP URL. */
export function workspaceMCPClientConfig(origin: string): string {
  const base = origin.replace(/\/+$/, "");
  return JSON.stringify(
    {
      mcpServers: {
        superplane: {
          url: `${base}${MCP_PATH}`,
        },
      },
    },
    null,
    2,
  );
}
