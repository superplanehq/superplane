import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { CanvasesCanvas } from "@/api-client";
import { FEATURE_FACTORY_CREATE_WITH_AGENT } from "@/lib/experimentalFeatures";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";

import type { CanvasSpecNode } from "../lib/columnCanvasAgent";
import { persistColumnAgent, useColumnCanvasAgentEditor } from "./useColumnCanvasAgentEditor";
import type { PlanningReviewDraft } from "./planningReviewMockup";

const hookState = vi.hoisted(() => ({
  canvas: { current: undefined as CanvasesCanvas | undefined },
  feature: {
    current: {
      has: (_featureId: string): boolean => false,
      enabledExperimentalFeatures: [] as string[],
      isLoading: false,
    },
  },
}));

vi.mock("@/hooks/useCanvasData", () => ({
  canvasKeys: { detail: (organizationId: string, canvasId: string) => ["canvas", organizationId, canvasId] },
  useCanvas: () => ({ data: hookState.canvas.current, isPending: false }),
  useCommitCanvasStaging: () => ({ mutateAsync: vi.fn() }),
  useUpdateCanvasVersion: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => hookState.feature.current,
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
  spec: { nodes: [implementerNode], edges: [] },
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

const draft: PlanningReviewDraft = {
  title: "Implement From Task Description",
  components: [
    {
      id: "implementation-agent",
      title: "Implement From Task Description",
      description: "",
      expanded: true,
      configuration: { model: "opus", steps: [{ name: "Clone Repo", type: "bash", command: "git clone --depth 1" }] },
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
    hookState.feature.current = {
      has: (_featureId: string): boolean => false,
      enabledExperimentalFeatures: [],
      isLoading: false,
    };
  });

  it("selects the refinement agent when Task Refinement is enabled", () => {
    hookState.feature.current.has = (featureId: string) => featureId === FEATURE_FACTORY_CREATE_WITH_AGENT;

    const { result } = renderHook(() => useColumnCanvasAgentEditor("organization-1", "backlog"), {
      wrapper: createWrapper(),
    });

    expect(result.current.agentNode?.id).toBe("refine-task");
    expect(result.current.draft?.title).toBe("Refine Task");
  });

  it("selects the first agent when Task Refinement is disabled", () => {
    const { result } = renderHook(() => useColumnCanvasAgentEditor("organization-1", "backlog"), {
      wrapper: createWrapper(),
    });

    expect(result.current.agentNode?.id).toBe("analyze");
  });

  it("stays loading while Task Refinement access resolves", () => {
    hookState.feature.current.isLoading = true;

    const { result } = renderHook(() => useColumnCanvasAgentEditor("organization-1", "backlog"), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
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
});
