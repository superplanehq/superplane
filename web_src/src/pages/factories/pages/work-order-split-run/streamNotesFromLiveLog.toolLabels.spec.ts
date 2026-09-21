import { describe, expect, it } from "bun:test";

import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { notesFromLiveLogSections } from "./streamNotesFromLiveLog";

describe("notesFromLiveLogSections tool labels", () => {
  it("formats legacy tool input without exposing transport JSON", () => {
    const section: CommandSection = {
      index: 5,
      text: "Implementation",
      kind: "prompt",
      preview: "Explore the repository",
      lines: [],
      events: [
        {
          kind: "tools",
          id: "5-tools-0",
          tools: [
            tool("task-1", "task", '{"description":"Explore repository","prompt":"Read the documentation"}'),
            tool("bash-1", "bash", '{"command":"git status"}'),
            tool("read-1", "read", '{"path":"README.md"}'),
          ],
        },
      ],
      status: "passed",
      duration_ms: 900,
      started_at: 1,
      collapsed: true,
    };

    const notes = notesFromLiveLogSections("agent", [section]);

    expect(notes.slice(1).map((note) => note.componentName)).toEqual(["task", "git status", "README.md"]);
  });
});

function tool(id: string, kind: string, text: string) {
  return { id, kind, text, lines: [], status: "passed" as const, duration_ms: 80 };
}
