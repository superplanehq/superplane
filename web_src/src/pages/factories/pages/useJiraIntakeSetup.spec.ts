import { describe, expect, it } from "vitest";

import { readyJiraConnectionId } from "./useJiraIntakeSetup";

describe("readyJiraConnectionId", () => {
  it("reuses the selected ready connection", () => {
    expect(readyJiraConnectionId([{ metadata: { id: "jira-a" } }, { metadata: { id: "jira-b" } }], "jira-b")).toBe(
      "jira-b",
    );
  });

  it("falls back to the first ready connection", () => {
    expect(readyJiraConnectionId([{ metadata: { id: "jira-a" } }], "")).toBe("jira-a");
  });

  it("is empty when SuperPlane has no Jira connection", () => {
    expect(readyJiraConnectionId([], "")).toBe("");
  });
});
