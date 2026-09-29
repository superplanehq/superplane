import { describe, expect, it } from "bun:test";

import { groupSplitRunActivities } from "./splitRunActivityGroups";
import type { SplitRunPhase } from "./splitRunMocks";

function phase(id: string, startedAt?: string, pullRequestId?: string, revisionSha?: string): SplitRunPhase {
  return {
    id,
    name: id,
    status: "passed",
    duration: "1m",
    componentName: id,
    artifacts: [],
    stream: [],
    canvasSteps: [],
    pullRequestActivity: startedAt
      ? {
          startedAt,
          pullRequest: { id: pullRequestId, number: "12" },
          revision: { sha: revisionSha },
        }
      : undefined,
  };
}

describe("groupSplitRunActivities", () => {
  it("separates task automations and sorts pull request activity by time", () => {
    const groups = groupSplitRunActivities([
      phase("line-run"),
      phase("comment", "2026-08-26T11:00:00Z", "pr-12"),
      phase("checks", "2026-08-26T10:00:00Z", "pr-12", "a82fd91"),
      phase("new-revision", "2026-08-26T12:00:00Z", "pr-12", "d8b80c2"),
    ]);

    expect(groups.taskAutomationPhases.map((entry) => entry.id)).toEqual(["line-run"]);
    expect(groups.pullRequestActivityGroups).toHaveLength(1);
    expect(groups.pullRequestActivityGroups[0]?.phases.map((entry) => entry.id)).toEqual([
      "checks",
      "comment",
      "new-revision",
    ]);
  });
});
