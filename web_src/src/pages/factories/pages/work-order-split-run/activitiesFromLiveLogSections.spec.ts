import { describe, expect, it } from "bun:test";

import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { activitiesFromLiveLogSections } from "./streamNotesFromLiveLog";

describe("activitiesFromLiveLogSections", () => {
  it("turns section notes and tools into a live activity without tool output dumps", () => {
    const activities = activitiesFromLiveLogSections([
      {
        index: 1,
        text: "Implementation",
        kind: "prompt",
        preview: "",
        lines: [],
        events: [
          { kind: "note", text: "Let me read the factory handler." },
          {
            kind: "tools",
            id: "tools-1",
            tools: [
              {
                id: "tool-1",
                kind: "read",
                text: "/home/node/.superplane/homes/repo/web_src/src/hooks/useFactoryPRFeedbackData.ts",
                lines: ["<path>/home/node/.superplane/homes/repo/web_src/src/hooks/useFactoryPRFeedbackData.ts</path>"],
                status: "passed",
                duration_ms: 12,
              },
            ],
          },
        ],
        status: "running",
        duration_ms: null,
        started_at: 1,
        collapsed: false,
      },
    ]);

    expect(activities).toHaveLength(1);
    expect(activities[0]?.status).toBe("running");
    expect(activities[0]?.items).toEqual([
      expect.objectContaining({
        type: "content",
        kind: "assistant",
        text: "Let me read the factory handler.",
        status: "passed",
      }),
      expect.objectContaining({ type: "tool", kind: "read", output: "" }),
    ]);
  });

  it("drops a bare Thinking note so it does not become a message", () => {
    const activities = activitiesFromLiveLogSections([
      {
        index: 1,
        text: "Implementation",
        kind: "prompt",
        preview: "",
        lines: [],
        events: [
          { kind: "note", text: "Thinking" },
          { kind: "note", text: "Let me read the factory handler." },
        ],
        status: "running",
        duration_ms: null,
        started_at: 1,
        collapsed: false,
      },
    ]);

    expect(activities[0]?.items).toEqual([
      expect.objectContaining({ type: "content", text: "Let me read the factory handler." }),
    ]);
  });

  it("leaves bash sections out so they stay collapsible step rows", () => {
    const bash: CommandSection = {
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
    const activities = activitiesFromLiveLogSections([
      bash,
      {
        index: 2,
        text: "Implementation",
        kind: "prompt",
        preview: "",
        lines: [],
        events: [{ kind: "note", text: "Let me start by exploring the codebase." }],
        status: "running",
        duration_ms: null,
        started_at: 2,
        collapsed: false,
      },
    ]);

    expect(activities[0]?.items).toEqual([
      expect.objectContaining({
        type: "content",
        kind: "assistant",
        text: "Let me start by exploring the codebase.",
      }),
    ]);
  });

  it("keeps a bash tool script apart from its command lines", () => {
    const script = "set -euo pipefail\n\ngit clone repo";
    const activities = activitiesFromLiveLogSections([
      {
        index: 2,
        text: "Implementation",
        kind: "prompt",
        preview: "",
        lines: [],
        events: [
          {
            kind: "tools",
            id: "tools-1",
            tools: [
              {
                id: "tool-1",
                kind: "bash",
                text: script,
                lines: ["Cloning into 'repo'...", "remote: Enumerating objects: 12, done."],
                status: "passed",
                duration_ms: 40,
              },
              {
                id: "tool-2",
                kind: "read",
                text: "README.md",
                lines: ["# Store"],
                status: "passed",
                duration_ms: 8,
              },
            ],
          },
        ],
        status: "passed",
        duration_ms: 40,
        started_at: 2,
        collapsed: true,
      },
    ]);

    expect(activities[0]?.items).toEqual([
      expect.objectContaining({
        type: "tool",
        kind: "bash",
        input: script,
        output: "Cloning into 'repo'...\nremote: Enumerating objects: 12, done.",
      }),
      expect.objectContaining({ type: "tool", kind: "read", input: "README.md", output: "" }),
    ]);
  });
});
