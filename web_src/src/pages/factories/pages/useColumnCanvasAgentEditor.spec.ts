import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { CanvasesCanvas } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import { PR_FEEDBACK_DISCUSSION_AGENT_NODE_IDS, type CanvasSpecNode } from "../lib/columnCanvasAgent";
import { persistColumnAgent, useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import type { PlanningReviewDraft } from "./planningReviewMockup";

const hookState = vi.hoisted(() => ({
  canvas: { current: undefined as CanvasesCanvas | undefined },
}));

vi.mock("@/hooks/useCanvasData", () => ({
  canvasKeys: { detail: (organizationId: string, canvasId: string) => ["canvas", organizationId, canvasId] },
  useCanvas: () => ({ data: hookState.canvas.current, isPending: false }),
  useCommitCanvasStaging: () => ({ mutateAsync: vi.fn() }),
  useUpdateCanvasVersion: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

vi.mock("@/pages/app/lib/workflow-spec-files", () => ({
  materializeCanvasSpec: (canvas: CanvasesCanvas) => JSON.stringify(canvas.spec),
}));

const implementerNode: CanvasSpecNode = {
  id: "implementation-agent",
  name: "Implement From Task Description",
  type: "TYPE_ACTION",
  component: "runnerClaudeCode",
  concurrency: { max: 5 },
  configuration: { model: "sonnet", steps: [{ name: "Clone Repo", type: "bash", command: "git clone" }] },
};

const canvas: CanvasesCanvas = {
  metadata: { id: "app-refund-implementer", liveVersionId: "version-live" },
  spec: {
    nodes: [{ id: "onrun-implement", name: "On run", type: "TYPE_TRIGGER", component: "onWorkOrder" }, implementerNode],
    edges: [],
  },
};

const backlogCanvas: CanvasesCanvas = {
  metadata: { id: "backlog", liveVersionId: "version-live" },
  spec: {
    nodes: [
      { ...implementerNode, id: "analyze", name: "Analyze intake" },
      { ...implementerNode, id: "refine-task", name: "Refine Task" },
    ],
    edges: [],
  },
};

const prFeedbackCanvas: CanvasesCanvas = {
  metadata: { id: "pr-feedback", liveVersionId: "version-live" },
  spec: {
    nodes: [
      { ...implementerNode, id: "address-pr-feedback", name: "Address PR feedback" },
      { ...implementerNode, id: "address-pr-review-feedback", name: "Address PR feedback" },
      { ...implementerNode, id: "address-pr-review-reply-feedback", name: "Address PR feedback" },
      { ...implementerNode, id: "custom-runner", name: "Custom agent" },
    ],
    edges: [],
  },
};

const draft: PlanningReviewDraft = {
  title: "Implement From Task Description",
  components: [
    {
      id: "implementation-agent",
      title: "Implement From Task Description",
      description: "",
      expanded: true,
      configuration: {
        model: "opus",
        includeVisualEvidence: true,
        steps: [{ name: "Clone Repo", type: "bash", command: "git clone --depth 1" }],
      },
      concurrency: { max: "5", key: "" },
    },
  ],
};

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
    hookState.canvas.current = canvas;

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

describe("persistColumnAgent", () => {
  it("stages the patched canvas yaml and commits", async () => {
    const stageYaml = vi.fn().mockResolvedValue({});
    const commit = vi.fn().mockResolvedValue({});
    const invalidate = vi.fn().mockResolvedValue({});

    await persistColumnAgent({
      appId: "app-refund-implementer",
      canvas,
      agentNodeId: "implementation-agent",
      draft,
      stageYaml,
      commit,
      invalidate,
    });

    expect(stageYaml).toHaveBeenCalledWith({
      versionId: "version-live",
      canvasYaml: expect.stringContaining("opus"),
    });
    expect(stageYaml.mock.calls[0][0].canvasYaml).toContain("git clone --depth 1");
    expect(stageYaml.mock.calls[0][0].canvasYaml).toContain('"includeVisualEvidence":true');
    expect(commit).toHaveBeenCalledWith("Update agent");
    expect(invalidate).toHaveBeenCalled();
  });

  it("does not commit when staging fails", async () => {
    const { showErrorToast } = await import("@/lib/toast");
    const stageYaml = vi.fn().mockRejectedValue(new Error("stage failed"));
    const commit = vi.fn();

    await expect(
      persistColumnAgent({
        appId: "app-refund-implementer",
        canvas,
        agentNodeId: "implementation-agent",
        draft,
        stageYaml,
        commit,
        invalidate: vi.fn(),
      }),
    ).rejects.toThrow("stage failed");

    expect(commit).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalled();
  });

  it("updates every synchronized discussion runner", async () => {
    const stageYaml = vi.fn().mockResolvedValue({});
    const commit = vi.fn().mockResolvedValue({});
    const invalidate = vi.fn().mockResolvedValue({});

    await persistColumnAgent({
      appId: "pr-feedback",
      canvas: prFeedbackCanvas,
      agentNodeIds: ["address-pr-feedback", "address-pr-review-feedback", "address-pr-review-reply-feedback"],
      draft,
      stageYaml,
      commit,
      invalidate,
    });

    const serialized = JSON.parse(stageYaml.mock.calls[0][0].canvasYaml) as NonNullable<CanvasesCanvas["spec"]>;
    const synchronizedIds = new Set<string>(PR_FEEDBACK_DISCUSSION_AGENT_NODE_IDS);
    const runners = serialized.nodes?.filter((node) => node.id && synchronizedIds.has(node.id)) ?? [];
    expect(runners).toHaveLength(3);
    for (const runner of runners) {
      expect(runner.configuration?.includeVisualEvidence).toBe(true);
      expect(runner.configuration?.model).toBe("opus");
    }
    const customRunner = serialized.nodes?.find((node) => node.id === "custom-runner");
    expect(customRunner?.configuration?.includeVisualEvidence).toBeUndefined();
    expect(customRunner?.configuration?.model).toBe("sonnet");
  });
});
