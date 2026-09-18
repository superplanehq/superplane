import { describe, expect, it } from "bun:test";

import { groupPlanningSessionLog, isPlanningSessionNoise, mergePlanningSessionNotes } from "./planningSessionLog";
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

describe("planning session joined notes", () => {
  it("does not treat mixed talk as runner noise", () => {
    expect(isPlanningSessionNoise("The repository is ready.\nRetrying API\nuntil success")).toBe(false);
  });

  it("keeps agent lines that look like runner noise inside a joined note", () => {
    const groups = groupPlanningSessionLog([
      note({
        id: "prompt",
        componentType: "prompt",
        componentName: "Plan with the user",
      }),
      note({
        id: "talk",
        noteParentId: "prompt",
        componentType: "note",
        componentName:
          "Planning session tools enabled\nThe retry loop prints:\nRetrying API\n✓ done\nuntil success\n✓ done · 1 turns · 1.5s",
      }),
    ]);

    expect(groups[0]?.events[0]).toEqual(
      expect.objectContaining({
        kind: "note",
        line: expect.objectContaining({
          id: "talk",
          componentName: "The retry loop prints:\nRetrying API\n✓ done\nuntil success",
        }),
      }),
    );
  });
});

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

  it("keeps a stored extra that only shares a short live prefix", () => {
    const live: SplitRunStreamLine[] = [
      note({
        id: "greet",
        componentType: "prompt",
        componentName: "Plan with the user",
      }),
      note({
        id: "found",
        noteParentId: "greet",
        componentType: "note",
        componentName: "I found the issue.",
        orderKey: 10,
      }),
    ];

    const merged = mergePlanningSessionNotes(live, [
      note({
        id: "stored",
        componentType: "note",
        componentName: "I found the issue. The tests fail because the parser drops fences.",
        orderKey: 99,
      }),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["greet", "found", "stored"]);
    expect(merged.find((line) => line.id === "found")?.orderKey).toBe(10);
  });
});
