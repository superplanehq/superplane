import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { canvasesGetCanvasStaging } from "@/api-client";
import { useDraftActions } from "./useDraftActions";
import type { AgentMessage } from "@/components/CanvasToolSidebar/types";

vi.mock("@/api-client", async () => {
  const actual = await vi.importActual<typeof import("@/api-client")>("@/api-client");
  return {
    ...actual,
    canvasesGetCanvasStaging: vi.fn(),
  };
});

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

function stagingActionsContent(canvasId: string): string {
  return ["Staging ready", "", ":::staging-actions", `canvasId: ${canvasId}`, "message: Staging ready", ":::"].join(
    "\n",
  );
}

function mockCanvasStaging(hasStaging: boolean) {
  vi.mocked(canvasesGetCanvasStaging).mockResolvedValue({
    data: { staging: { hasStaging } },
  } as never);
}

describe("useDraftActions", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("keeps the latest staging action visible after a follow-up user message", async () => {
    mockCanvasStaging(true);

    const { result } = renderHook(() =>
      useDraftActions({
        messages: [
          message({ id: "assistant-1", role: "assistant", content: stagingActionsContent("canvas-1") }),
          message({ id: "user-1", role: "user", content: "Make one more change" }),
        ],
        canvasId: "canvas-1",
        organizationId: "org-1",
      }),
    );

    await waitFor(() => expect(result.current.latestDraft?.canvasId).toBe("canvas-1"));
    expect(canvasesGetCanvasStaging).toHaveBeenCalled();
  });

  it("hides staging actions when staging no longer exists", async () => {
    mockCanvasStaging(false);

    const { result } = renderHook(() =>
      useDraftActions({
        messages: [message({ id: "assistant-1", role: "assistant", content: stagingActionsContent("canvas-1") })],
        canvasId: "canvas-1",
        organizationId: "org-1",
      }),
    );

    await waitFor(() => expect(canvasesGetCanvasStaging).toHaveBeenCalled());
    expect(result.current.latestDraft).toBeNull();
  });
});
