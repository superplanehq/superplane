import { describe, expect, it } from "bun:test";

import {
  ANALYSIS_REPLY_THINKING_STATES,
  ANALYSIS_THINKING_INTERVAL_MS,
  ANALYSIS_THINKING_STATES,
  analysisLiveWorkKind,
  hasAgentReasoning,
  hidePreviousAgentStream,
  liveReasoningForTurn,
  previousAgentStreamText,
  reasoningLinesFromPlanningNotes,
  thinkingStatusFor,
  waitingForAgentReply,
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

describe("waitingForAgentReply", () => {
  it("waits before the first agent line and after a user reply", () => {
    expect(waitingForAgentReply([])).toBe(true);
    expect(waitingForAgentReply([{ role: "user" }])).toBe(true);
    expect(waitingForAgentReply([{ role: "agent" }])).toBe(false);
    expect(waitingForAgentReply([{ role: "agent" }, { role: "user" }])).toBe(true);
  });
});

describe("hidePreviousAgentStream", () => {
  it("hides leftover live notes only after a user reply", () => {
    expect(hidePreviousAgentStream([])).toBe(false);
    expect(hidePreviousAgentStream([{ role: "agent" }])).toBe(false);
    expect(hidePreviousAgentStream([{ role: "agent" }, { role: "user" }])).toBe(true);
  });
});

describe("previousAgentStreamText", () => {
  it("returns prior agent lines only after a user reply", () => {
    expect(previousAgentStreamText([])).toBeUndefined();
    expect(previousAgentStreamText([{ role: "agent", text: "I asked one survey question." }])).toBeUndefined();
    expect(
      previousAgentStreamText([
        { role: "agent", text: "I asked one survey question." },
        { role: "user", text: "Add cats." },
      ]),
    ).toBe("I asked one survey question.");
    expect(
      previousAgentStreamText([
        { role: "agent", text: "Score: 2 of 5.\n\nAnalysis files written: /tmp/intent.md." },
        { role: "user", text: "Add cats." },
      ]),
    ).toBe("Score: 2 of 5.\n\nAnalysis files written: /tmp/intent.md.");
  });
});

describe("liveReasoningForTurn", () => {
  it("removes only the previous agent line and keeps the next stream", () => {
    const items = [
      { id: "note", text: "I asked one survey question to pin down the intended reading." },
      { id: "tools", text: "Ran 1 command", details: ["git clone"] },
      { id: "next", text: "The separate-pages-per-animal reading is clear." },
    ];
    expect(
      liveReasoningForTurn(items, "I asked one survey question to pin down the intended reading. Once that is answered."),
    ).toEqual([items[1], items[2]]);
    expect(liveReasoningForTurn(items)).toEqual(items);
  });

  it("drops a clamped leftover of the previous agent paragraph", () => {
    const previous =
      "Because the premise is false, I scored this low (confidence 2) and asked one clarifying question: close it as already fixed with a test, upgrade the plain-text 404 into a real page, or provide steps where the 500 still occurs. Spec, score, and survey are published. Files written: /tmp/intake-analysis.json and /tmp/intent.md.";
    const items = [
      {
        id: "note",
        text: "Because the premise is false, I scored this low (confidence 2) and asked one clarifying question: close it as already fixed with a test, up…",
      },
      { id: "tools", text: "Ran 1 command", details: ["git clone"] },
    ];
    expect(liveReasoningForTurn(items, previous)).toEqual([items[1]]);
  });

  it("drops a leftover that is only the last line of the previous agent message", () => {
    const previous =
      "So I set confidence to 2 and posted a survey to pick the reading and how the type is entered. Once that is answered I can raise the score.\n\nAnalysis files written: /tmp/intake-analysis.json (validated with jq) and /tmp/intent.md.";
    const items = [
      {
        id: "note",
        text: "Analysis files written: /tmp/intake-analysis.json (validated with jq) and /tmp/intent.md.",
      },
      { id: "tools", text: "Ran 1 command", details: ["git clone"] },
    ];
    expect(liveReasoningForTurn(items, previous)).toEqual([items[1]]);
  });

  it("drops leftover notes by id when the published text does not match", () => {
    const items = [
      { id: "note", text: "Because the premise is false, I scored this low." },
      { id: "next", text: "Clear now. I will lock in the behavior with a test." },
    ];
    expect(liveReasoningForTurn(items, "A different published paragraph.", new Set(["note"]))).toEqual([items[1]]);
  });
});

describe("hasAgentReasoning", () => {
  it("treats a plain note as the first agent line", () => {
    expect(hasAgentReasoning([{ id: "note", text: "I have enough understanding of the repository." }])).toBe(true);
  });

  it("ignores ran-command summaries", () => {
    expect(hasAgentReasoning([{ id: "tools", text: "Ran 1 command", details: ["git clone"] }])).toBe(false);
  });
});

describe("thinkingStatusFor", () => {
  it("rotates the status line on a fixed interval", () => {
    expect(thinkingStatusFor(0)).toBe(ANALYSIS_THINKING_STATES[0]);
    expect(thinkingStatusFor(2400)).toBe(ANALYSIS_THINKING_STATES[1]);
    expect(thinkingStatusFor(4800)).toBe(ANALYSIS_THINKING_STATES[2]);
    expect(thinkingStatusFor(7200)).toBe(ANALYSIS_THINKING_STATES[0]);
  });

  it("uses reply states after the session is already open", () => {
    expect(thinkingStatusFor(0, ANALYSIS_THINKING_INTERVAL_MS, ANALYSIS_REPLY_THINKING_STATES)).toBe(
      ANALYSIS_REPLY_THINKING_STATES[0],
    );
    expect(thinkingStatusFor(2400, ANALYSIS_THINKING_INTERVAL_MS, ANALYSIS_REPLY_THINKING_STATES)).toBe(
      ANALYSIS_REPLY_THINKING_STATES[1],
    );
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
