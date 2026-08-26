import { describe, expect, it } from "vitest";

import {
  appendLineToLatestSection,
  completeCommandSection,
  endToolOnLatestSection,
  sectionTitle,
  startCommandSection,
  startToolOnLatestSection,
} from "./liveLogSections";
import type { LogState } from "./types";

function emptyState(): LogState {
  return { sections: [], orphanLines: [], error: null, isStreaming: false };
}

describe("liveLogSections", () => {
  it("stores kind and preview on cmd_start", () => {
    const state = startCommandSection(emptyState(), 1, "Implementation", 10, "prompt", "You are implementing");
    expect(state.sections[0]).toMatchObject({
      kind: "prompt",
      preview: "You are implementing",
      text: "Implementation",
    });
    expect(sectionTitle(state.sections[0])).toBe("You are implementing");
  });

  it("nests tool output under a prompt section and keeps notes between tools", () => {
    let state = startCommandSection(emptyState(), 5, "Implementation", 1, "prompt", "You are implementing");
    state = appendLineToLatestSection(state, "Gathering context.");
    state = startToolOnLatestSection(state, "read", "pkg/foo.go");
    state = appendLineToLatestSection(state, "package workers");
    state = endToolOnLatestSection(state, "passed", 80);
    state = startToolOnLatestSection(state, "bash", "git status");
    state = appendLineToLatestSection(state, "On branch main");
    state = endToolOnLatestSection(state, "passed", 40);
    state = completeCommandSection(state, 5, "passed", 90000);

    const section = state.sections[0];
    expect(section.events[0]).toEqual({ kind: "note", text: "Gathering context." });
    expect(section.events[1]?.kind).toBe("tools");
    if (section.events[1]?.kind !== "tools") {
      throw new Error("expected tools group");
    }
    expect(section.events[1].tools).toHaveLength(2);
    expect(section.events[1].tools[0]).toMatchObject({
      kind: "read",
      text: "pkg/foo.go",
      status: "passed",
      lines: ["package workers"],
    });
    expect(section.events[1].tools[1]).toMatchObject({
      kind: "bash",
      text: "git status",
      lines: ["On branch main"],
    });
  });

  it("ends overlapping tools by source id, not start order", () => {
    let state = startCommandSection(emptyState(), 5, "Implementation", 1, "prompt", "You are implementing");
    state = startToolOnLatestSection(state, "read", "a.go", "toolu_a");
    state = startToolOnLatestSection(state, "bash", "git status", "toolu_b");
    state = endToolOnLatestSection(state, "failed", 10, "toolu_b");
    state = endToolOnLatestSection(state, "passed", 20, "toolu_a");

    const tools = state.sections[0].events[0];
    expect(tools?.kind).toBe("tools");
    if (tools?.kind !== "tools") {
      throw new Error("expected tools group");
    }
    expect(tools.tools[0]).toMatchObject({ sourceId: "toolu_a", status: "passed", duration_ms: 20 });
    expect(tools.tools[1]).toMatchObject({ sourceId: "toolu_b", status: "failed", duration_ms: 10 });
  });

  it("ignores replayed tool records with the same source id", () => {
    let state = startCommandSection(emptyState(), 5, "Implementation", 1, "prompt", "You are implementing");
    state = startToolOnLatestSection(state, "read", "a.go", "toolu_a");
    state = endToolOnLatestSection(state, "passed", 20, "toolu_a");
    state = startToolOnLatestSection(state, "read", "a.go", "toolu_a");
    state = endToolOnLatestSection(state, "failed", 99, "toolu_a");

    const tools = state.sections[0].events[0];
    expect(tools?.kind).toBe("tools");
    if (tools?.kind !== "tools") {
      throw new Error("expected tools group");
    }
    expect(tools.tools).toHaveLength(1);
    expect(tools.tools[0]).toMatchObject({ status: "passed", duration_ms: 20 });
  });

  it("keeps bash section lines flat", () => {
    let state = startCommandSection(emptyState(), 0, "Create Branch", 1, "bash", "git clone");
    state = appendLineToLatestSection(state, "Cloning...");
    expect(state.sections[0].events).toEqual([]);
    expect(state.sections[0].lines).toEqual(["Cloning..."]);
  });
});
