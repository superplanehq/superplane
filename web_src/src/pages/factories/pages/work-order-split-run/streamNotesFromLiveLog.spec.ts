import { describe, expect, it } from "vitest";

import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { notesFromLiveLogSections } from "./streamNotesFromLiveLog";

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
});
