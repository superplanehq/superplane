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
    const state = startCommandSection(emptyState(), {
      index: 1,
      text: "Implementation",
      startedAtMs: 10,
      kind: "prompt",
      preview: "You are implementing",
    });
    expect(state.sections[0]).toMatchObject({
      kind: "prompt",
      preview: "You are implementing",
      text: "Implementation",
    });
    expect(sectionTitle(state.sections[0])).toBe("You are implementing");
  });

  it("uses only the first line of a multi-line preview for the compact title", () => {
    const state = startCommandSection(emptyState(), {
      index: 1,
      text: "Implementation",
      startedAtMs: 10,
      kind: "prompt",
      preview: "You are implementing\nthe rest of the prompt.",
    });
    expect(sectionTitle(state.sections[0])).toBe("You are implementing");
  });

  it("nests tool output under a prompt section and keeps notes between tools", () => {
    let state = startCommandSection(emptyState(), {
      index: 5,
      text: "Implementation",
      startedAtMs: 1,
      kind: "prompt",
      preview: "You are implementing",
    });
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
    let state = startCommandSection(emptyState(), {
      index: 5,
      text: "Implementation",
      startedAtMs: 1,
      kind: "prompt",
      preview: "You are implementing",
    });
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

  it("attaches replayed tools to the active command, not the latest section", () => {
    let state = startCommandSection(emptyState(), {
      index: 2,
      text: "Plan with the user",
      startedAtMs: 1,
      kind: "prompt",
      preview: "Greet the user",
    });
    state = startToolOnLatestSection(state, "bash", "ls", "call_1");
    state = endToolOnLatestSection(state, "passed", 5, "call_1");
    state = startCommandSection(state, {
      index: 1000,
      text: "Wait for the next message",
      startedAtMs: 2,
      kind: "prompt",
      preview: "Wait for the next user message",
    });

    state = startToolOnLatestSection(state, "bash", "ls", "call_1", 2);
    state = endToolOnLatestSection(state, "failed", 99, "call_1", 2);

    const first = state.sections[0].events[0];
    expect(first?.kind).toBe("tools");
    if (first?.kind !== "tools") {
      throw new Error("expected tools group");
    }
    expect(first.tools).toHaveLength(1);
    expect(first.tools[0]).toMatchObject({ status: "passed", duration_ms: 5 });
    expect(state.sections[1].events).toEqual([]);
  });

  it("ignores replayed tool records with the same source id", () => {
    let state = startCommandSection(emptyState(), {
      index: 5,
      text: "Implementation",
      startedAtMs: 1,
      kind: "prompt",
      preview: "You are implementing",
    });
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

  it("keeps overlapping tool stdout as notes instead of the newest tool", () => {
    let state = startCommandSection(emptyState(), {
      index: 5,
      text: "Implementation",
      startedAtMs: 1,
      kind: "prompt",
      preview: "You are implementing",
    });
    state = startToolOnLatestSection(state, "read", "a.go", "toolu_a");
    state = startToolOnLatestSection(state, "bash", "git status", "toolu_b");
    state = appendLineToLatestSection(state, "boom");

    const section = state.sections[0];
    expect(section.events.at(-1)).toEqual({ kind: "note", text: "boom" });
    const tools = section.events[0];
    expect(tools?.kind).toBe("tools");
    if (tools?.kind !== "tools") {
      throw new Error("expected tools group");
    }
    expect(tools.tools[0].lines).toEqual([]);
    expect(tools.tools[1].lines).toEqual([]);
  });

  it("does not turn blank stdout into prompt notes between tools", () => {
    let state = startCommandSection(emptyState(), {
      index: 5,
      text: "Implementation",
      startedAtMs: 1,
      kind: "prompt",
      preview: "You are implementing",
    });
    state = startToolOnLatestSection(state, "bash", "echo a", "toolu_a");
    state = endToolOnLatestSection(state, "passed", 1, "toolu_a");
    state = appendLineToLatestSection(state, "");
    state = appendLineToLatestSection(state, "   ");
    state = startToolOnLatestSection(state, "bash", "echo b", "toolu_b");

    const section = state.sections[0];
    expect(section.lines).toEqual(["", "   "]);
    expect(section.events.filter((event) => event.kind === "note")).toEqual([]);
    expect(section.events).toHaveLength(1);
    expect(section.events[0]?.kind).toBe("tools");
    if (section.events[0]?.kind !== "tools") {
      throw new Error("expected tools group");
    }
    expect(section.events[0].tools.map((tool) => tool.id)).toEqual(["toolu_a", "toolu_b"]);
  });

  it("keeps bash section lines flat", () => {
    let state = startCommandSection(emptyState(), {
      index: 0,
      text: "Create Branch",
      startedAtMs: 1,
      kind: "bash",
      preview: "git clone",
    });
    state = appendLineToLatestSection(state, "Cloning...");
    expect(state.sections[0].events).toEqual([]);
    expect(state.sections[0].lines).toEqual(["Cloning..."]);
  });

  it("does not replay earlier command output onto the current prompt", () => {
    let state = startCommandSection(emptyState(), {
      index: 0,
      text: "Prepare OpenRouter agent",
      startedAtMs: 1,
      kind: "setup",
      preview: "Prepare OpenRouter agent",
    });
    state = appendLineToLatestSection(state, "Agent ready");
    state = appendLineToLatestSection(state, "opencode=1.18.3");
    state = completeCommandSection(state, 0, "passed", 10);
    state = startCommandSection(state, {
      index: 1,
      text: "Clone Repo",
      startedAtMs: 2,
      kind: "bash",
      preview: "git clone",
    });
    state = appendLineToLatestSection(state, "Cloning into 'repo'...");
    state = completeCommandSection(state, 1, "passed", 20);
    state = startCommandSection(state, {
      index: 2,
      text: "Plan with the user",
      startedAtMs: 3,
      kind: "prompt",
      preview: "Greet the user",
    });
    state = appendLineToLatestSection(state, "Starting OpenCode");
    state = appendLineToLatestSection(state, "Hello! How can I help you today?");

    const skip = new Map<number, number>([
      [0, state.sections[0].lines.length],
      [1, state.sections[1].lines.length],
      [2, state.sections[2].lines.length],
    ]);
    state = appendLineToLatestSection(state, "Agent ready", skip, 0);
    state = appendLineToLatestSection(state, "opencode=1.18.3", skip, 0);
    state = appendLineToLatestSection(state, "Cloning into 'repo'...", skip, 1);
    state = appendLineToLatestSection(state, "Starting OpenCode", skip, 2);
    state = appendLineToLatestSection(state, "Hello! How can I help you today?", skip, 2);
    state = appendLineToLatestSection(state, "What should we work on?", skip, 2);

    const prompt = state.sections[2];
    expect(prompt.lines).toEqual(["Starting OpenCode", "Hello! How can I help you today?", "What should we work on?"]);
    expect(prompt.events.map((event) => (event.kind === "note" ? event.text : event.kind))).toEqual([
      "Starting OpenCode",
      "Hello! How can I help you today?",
      "What should we work on?",
    ]);
  });

  it("drops replayed clone and greet lines that already belong to earlier commands", () => {
    let state = startCommandSection(emptyState(), {
      index: 0,
      text: "Prepare OpenRouter agent",
      startedAtMs: 1,
      kind: "setup",
      preview: "Prepare OpenRouter agent",
    });
    state = appendLineToLatestSection(state, "Agent ready");
    state = completeCommandSection(state, 0, "passed", 10);
    state = startCommandSection(state, {
      index: 1,
      text: "Clone Repo",
      startedAtMs: 2,
      kind: "bash",
      preview: "git clone",
    });
    state = appendLineToLatestSection(state, "Cloning into 'repo'...");
    state = completeCommandSection(state, 1, "passed", 20);
    state = startCommandSection(state, {
      index: 2,
      text: "Plan with the user",
      startedAtMs: 3,
      kind: "prompt",
      preview: "Greet the user",
    });
    state = appendLineToLatestSection(state, "Hello! How can I help you today?");
    state = completeCommandSection(state, 2, "passed", 30);
    state = startCommandSection(state, {
      index: 1000,
      text: "Wait for the next message",
      startedAtMs: 4,
      kind: "prompt",
      preview: "Wait for the next user message",
    });
    state = appendLineToLatestSection(state, "I've proposed a draft.");

    state = appendLineToLatestSection(state, "Agent ready");
    state = appendLineToLatestSection(state, "Cloning into 'repo'...");
    state = appendLineToLatestSection(state, "Hello! How can I help you today?");
    state = appendLineToLatestSection(state, "Agent ready", undefined, 0);
    state = appendLineToLatestSection(state, "Cloning into 'repo'...", undefined, 1);
    state = appendLineToLatestSection(state, "Hello! How can I help you today?", undefined, 2);

    const prompt = state.sections[3];
    expect(prompt.lines).toEqual(["I've proposed a draft."]);
    expect(prompt.events).toEqual([{ kind: "note", text: "I've proposed a draft." }]);
  });

  it("drops raw turn JSON so it does not appear as a log line", () => {
    let state = startCommandSection(emptyState(), {
      index: 5,
      text: "Implementation",
      startedAtMs: 1,
      kind: "prompt",
      preview: "You are implementing",
    });
    state = appendLineToLatestSection(state, '{"type":"turn","turn":1,"usage":{"input_tokens":2,"output_tokens":1}}');
    state = appendLineToLatestSection(state, "Turn 1 · 3 tokens");

    expect(state.sections[0].lines).toEqual(["Turn 1 · 3 tokens"]);
    expect(state.sections[0].events).toEqual([{ kind: "note", text: "Turn 1 · 3 tokens" }]);
  });
});
