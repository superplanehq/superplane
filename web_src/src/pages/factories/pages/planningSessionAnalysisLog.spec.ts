import { describe, expect, it } from "vitest";

import { groupPlanningSessionLog } from "./planningSessionLog";
import type { SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

function note(partial: Partial<SplitRunStreamLine> & Pick<SplitRunStreamLine, "id">): SplitRunStreamLine {
  return {
    nodeId: "agent",
    at: "",
    note: true,
    status: "passed",
    componentName: "",
    ...partial,
  };
}

describe("analysis planning-session log", () => {
  it("hides the Analyze and score prompt header", () => {
    const groups = groupPlanningSessionLog([
      note({ id: "agent-step-2", componentType: "prompt", componentName: "Analyze and score" }),
      note({
        id: "talk",
        noteParentId: "agent-step-2",
        componentType: "note",
        componentName: "I will read the ticket and the repository.",
      }),
    ]);

    expect(groups).toHaveLength(1);
    expect(groups[0]?.line.componentName).toBe("");
    expect(groups[0]?.events[0]).toEqual(
      expect.objectContaining({ kind: "note", line: expect.objectContaining({ id: "talk" }) }),
    );
  });
});
