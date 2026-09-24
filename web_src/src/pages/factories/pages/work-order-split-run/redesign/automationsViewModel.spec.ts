import { describe, expect, it } from "bun:test";

import { SPLIT_RUN_SUPER503 } from "../splitRunSuper503Fixture";
import { allStages, outcomeSummary, stagesFromFixture } from "./automationsViewModel";

describe("automations view model for SUPER-503", () => {
  it("keeps task stages separate from the pull request runs", () => {
    const groups = stagesFromFixture(SPLIT_RUN_SUPER503);

    expect(groups.taskStages.map((stage) => stage.name)).toEqual(["Backlog", "Analysis", "Implement", "Done"]);
    expect(groups.pullRequestGroups).toHaveLength(1);
    expect(groups.pullRequestGroups[0].stages).toHaveLength(17);
    expect(allStages(groups)).toHaveLength(21);
  });

  it("summarizes the merged pull request, the confidence check, and the spend", () => {
    const outcome = outcomeSummary(SPLIT_RUN_SUPER503);
    const implement = stagesFromFixture(SPLIT_RUN_SUPER503).taskStages.find((stage) => stage.id === "implement");

    expect(outcome.statusLabel).toBe("Completed");
    expect(outcome.pullRequests.map((pullRequest) => pullRequest.number)).toEqual(["7771"]);
    expect(outcome.checksPassed).toBe(1);
    expect(outcome.checksTotal).toBe(1);
    expect(outcome.models).toEqual(["grok-4.6"]);
    expect(implement?.agentSteps.map((step) => step.title)).toContain("Implementation");
    expect(implement?.rawLog).toContain("$ Implementation");
  });
});
