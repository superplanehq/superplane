import { describe, expect, it } from "bun:test";

import { initialsForName, isAgentPath, isWorkspaceAgentNavVisible } from "./factoriesRail";

describe("initialsForName", () => {
  it("falls back to '?' when no word has a letter or digit", () => {
    expect(initialsForName("")).toBe("?");
    expect(initialsForName("🚀")).toBe("?");
  });

  it("drops emoji-only words and keeps the first + last letter word", () => {
    expect(initialsForName("SuperPlane Prod 🚀")).toBe("SP");
  });
});

describe("isAgentPath", () => {
  it("matches only the workspace Agent page", () => {
    expect(isAgentPath("/org/workspaces/rf/agent")).toBe(true);
    expect(isAgentPath("/org/workspaces/rf/agent/mcp/resource-1")).toBe(true);
    expect(isAgentPath("/org/workspaces/rf/agent/skills/new")).toBe(true);
    expect(isAgentPath("/agent/workspaces/rf-p8n3wrcd/agent")).toBe(true);
    expect(isAgentPath("/org/workspaces/rf/settings/workspace/agent-resources")).toBe(false);
    expect(isAgentPath("/org/workspaces/rf/settings/workspace/mcp/resource-1")).toBe(false);
  });

  it("does not mark Agent current when the organization slug is agent", () => {
    expect(isAgentPath("/agent/workspaces/rf-p8n3wrcd/velocity")).toBe(false);
  });
});

describe("isWorkspaceAgentNavVisible", () => {
  const visible = {
    permissionsLoading: false,
    canUpdateFactories: true,
    workspaceAgentEnabled: true,
  };

  it("shows Agent when the user can update factories and SuperPlane Agent is on", () => {
    expect(isWorkspaceAgentNavVisible(visible)).toBe(true);
  });

  it("hides Agent while permissions load, without update permission, or when the flag is off", () => {
    expect(isWorkspaceAgentNavVisible({ ...visible, permissionsLoading: true })).toBe(false);
    expect(isWorkspaceAgentNavVisible({ ...visible, canUpdateFactories: false })).toBe(false);
    expect(isWorkspaceAgentNavVisible({ ...visible, workspaceAgentEnabled: false })).toBe(false);
  });
});
