import { describe, expect, it } from "bun:test";

import { groupClaudeSteps, groupSplitRunStream } from "./phaseLogStream";
import { groupSplitRunActivities } from "./splitRunActivityGroups";
import { autoExpandedPhaseId } from "./splitRunMocks";
import { SPLIT_RUN_SUPER503 } from "./splitRunSuper503Fixture";

describe("SPLIT_RUN_SUPER503", () => {
  it("closes after Implement and seventeen pull request runs on #7771", () => {
    const groups = groupSplitRunActivities(SPLIT_RUN_SUPER503.phases);

    expect(groups.taskAutomationPhases.map((phase) => phase.name)).toEqual([
      "Backlog",
      "Analysis",
      "Implement",
      "Done",
    ]);
    expect(groups.pullRequestActivityGroups).toHaveLength(1);
    expect(groups.pullRequestActivityGroups[0].pullRequest?.number).toBe("7771");
    expect(groups.pullRequestActivityGroups[0].pullRequest?.state).toBe("STATE_MERGED");
    expect(groups.pullRequestActivityGroups[0].phases).toHaveLength(17);
    expect(SPLIT_RUN_SUPER503.footer.note?.headline).toBe("PR Closure marked this task as successful");
  });

  it("carries the Implement transcript, the plan, and a confidence score", () => {
    const implement = SPLIT_RUN_SUPER503.phases.find((phase) => phase.id === "implement");
    const nodes = groupSplitRunStream(implement?.stream ?? []);
    const agent = nodes.find((node) => node.line.component === "runnerOpenRouter");
    const steps = groupClaudeSteps(agent?.notes ?? []);
    const analysis = SPLIT_RUN_SUPER503.phases.find((phase) => phase.id === "analysis");

    expect(steps.map((step) => step.line.componentName)).toContain("Implementation");
    expect(steps[0].line.componentName).toBe("Clone Repo");
    const analysisNodes = groupSplitRunStream(analysis?.stream ?? []);
    const analysisAgent = analysisNodes.find((node) => node.line.component === "runnerOpenRouter");
    const analysisSteps = groupClaudeSteps(analysisAgent?.notes ?? []);
    expect(analysisSteps.slice(0, 2).map((step) => [step.line.componentName, step.line.duration])).toEqual([
      ["Clone repository", "3s"],
      ["Refine Task", "1m 55s"],
    ]);
    expect(analysis?.checks?.[0]?.name).toBe("Confidence score");
    expect(analysis?.artifacts[0]?.data).toMatchObject({ name: "plan.md" });
    expect(autoExpandedPhaseId(SPLIT_RUN_SUPER503)).toBe("implement");
  });
});
