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
  it("matches only the agent path segment", () => {
    expect(isAgentPath("/org/workspaces/rf/agent")).toBe(true);
    expect(isAgentPath("/org/workspaces/rf/settings/workspace/agent-resources")).toBe(false);
    expect(isAgentPath("/org/workspaces/rf/settings/workspace/mcp/resource-1")).toBe(false);
  });
});

describe("isWorkspaceAgentNavVisible", () => {
  const visible = {
    permissionsLoading: false,
    canUpdateFactories: true,
    workspaceMcpEnabled: true,
    workspaceSkillsEnabled: false,
  };

  it("shows Agent when the user can update factories and a workspace feature is on", () => {
    expect(isWorkspaceAgentNavVisible(visible)).toBe(true);
    expect(isWorkspaceAgentNavVisible({ ...visible, workspaceMcpEnabled: false, workspaceSkillsEnabled: true })).toBe(
      true,
    );
  });

  it("hides Agent while permissions load, without update permission, or when both features are off", () => {
    expect(isWorkspaceAgentNavVisible({ ...visible, permissionsLoading: true })).toBe(false);
    expect(isWorkspaceAgentNavVisible({ ...visible, canUpdateFactories: false })).toBe(false);
    expect(isWorkspaceAgentNavVisible({ ...visible, workspaceMcpEnabled: false })).toBe(false);
  });
});
