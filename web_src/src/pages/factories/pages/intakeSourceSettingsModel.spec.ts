import { describe, expect, it } from "vitest";

import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  GITHUB_INTAKE_RUNS,
  isIntakeSettingsTab,
  intakePlacementActivity,
  intakePlacementLabel,
  intakeRelativeTime,
  intakeSettingsFromApi,
  intakeSettingsToApi,
  normalizeIntakeSourceSettings,
  toggleIntakeLabel,
} from "./intakeSourceSettingsModel";

describe("intakeSourceSettingsModel", () => {
  it("adds and removes a label", () => {
    expect(toggleIntakeLabel([], "bug")).toEqual(["bug"]);
    expect(toggleIntakeLabel(["bug", "enhancement"], "bug")).toEqual(["enhancement"]);
  });

  it("keeps a default name when the draft name is empty", () => {
    const next = normalizeIntakeSourceSettings({
      ...DEFAULT_GITHUB_INTAKE_SETTINGS,
      name: "   ",
      confidencePct: 140.6,
    });

    expect(next.name).toBe("GitHub issues");
    expect(next.confidencePct).toBe(100);
  });

  it("labels ticket placement for backlog, rejected, and in-progress work", () => {
    const implement = GITHUB_INTAKE_RUNS.find((run) => run.id === "gh-issue-1")!;
    const backlog = GITHUB_INTAKE_RUNS.find((run) => run.id === "gh-issue-3")!;
    const rejected = GITHUB_INTAKE_RUNS.find((run) => run.id === "gh-issue-4")!;
    const held = GITHUB_INTAKE_RUNS.find((run) => run.id === "gh-issue-6")!;

    expect(intakePlacementLabel(implement)).toBe("Implement");
    expect(intakePlacementActivity(implement)).toBe("Writing the retry handler.");
    expect(intakePlacementLabel(backlog)).toBe("In Backlog");
    expect(intakePlacementActivity(backlog)).toBe("Waiting for review.");
    expect(intakePlacementLabel(rejected)).toBe("Rejected");
    expect(intakePlacementLabel(held)).toBe("Not moved to Backlog");
    expect(intakeRelativeTime(180)).toBe("3h ago");
  });

  it("accepts only the three settings tabs", () => {
    expect(isIntakeSettingsTab("automation")).toBe(true);
    expect(isIntakeSettingsTab("runs")).toBe(true);
    expect(isIntakeSettingsTab("general")).toBe(true);
    expect(isIntakeSettingsTab("listen")).toBe(false);
  });

  it("defaults the authors filter to off", () => {
    expect(DEFAULT_GITHUB_INTAKE_SETTINGS.authorsWithAccess).toBe(false);
    expect(intakeSettingsFromApi("GitHub issues", undefined).authorsWithAccess).toBe(false);
  });

  it("round-trips the authors filter through the API shape", () => {
    const on = intakeSettingsFromApi("GitHub issues", { authorsWithAccess: true });
    expect(on.authorsWithAccess).toBe(true);
    expect(intakeSettingsToApi(on).authorsWithAccess).toBe(true);

    const off = intakeSettingsFromApi("GitHub issues", { authorsWithAccess: false });
    expect(off.authorsWithAccess).toBe(false);
    expect(intakeSettingsToApi(off).authorsWithAccess).toBe(false);
  });
});
