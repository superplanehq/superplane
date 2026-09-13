import { describe, expect, it } from "bun:test";

import {
  ANALYSIS_THINKING_STATES,
  analysisLiveWorkKind,
  reasoningLinesFromPlanningNotes,
  thinkingStatusFor,
} from "./analysisLiveWorkState";
import type { SplitRunStreamLine } from "./splitRunMocks";

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

describe("analysisLiveWorkKind", () => {
  it("stays idle after the machine waits or stops", () => {
    expect(analysisLiveWorkKind({ machineStatus: "waiting", items: [{ id: "n", text: "Read the ticket" }] })).toBe(
      "idle",
    );
    expect(analysisLiveWorkKind({ machineStatus: "passed", items: [] })).toBe("idle");
    expect(analysisLiveWorkKind({ machineStatus: "failed", items: [] })).toBe("idle");
  });

  it("shows thinking while the machine is on and no reasoning lines exist", () => {
    expect(analysisLiveWorkKind({ machineStatus: "starting", items: [] })).toBe("thinking");
    expect(analysisLiveWorkKind({ machineStatus: "running", items: [] })).toBe("thinking");
  });

  it("shows the two-line stream when reasoning lines exist", () => {
    expect(analysisLiveWorkKind({ machineStatus: "running", items: [{ id: "n", text: "Read billing.ts" }] })).toBe(
      "reasoning",
    );
  });
});

describe("thinkingStatusFor", () => {
  it("rotates the status line on a fixed interval", () => {
    expect(thinkingStatusFor(0)).toBe(ANALYSIS_THINKING_STATES[0]);
    expect(thinkingStatusFor(2400)).toBe(ANALYSIS_THINKING_STATES[1]);
    expect(thinkingStatusFor(4800)).toBe(ANALYSIS_THINKING_STATES[2]);
    expect(thinkingStatusFor(7200)).toBe(ANALYSIS_THINKING_STATES[0]);
  });
});

describe("reasoningLinesFromPlanningNotes", () => {
  it("keeps the last two human notes and tool summaries", () => {
    const lines = reasoningLinesFromPlanningNotes([
      note({
        id: "prompt",
        componentType: "prompt",
        componentName: "You are in a SuperPlane planning session. The repository is cloned.",
      }),
      note({
        id: "talk",
        componentType: "note",
        componentName: "I will read the empty billing view first.",
      }),
      note({
        id: "read",
        componentType: "read",
        noteParentId: "prompt",
        componentName: "web_src/src/pages/billing/EmptyState.tsx",
      }),
      note({
        id: "user",
        componentType: "note",
        userTalk: "message",
        componentName: "Use the current empty-state component.",
      }),
      note({
        id: "next",
        componentType: "note",
        componentName: "The empty view only names the page.",
      }),
    ]);

    expect(lines).toEqual([
      {
        id: "prompt-tools-0",
        text: "Read 1 file",
        details: ["web_src/src/pages/billing/EmptyState.tsx"],
      },
      { id: "next", text: "The empty view only names the page." },
    ]);
  });

  it("keeps command names on a ran-commands summary", () => {
    const lines = reasoningLinesFromPlanningNotes([
      note({
        id: "prompt",
        componentType: "prompt",
        componentName: "You are in a SuperPlane planning session. The repository is cloned.",
      }),
      note({
        id: "ls",
        componentType: "bash",
        noteParentId: "prompt",
        componentName: "ls src",
      }),
      note({
        id: "test",
        componentType: "bash",
        noteParentId: "prompt",
        componentName: "npm test",
      }),
    ]);

    expect(lines).toEqual([
      {
        id: "prompt-tools-0",
        text: "Ran 2 commands",
        details: ["ls src", "npm test"],
      },
    ]);
  });

  it("drops runner noise and JSON tool payloads", () => {
    expect(
      reasoningLinesFromPlanningNotes([
        note({ id: "noise", componentName: "Claude Code started · model=claude-opus-4-8" }),
        note({ id: "json", componentName: '{"message":"Ready to plan."}' }),
      ]),
    ).toEqual([]);
  });
});
