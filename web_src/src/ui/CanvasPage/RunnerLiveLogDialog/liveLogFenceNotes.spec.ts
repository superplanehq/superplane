import { describe, expect, it } from "bun:test";

import { appendLineToLatestSection, startCommandSection } from "./liveLogSections";
import type { LogState } from "./types";

function promptState(): LogState {
  return startCommandSection(
    { sections: [], orphanLines: [], error: null, isLoading: false, isStreaming: false },
    {
      index: 5,
      text: "Implementation",
      startedAtMs: 1,
      kind: "prompt",
      preview: "You are implementing",
    },
  );
}

describe("liveLogFenceNotes", () => {
  it("keeps a streamed fenced code block in one note", () => {
    let state = promptState();
    state = appendLineToLatestSection(state, "```go");
    state = appendLineToLatestSection(state, "// comment");
    state = appendLineToLatestSection(state, '\tfmt.Println("hi")');
    state = appendLineToLatestSection(state, "");
    state = appendLineToLatestSection(state, "```");

    expect(state.sections[0].lines).toEqual(["```go", "// comment", '\tfmt.Println("hi")', "", "```"]);
    expect(state.sections[0].events).toEqual([{ kind: "note", text: '```go\n// comment\n\tfmt.Println("hi")\n\n```' }]);
  });

  it("keeps collecting lines while a fence is open", () => {
    let state = promptState();
    state = appendLineToLatestSection(state, "```go");
    state = appendLineToLatestSection(state, "package main");

    expect(state.sections[0].events).toEqual([{ kind: "note", text: "```go\npackage main" }]);
  });

  it("keeps a longer outer fence open when an inner shorter fence appears", () => {
    let state = promptState();
    state = appendLineToLatestSection(state, "````markdown");
    state = appendLineToLatestSection(state, "```go");
    state = appendLineToLatestSection(state, "package main");
    state = appendLineToLatestSection(state, "```");
    state = appendLineToLatestSection(state, " ```text");
    state = appendLineToLatestSection(state, "still in the outer block");
    state = appendLineToLatestSection(state, "````");
    state = appendLineToLatestSection(state, "After.");

    expect(state.sections[0].events).toEqual([
      {
        kind: "note",
        text: "````markdown\n```go\npackage main\n```\n ```text\nstill in the outer block\n````",
      },
      { kind: "note", text: "After." },
    ]);
  });

  it("starts a new note after the closing fence", () => {
    let state = promptState();
    state = appendLineToLatestSection(state, "```go");
    state = appendLineToLatestSection(state, "package main");
    state = appendLineToLatestSection(state, "```");
    state = appendLineToLatestSection(state, "Done.");

    expect(state.sections[0].events).toEqual([
      { kind: "note", text: "```go\npackage main\n```" },
      { kind: "note", text: "Done." },
    ]);
  });
});
