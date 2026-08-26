import { describe, expect, it } from "vitest";

import {
  analyzingTicketsFromApi,
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  GITHUB_INTAKE_RUNS,
  isIntakeSettingsTab,
  intakePlacementActivity,
  intakePlacementLabel,
  intakeRelativeTime,
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

  it("keeps analyzing runs and tickets below the minimum in the Intake list", () => {
    expect(
      analyzingTicketsFromApi(
        [
          { id: "run-analyzing", title: "Handle duplicate refunds on retry", placement: "PLACEMENT_ANALYZING" },
          { id: "run-backlog", title: "Already in Backlog", placement: "PLACEMENT_BACKLOG", confidencePct: 81 },
          {
            id: "run-held",
            title: "Document the refund webhook contract",
            placement: "PLACEMENT_BELOW_THRESHOLD",
            confidencePct: 52,
          },
          { id: "run-rejected", title: "A person rejected this", placement: "PLACEMENT_REJECTED" },
        ],
        "app-github-issues-intake",
      ),
    ).toEqual([
      {
        id: "run-analyzing",
        title: "Handle duplicate refunds on retry",
        runId: "run-analyzing",
        appId: "app-github-issues-intake",
      },
      {
        id: "run-held",
        title: "Document the refund webhook contract",
        runId: "run-held",
        appId: "app-github-issues-intake",
        outcome: "below-threshold",
        confidencePct: 52,
      },
    ]);
  });
});
