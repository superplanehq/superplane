import { describe, expect, it } from "vitest";

import { columnAutomationHeaderRowCount, columnAutomationHeadline } from "./columnAutomationHeadline";
import type { ColumnAutomation } from "./columnAutomations";

function automation(overrides: Partial<ColumnAutomation>): ColumnAutomation {
  return {
    id: "automation-1",
    kind: "intake",
    name: "GitHub issues",
    trigger: "On GitHub issue",
    action: "Create a task",
    iconSrc: "",
    iconAlt: "",
    health: "healthy",
    runningCount: 0,
    catalogId: "github-issues",
    ...overrides,
  };
}

describe("columnAutomationHeadline", () => {
  it("names the intake source, not the custom intake name", () => {
    const intake = automation({ kind: "intake", name: "Refund bugs", catalogId: "github-issues" });
    expect(columnAutomationHeadline(intake)).toBe("Listens to GitHub issues");
  });

  it("falls back to the automation name for an unknown intake source", () => {
    const intake = automation({ kind: "intake", name: "Jira tickets", catalogId: "jira" });
    expect(columnAutomationHeadline(intake)).toBe("Listens to Jira tickets");
  });

  it("describes task analysis", () => {
    expect(columnAutomationHeadline(automation({ kind: "analysis", catalogId: "analysis" }))).toBe("Scores new tasks");
  });

  it("names the agent after the column step", () => {
    expect(columnAutomationHeadline(automation({ kind: "agent-step", name: "Implement" }))).toBe(
      "Runs the Implement agent",
    );
  });

  it("names a custom automation", () => {
    expect(columnAutomationHeadline(automation({ kind: "custom", name: "Notify QA" }))).toBe(
      "Runs the Notify QA automation",
    );
  });

  it("describes the pull request listeners", () => {
    expect(columnAutomationHeadline(automation({ kind: "pr-discussion" }))).toBe("Addresses pull request comments");
    expect(columnAutomationHeadline(automation({ kind: "pr-checks" }))).toBe("Fixes failing status checks");
    expect(columnAutomationHeadline(automation({ kind: "pr-closure" }))).toBe("Closes tasks when pull requests merge");
  });
});

describe("columnAutomationHeaderRowCount", () => {
  it("uses the largest column so every header keeps the same height", () => {
    const two = [automation({ id: "a" }), automation({ id: "b" })];
    const one = [automation({ id: "c" })];
    expect(columnAutomationHeaderRowCount([two, one, [], one])).toBe(2);
  });

  it("keeps one row when no column has an automation", () => {
    expect(columnAutomationHeaderRowCount([[], []])).toBe(1);
  });
});
