import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { CanvasesCanvas } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import { PR_FEEDBACK_DISCUSSION_AGENT_NODE_IDS } from "../lib/columnCanvasAgent";
import { agentCanvas, backlogCanvas, prFeedbackCanvas } from "./useColumnCanvasAgentEditor.testHelpers";
import { useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";

const hookState = vi.hoisted(() => ({
  canvas: { current: undefined as CanvasesCanvas | undefined },
}));

vi.mock("@/hooks/useCanvasData", () => ({
  canvasKeys: { detail: (organizationId: string, canvasId: string) => ["canvas", organizationId, canvasId] },
  useCanvas: () => ({ data: hookState.canvas.current, isPending: false }),
  useCanvasStaging: () => ({ refetch: vi.fn() }),
  useCommitCanvasStaging: () => ({ mutateAsync: vi.fn() }),
  useDiscardCanvasStaging: () => ({ mutateAsync: vi.fn() }),
  useUpdateCanvasVersion: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useColumnCanvasAgentEditor", () => {
  beforeEach(() => {
    hookState.canvas.current = backlogCanvas;
  });

  it("selects the Refine Task agent on the Backlog canvas", () => {
    const { result } = renderHook(() => useColumnCanvasAgentEditor("organization-1", "backlog"), {
      wrapper: createWrapper(),
    });

    expect(result.current.agentNode?.id).toBe("refine-task");
    expect(result.current.draft?.title).toBe("Refine Task");
  });

  it("shows the visual evidence setting only for line implementation", () => {
    hookState.canvas.current = agentCanvas;

    const { result, rerender } = renderHook(({ appId }) => useColumnCanvasAgentEditor("organization-1", appId), {
      initialProps: { appId: "app-refund-implementer" },
      wrapper: createWrapper(),
    });

    expect(result.current.showVisualEvidenceSetting).toBe(true);
    hookState.canvas.current = backlogCanvas;
    rerender({ appId: "backlog" });
    expect(result.current.showVisualEvidenceSetting).toBe(false);
  });

  it("shows visual evidence for discussion feedback when requested", () => {
    hookState.canvas.current = prFeedbackCanvas;

    const { result } = renderHook(
      () =>
        useColumnCanvasAgentEditor("organization-1", "pr-feedback", {
          showVisualEvidenceSetting: true,
          synchronizedAgentNodeIds: PR_FEEDBACK_DISCUSSION_AGENT_NODE_IDS,
        }),
      { wrapper: createWrapper() },
    );

    expect(result.current.showVisualEvidenceSetting).toBe(true);
    expect(result.current.agentNodeIds).toEqual([
      "address-pr-feedback",
      "address-pr-review-feedback",
      "address-pr-review-reply-feedback",
    ]);
  });
});
