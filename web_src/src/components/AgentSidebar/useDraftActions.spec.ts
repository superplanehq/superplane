import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useDraftActions } from "./useDraftActions";
import type { AgentMessage } from "@/components/CanvasToolSidebar/types";

function message(overrides: Partial<AgentMessage>): AgentMessage {
  return {
    id: "message-1",
    role: "assistant",
    content: "",
    toolName: "",
    toolCallId: "",
    toolStatus: "",
    createdAt: null,
    ...overrides,
  };
}

function draftActionsContent(versionId: string): string {
  return ["Draft ready", "", ":::draft-actions", `versionId: ${versionId}`, "message: Draft ready", ":::"].join("\n");
}

function mockDraftVersion(state = "STATE_DRAFT") {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      return new Response(JSON.stringify({ version: { metadata: { state } } }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }),
  );
}

describe("useDraftActions", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps the latest draft action visible after a follow-up user message", async () => {
    mockDraftVersion();

    const { result } = renderHook(() =>
      useDraftActions({
        messages: [
          message({ id: "assistant-1", role: "assistant", content: draftActionsContent("draft-1") }),
          message({ id: "user-1", role: "user", content: "Make one more change" }),
        ],
        canvasId: "canvas-1",
        organizationId: "org-1",
      }),
    );

    await waitFor(() => expect(result.current.latestDraft?.versionId).toBe("draft-1"));
  });

  it("hides draft actions when the version is no longer a draft", async () => {
    mockDraftVersion("STATE_PUBLISHED");

    const { result } = renderHook(() =>
      useDraftActions({
        messages: [message({ id: "assistant-1", role: "assistant", content: draftActionsContent("draft-1") })],
        canvasId: "canvas-1",
        organizationId: "org-1",
      }),
    );

    await waitFor(() => expect(fetch).toHaveBeenCalled());
    expect(result.current.latestDraft).toBeNull();
  });
});
