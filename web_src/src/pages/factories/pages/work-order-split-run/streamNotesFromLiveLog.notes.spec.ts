import { describe, expect, it } from "bun:test";

import { appendLineToLatestSection, startCommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/liveLogSections";
import type { LogState } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { mergeLiveStreamNotes, notesFromLiveLogSections, streamNoteTextMatches } from "./streamNotesFromLiveLog";
import type { SplitRunStreamLine } from "./splitRunMocks";

function emptyLiveLogState(): LogState {
  return { sections: [], orphanLines: [], pendingRecords: [], error: null, isLoading: false, isStreaming: false };
}

describe("notesFromLiveLogSections joined notes", () => {
  it("keeps blank lines inside a joined prompt note", () => {
    let state = startCommandSection(emptyLiveLogState(), {
      index: 5,
      text: "Implementation",
      startedAtMs: 1,
      kind: "prompt",
      preview: "You are implementing",
    });
    state = appendLineToLatestSection(state, "Here is the change:");
    state = appendLineToLatestSection(state, "");
    state = appendLineToLatestSection(state, "```ts");
    state = appendLineToLatestSection(state, "const n = 1;");
    state = appendLineToLatestSection(state, "");
    state = appendLineToLatestSection(state, "const m = 2;");
    state = appendLineToLatestSection(state, "```");

    const notes = notesFromLiveLogSections("agent", state.sections);
    const talk = notes.find((note) => note.componentType === "note");

    expect(talk?.componentName).toBe("Here is the change:\n\n```ts\nconst n = 1;\n\nconst m = 2;\n```");
  });
});

describe("streamNoteTextMatches", () => {
  it("pairs a joined live note with a stored extra that continues on a new line", () => {
    expect(
      streamNoteTextMatches(
        "Hi! I'm ready to help you plan work in this repo.",
        "Hi! I'm ready to help you plan work in this repo.\n\nTell me what you want to do.",
      ),
    ).toBe(true);
    expect(
      streamNoteTextMatches(
        "Hi! I'm ready to help you plan work in this repo.\n\nTell me what you want to do.",
        "Hi! I'm ready to help you plan work in this repo.",
      ),
    ).toBe(true);
  });

  it("does not treat a distinct extra as the same as a short live note", () => {
    expect(
      streamNoteTextMatches("I found the issue.", "I found the issue. The tests fail because the parser drops fences."),
    ).toBe(false);
  });
});

describe("mergeLiveStreamNotes joined notes", () => {
  it("keeps a stored extra that only shares a short live prefix", () => {
    const live: SplitRunStreamLine[] = [
      {
        id: "found",
        nodeId: "agent",
        at: "",
        note: true,
        componentName: "I found the issue.",
        componentType: "note",
        status: "passed",
      },
      {
        id: "wait",
        nodeId: "agent",
        at: "",
        note: true,
        componentType: "prompt",
        componentName: "Wait for the next user message",
        status: "running",
      },
    ];

    const merged = mergeLiveStreamNotes(live, [
      {
        id: "stored",
        nodeId: "agent",
        at: "",
        note: true,
        componentName: "I found the issue. The tests fail because the parser drops fences.",
        componentType: "note",
        status: "passed",
      },
    ]);

    expect(merged.map((line) => line.id)).toEqual(["found", "stored", "wait"]);
  });
});
