import { describe, expect, it } from "bun:test";

import { appendLineToLatestSection, startCommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/liveLogSections";
import type { LogState } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { notesFromLiveLogSections } from "./streamNotesFromLiveLog";

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
