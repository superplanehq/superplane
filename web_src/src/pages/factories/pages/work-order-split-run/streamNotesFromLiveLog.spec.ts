import { describe, expect, it } from "vitest";

import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { mergeLiveStreamNotes, notesForLiveStream, notesFromLiveLogSections } from "./streamNotesFromLiveLog";
import type { SplitRunStreamLine } from "./splitRunMocks";

function bashSection(): CommandSection {
  return {
    index: 1,
    text: "Set Up Git User",
    kind: "bash",
    preview: 'echo "Using superplaneagent@superplane.com"',
    lines: ["Using superplaneagent@superplane.com"],
    events: [],
    status: "passed",
    duration_ms: 20,
    started_at: 1,
    collapsed: true,
  };
}

function promptSection(): CommandSection {
  return {
    index: 5,
    text: "Implementation",
    kind: "prompt",
    preview: "You are implementing a fix",
    lines: ["Gathering context.", "package workers"],
    events: [
      { kind: "note", text: "Gathering context." },
      {
        kind: "tools",
        id: "5-tools-0",
        tools: [
          {
            id: "5-tool-0",
            kind: "read",
            text: "pkg/foo.go",
            lines: ["package workers"],
            status: "passed",
            duration_ms: 80,
          },
        ],
      },
    ],
    status: "passed",
    duration_ms: 900,
    started_at: 1,
    collapsed: true,
  };
}

describe("notesFromLiveLogSections", () => {
  it("hides setup and maps bash plus nested prompt tools", () => {
    const notes = notesFromLiveLogSections("agent", [
      {
        ...bashSection(),
        index: 0,
        kind: "setup",
        text: "Prepare Claude Code",
      },
      bashSection(),
      promptSection(),
    ]);

    expect(notes.map((note) => note.componentType)).toEqual(["bash", "prompt", "note", "read"]);
    expect(notes[0]?.componentName).toBe('echo "Using superplaneagent@superplane.com"');
    expect(notes[0]?.detail).toContain("Using superplaneagent");
    expect(notes[1]?.componentName).toBe("You are implementing a fix");
    expect(notes[3]?.componentName).toBe("pkg/foo.go");
    expect(notes[3]?.noteParentId).toBe("agent-step-5");
  });

  it("falls back to parseClaudeCodeLog for old -> [Tool] lines", () => {
    const notes = notesFromLiveLogSections("agent", [
      {
        index: 2,
        text: "Implementation",
        kind: "prompt",
        preview: "You are implementing",
        lines: ["-> [Bash] git status", "     On branch main"],
        events: [],
        status: "passed",
        duration_ms: 10,
        started_at: 1,
        collapsed: true,
      },
    ]);

    expect(notes.some((note) => note.componentType === "bash" && note.componentName.includes("git status"))).toBe(true);
  });

  it("skips blank prompt notes", () => {
    const notes = notesFromLiveLogSections("agent", [
      {
        ...promptSection(),
        events: [
          { kind: "note", text: "Claude Code started" },
          { kind: "note", text: "" },
          { kind: "note", text: "  " },
          {
            kind: "tools",
            id: "g1",
            tools: [
              {
                id: "t1",
                kind: "bash",
                text: "echo a",
                lines: [],
                status: "passed",
                duration_ms: 1,
              },
            ],
          },
        ],
      },
    ]);

    expect(notes.filter((note) => note.componentType === "note").map((note) => note.componentName)).toEqual([
      "Claude Code started",
    ]);
    expect(notes.some((note) => note.componentType === "bash" && note.componentName === "echo a")).toBe(true);
  });
});

function talkLine(id: string, text: string, componentType: "prompt" | "note"): SplitRunStreamLine {
  return {
    id,
    nodeId: "agent",
    at: "",
    note: true,
    componentName: text,
    componentType,
    status: "passed",
  };
}

describe("notesForLiveStream", () => {
  it("shows orphan live-log lines instead of Waiting for logs", () => {
    const notes = notesForLiveStream({
      nodeId: "agent",
      sections: [],
      orphanLines: ["Claude Code ready", "Cloning into 'repo'...", ""],
      error: null,
      isStreaming: true,
      nodeStatus: "running",
    });

    expect(notes?.map((note) => note.componentName)).toEqual(["Claude Code ready", "Cloning into 'repo'..."]);
    expect(notes?.some((note) => note.componentName.includes("Waiting for logs"))).toBe(false);
  });

  it("keeps Waiting for logs when the stream has no lines yet", () => {
    const notes = notesForLiveStream({
      nodeId: "agent",
      sections: [],
      orphanLines: [],
      error: null,
      isStreaming: true,
      nodeStatus: "running",
    });

    expect(notes).toEqual([
      expect.objectContaining({
        id: "agent-live-status",
        componentName: "Waiting for logs…",
        status: "running",
      }),
    ]);
  });
});

describe("mergeLiveStreamNotes", () => {
  it("inserts a Send before the open follow-up prompt and skips a say already in the log", () => {
    const live: SplitRunStreamLine[] = [
      {
        id: "hello",
        nodeId: "agent",
        at: "",
        note: true,
        componentType: "prompt",
        componentName: "Greet the user with say. Then stop.",
        status: "passed",
      },
      {
        id: "hi",
        nodeId: "agent",
        at: "",
        note: true,
        noteParentId: "hello",
        componentType: "note",
        componentName: "Hi! I'm ready to help you plan work in this repo. Tell me what you want to do.",
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
      {
        id: "picture",
        nodeId: "agent",
        at: "",
        note: true,
        noteParentId: "wait",
        componentType: "note",
        componentName: "I've got a clear picture. It's a simple puppy CRUD app.",
        status: "passed",
      },
    ];

    const merged = mergeLiveStreamNotes(live, [
      talkLine("greet", "Hi! I'm ready to help you plan work in this repo. Tell me what you want to do.", "note"),
      talkLine("user", "I want to add a new puppy field - color", "prompt"),
    ]);

    expect(merged.map((line) => line.id)).toEqual(["hello", "hi", "user", "wait", "picture"]);
  });

  it("keeps extras when the live log is empty", () => {
    const extra = [talkLine("user", "Add a Size field", "prompt")];
    expect(mergeLiveStreamNotes(undefined, extra)).toEqual(extra);
    expect(mergeLiveStreamNotes([], extra)).toEqual(extra);
  });
});
