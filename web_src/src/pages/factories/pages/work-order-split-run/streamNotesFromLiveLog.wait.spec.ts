import { describe, expect, it } from "bun:test";

import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { notesForLiveStream } from "./streamNotesFromLiveLog";

function commandSection(overrides: Partial<CommandSection> & Pick<CommandSection, "text">): CommandSection {
  return {
    index: 8,
    kind: "bash",
    preview: "Wait for the next user message",
    lines: [],
    events: [],
    status: "failed",
    duration_ms: 1200,
    started_at: 1,
    collapsed: true,
    ...overrides,
  };
}

describe("notesForLiveStream analysis wait", () => {
  it("marks a failed wait canceled when Start passes the analysis", () => {
    const notes = notesForLiveStream({
      nodeId: "agent",
      sections: [commandSection({ text: "Wait for the next message" })],
      error: null,
      isStreaming: false,
      nodeStatus: "cancelled",
      analysisStatus: "passed",
    });

    expect(notes?.[0]).toEqual(
      expect.objectContaining({
        componentName: "Wait for the next message",
        detail: "Wait for the next user message",
        status: "cancelled",
      }),
    );
  });

  it("marks a failed wait canceled when the analysis outcome is canceled", () => {
    const notes = notesForLiveStream({
      nodeId: "agent",
      sections: [commandSection({ text: "Wait for the next message" })],
      error: null,
      isStreaming: false,
      nodeStatus: "failed",
      analysisStatus: "cancelled",
    });

    expect(notes?.[0]?.status).toBe("cancelled");
  });

  it("keeps a failed wait failed when an explicit stop leaves the analysis failed", () => {
    const notes = notesForLiveStream({
      nodeId: "agent",
      sections: [commandSection({ text: "Wait for the next message" })],
      error: null,
      isStreaming: false,
      nodeStatus: "cancelled",
      analysisStatus: "failed",
    });

    expect(notes?.[0]?.status).toBe("failed");
  });

  it("keeps a different failed command failed when the analysis run passed", () => {
    const notes = notesForLiveStream({
      nodeId: "agent",
      sections: [
        commandSection({
          index: 1,
          text: "Set Up Git User",
          preview: 'echo "Using superplaneagent@superplane.com"',
          lines: ["Using superplaneagent@superplane.com"],
          duration_ms: 20,
        }),
      ],
      error: null,
      isStreaming: false,
      nodeStatus: "passed",
      analysisStatus: "passed",
    });

    expect(notes?.[0]).toEqual(
      expect.objectContaining({
        componentName: "Set Up Git User",
        status: "failed",
      }),
    );
  });
});
