import { describe, expect, it } from "bun:test";

import {
  workspaceMCPClaudeCodeCommand,
  workspaceMCPClientSnippet,
  workspaceMCPCursorConfig,
  workspaceMCPServerURL,
  workspaceMCPVSCodeConfig,
} from "./workspaceMCPClientConfig";

const ORIGIN = "http://localhost:8000";
const URL = "http://localhost:8000/mcp";

describe("workspaceMCPClientConfig", () => {
  it("builds the server URL and strips a trailing slash", () => {
    expect(workspaceMCPServerURL(ORIGIN)).toBe(URL);
    expect(workspaceMCPServerURL("https://app.example.com/")).toBe("https://app.example.com/mcp");
  });

  it("builds a Cursor mcp.json snippet", () => {
    expect(workspaceMCPCursorConfig(ORIGIN)).toBe(
      JSON.stringify({ mcpServers: { superplane: { url: URL } } }, null, 2),
    );
  });

  it("builds a Claude Code add command", () => {
    expect(workspaceMCPClaudeCodeCommand(ORIGIN)).toBe(`claude mcp add --transport http superplane ${URL}`);
  });

  it("builds a VS Code mcp.json snippet", () => {
    expect(workspaceMCPVSCodeConfig(ORIGIN)).toBe(
      JSON.stringify({ servers: { superplane: { type: "http", url: URL } } }, null, 2),
    );
  });

  it("selects the snippet for each tool", () => {
    expect(workspaceMCPClientSnippet("cursor", ORIGIN)).toBe(workspaceMCPCursorConfig(ORIGIN));
    expect(workspaceMCPClientSnippet("claudeCode", ORIGIN)).toBe(workspaceMCPClaudeCodeCommand(ORIGIN));
    expect(workspaceMCPClientSnippet("vscode", ORIGIN)).toBe(workspaceMCPVSCodeConfig(ORIGIN));
  });
});
