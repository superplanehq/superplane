import { describe, expect, it } from "vitest";

import { BACKLOG_INTAKE_EMPTY_HINT, shouldShowBacklogIntakeEmptyHint } from "./backlogIntakeEmptyHint";

describe("shouldShowBacklogIntakeEmptyHint", () => {
  it("shows the hint when the backlog is empty and Intake is running", () => {
    expect(shouldShowBacklogIntakeEmptyHint({ empty: true, hasIntake: true, onboarding: false })).toBe(true);
  });

  it("hides the hint when the backlog has tasks", () => {
    expect(shouldShowBacklogIntakeEmptyHint({ empty: false, hasIntake: true, onboarding: false })).toBe(false);
  });

  it("hides the hint when Intake is not running", () => {
    expect(shouldShowBacklogIntakeEmptyHint({ empty: true, hasIntake: false, onboarding: false })).toBe(false);
  });

  it("hides the hint on the first-run board", () => {
    expect(shouldShowBacklogIntakeEmptyHint({ empty: true, hasIntake: true, onboarding: true })).toBe(false);
  });
});

describe("BACKLOG_INTAKE_EMPTY_HINT", () => {
  it("names Intake and the next actions in one short note", () => {
    const copy = `${BACKLOG_INTAKE_EMPTY_HINT.linkLabel}${BACKLOG_INTAKE_EMPTY_HINT.afterLink}`;
    expect(copy).toBe("Intake is running. Import an issue or create a task while you wait.");
    expect(copy.split(/\s+/).length).toBeLessThanOrEqual(25);
  });
});
