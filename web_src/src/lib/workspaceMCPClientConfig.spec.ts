import { describe, expect, it } from "bun:test";

import { workspaceMCPClientConfig } from "./workspaceMCPClientConfig";

describe("workspaceMCPClientConfig", () => {
  it("builds a Cursor mcp.json snippet for the origin", () => {
    expect(workspaceMCPClientConfig("http://localhost:8000")).toBe(
      JSON.stringify(
        {
          mcpServers: {
            superplane: {
              url: "http://localhost:8000/mcp",
            },
          },
        },
        null,
        2,
      ),
    );
  });

  it("strips a trailing slash from the origin", () => {
    expect(workspaceMCPClientConfig("https://app.example.com/")).toContain("https://app.example.com/mcp");
  });
});
