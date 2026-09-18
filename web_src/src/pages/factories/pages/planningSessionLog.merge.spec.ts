import { describe, expect, it } from "bun:test";

import { mergePlanningSessionNotes } from "./planningSessionLog";
import type { SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

function note(
  partial: Partial<SplitRunStreamLine> & Pick<SplitRunStreamLine, "id" | "componentName">,
): SplitRunStreamLine {
  return {
    nodeId: "agent",
    at: "",
    note: true,
    status: "passed",
    ...partial,
  };
}

describe("mergePlanningSessionNotes joined notes", () => {
  it("pairs a joined live note with a stored extra that continues the same text", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "greet",
        componentType: "prompt",
        componentName: "Plan with the user",
      }),
      note({
        id: "hi",
        noteParentId: "greet",
        componentType: "note",
        componentName: "Hi! I'm ready to help you plan work in this repo.",
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "stored",
        componentType: "note",
        componentName: "Hi! I'm ready to help you plan work in this repo.\n\nTell me what you want to do.",
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["greet", "hi"]);
  });
});
