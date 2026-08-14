import { describe, expect, it } from "vitest";

import {
  STORYBOOK_WORKSPACE_CONNECTIONS,
  agentSummary,
  appRepositorySummary,
  backlogSummary,
} from "./workspaceConnections";

describe("workspaceConnections summaries", () => {
  it("describes the seeded post-onboarding app repository", () => {
    expect(appRepositorySummary(STORYBOOK_WORKSPACE_CONNECTIONS)).toBe("GitHub · acme/api");
  });

  it("describes GitHub Issues on the backlog repository", () => {
    expect(backlogSummary(STORYBOOK_WORKSPACE_CONNECTIONS)).toBe("GitHub Issues · acme/api");
  });

  it("describes a skipped backlog", () => {
    expect(backlogSummary({ ...STORYBOOK_WORKSPACE_CONNECTIONS, issuesChoice: "skip", issuesRepo: null })).toBe(
      "No backlog. Create each work order yourself.",
    );
  });

  it("describes the selected coding agent", () => {
    expect(agentSummary(STORYBOOK_WORKSPACE_CONNECTIONS)).toBe("Claude Code");
  });
});
