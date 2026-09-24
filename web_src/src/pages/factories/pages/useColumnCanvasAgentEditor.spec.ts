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
  useCanvasStaging: () => ({ refetch: vi.fn() }),
  useCommitCanvasStaging: () => ({ mutateAsync: vi.fn() }),
  useDiscardCanvasStaging: () => ({ mutateAsync: vi.fn() }),
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

function stagingSaveDeps(summary?: { hasStaging?: boolean; stale?: boolean }) {
  return {
    readStagingSummary: vi.fn().mockResolvedValue(summary),
    refreshCanvas: vi.fn(async () => {
      throw new Error("live canvas refresh should not run");
    }),
    readStagedCanvas: vi.fn(async () => {
      throw new Error("staged canvas read should not run");
    }),
  };
}

function canvasWithPublishedNode(versionId: string): CanvasesCanvas {
  return {
    metadata: { id: "app-refund-implementer", liveVersionId: versionId },
    spec: {
      nodes: [
        ...(canvas.spec?.nodes ?? []),
        { id: "published-elsewhere", name: "Published elsewhere", type: "TYPE_ACTION", component: "http" },
      ],
      edges: [{ sourceId: "onrun-implement", targetId: "published-elsewhere" }],
    },
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
  beforeEach(async () => {
    const toast = await import("@/lib/toast");
    vi.mocked(toast.showErrorToast).mockClear();
    vi.mocked(toast.showSuccessToast).mockClear();
  });

  it("stages the patched canvas yaml and commits", async () => {
    const stageYaml = vi.fn().mockResolvedValue({});
    const commit = vi.fn().mockResolvedValue({});
    const invalidate = vi.fn().mockResolvedValue({});
    const staging = stagingSaveDeps();

    await persistColumnAgent({
      appId: "app-refund-implementer",
      canvas,
      agentNodeId: "implementation-agent",
      draft,
      stageYaml,
      commit,
      invalidate,
      ...staging,
    });

    expect(stageYaml).toHaveBeenCalledWith({
      versionId: "version-live",
      canvasYaml: expect.stringContaining("opus"),
    });
    expect(stageYaml.mock.calls[0][0].canvasYaml).toContain("git clone --depth 1");
    expect(stageYaml.mock.calls[0][0].canvasYaml).toContain('"includeVisualEvidence":true');
    expect(commit).toHaveBeenCalledWith("Update agent");
    expect(invalidate).toHaveBeenCalled();
    expect(staging.refreshCanvas).not.toHaveBeenCalled();
    expect(staging.readStagedCanvas).not.toHaveBeenCalled();
  });

  it("does not commit when staging fails", async () => {
    const { showErrorToast } = await import("@/lib/toast");
    const stageYaml = vi.fn().mockRejectedValue(new Error("stage failed"));
    const commit = vi.fn();
    const staging = stagingSaveDeps({ hasStaging: true, stale: false });

    await expect(
      persistColumnAgent({
        appId: "app-refund-implementer",
        canvas,
        agentNodeId: "implementation-agent",
        draft,
        stageYaml,
        commit,
        invalidate: vi.fn(),
        ...staging,
      }),
    ).rejects.toThrow("stage failed");

    expect(commit).not.toHaveBeenCalled();
    expect(staging.refreshCanvas).not.toHaveBeenCalled();
    expect(staging.readStagedCanvas).not.toHaveBeenCalled();
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
      ...stagingSaveDeps(),
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

  it("replaces a stale draft with the agent edit on the current live canvas", async () => {
    const { showSuccessToast } = await import("@/lib/toast");
    const order: string[] = [];
    const newerCanvas = canvasWithPublishedNode("version-newer");
    const stageYaml = vi.fn(
      async (input: { versionId: string; canvasYaml: string; replaceIfStale?: boolean }) => {
        order.push("stage");
        return input;
      },
    );
    const commit = vi.fn(async () => {
      order.push("commit");
    });
    const refreshCanvas = vi.fn(async () => {
      order.push("refresh");
      return newerCanvas;
    });
    const readStagedCanvas = vi.fn(async () => {
      throw new Error("staged canvas read should not run");
    });

    await persistColumnAgent({
      appId: "app-refund-implementer",
      canvas,
      agentNodeId: "implementation-agent",
      draft,
      stageYaml,
      commit,
      invalidate: vi.fn().mockResolvedValue({}),
      readStagingSummary: vi.fn().mockResolvedValue({
        hasStaging: true,
        stale: true,
        stagedPaths: ["canvas.yaml"],
      }),
      refreshCanvas,
      readStagedCanvas,
    });

    expect(order).toEqual(["refresh", "stage", "commit"]);
    const staged = stageYaml.mock.calls[0][0];
    expect(staged.versionId).toBe("version-newer");
    expect(staged.replaceIfStale).toBe(true);
    expect(staged.canvasYaml).toContain("published-elsewhere");
    expect(staged.canvasYaml).toContain("opus");
    expect(staged.canvasYaml).not.toContain('"model":"sonnet"');
    expect(commit).toHaveBeenCalledWith("Update agent");
    expect(readStagedCanvas).not.toHaveBeenCalled();
    expect(showSuccessToast).toHaveBeenCalledWith(
      "Agent saved. Earlier canvas edits were discarded because the live canvas changed.",
    );
  });

  it("keeps a current draft that replaces a stale draft before save", async () => {
    const { showSuccessToast } = await import("@/lib/toast");
    const currentDraft = canvasWithPublishedNode("version-live");
    const stageYaml = vi
      .fn()
      .mockRejectedValueOnce(new Error("current staging cannot be discarded"))
      .mockResolvedValueOnce({});
    const readStagedCanvas = vi.fn().mockResolvedValue(currentDraft);

    await persistColumnAgent({
      appId: "app-refund-implementer",
      canvas,
      agentNodeId: "implementation-agent",
      draft,
      stageYaml,
      commit: vi.fn().mockResolvedValue({}),
      invalidate: vi.fn().mockResolvedValue({}),
      readStagingSummary: vi.fn().mockResolvedValue({
        hasStaging: true,
        stale: true,
        stagedPaths: ["canvas.yaml"],
      }),
      refreshCanvas: vi.fn().mockResolvedValue(canvasWithPublishedNode("version-newer")),
      readStagedCanvas,
    });

    expect(readStagedCanvas).toHaveBeenCalledTimes(1);
    expect(stageYaml).toHaveBeenCalledTimes(2);
    expect(stageYaml.mock.calls[0][0].replaceIfStale).toBe(true);
    expect(stageYaml.mock.calls[0][0].canvasYaml).toContain("published-elsewhere");
    expect(stageYaml.mock.calls[1][0].replaceIfStale).toBeUndefined();
    expect(stageYaml.mock.calls[1][0].versionId).toBe("version-live");
    expect(stageYaml.mock.calls[1][0].canvasYaml).toContain("published-elsewhere");
    expect(stageYaml.mock.calls[1][0].canvasYaml).toContain("opus");
    expect(showSuccessToast).toHaveBeenCalledWith("Agent saved.");
  });

  it("does not discard a draft that still matches the live canvas", async () => {
    const { showSuccessToast } = await import("@/lib/toast");
    const stageYaml = vi.fn().mockResolvedValue({});
    const commit = vi.fn().mockResolvedValue({});
    const staging = stagingSaveDeps({ hasStaging: true, stale: false });

    await persistColumnAgent({
      appId: "app-refund-implementer",
      canvas,
      agentNodeId: "implementation-agent",
      draft,
      stageYaml,
      commit,
      invalidate: vi.fn().mockResolvedValue({}),
      ...staging,
    });

    expect(staging.refreshCanvas).not.toHaveBeenCalled();
    expect(staging.readStagedCanvas).not.toHaveBeenCalled();
    expect(stageYaml).toHaveBeenCalledTimes(1);
    expect(stageYaml).toHaveBeenCalledWith({
      versionId: "version-live",
      canvasYaml: expect.stringContaining("opus"),
    });
    expect(commit).toHaveBeenCalledWith("Update agent");
    expect(showSuccessToast).toHaveBeenCalledWith("Agent saved.");
  });

  it("retries a stale stage on the current live canvas", async () => {
    const { showSuccessToast } = await import("@/lib/toast");
    const newerCanvas = canvasWithPublishedNode("version-newer");
    const stageYaml = vi
      .fn()
      .mockRejectedValueOnce(new Error("stale staging cannot be updated"))
      .mockResolvedValueOnce({});
    const commit = vi.fn().mockResolvedValue({});
    const refreshCanvas = vi.fn().mockResolvedValue(newerCanvas);
    const readStagedCanvas = vi.fn(async () => {
      throw new Error("staged canvas read should not run");
    });

    await persistColumnAgent({
      appId: "app-refund-implementer",
      canvas,
      agentNodeId: "implementation-agent",
      draft,
      stageYaml,
      commit,
      invalidate: vi.fn().mockResolvedValue({}),
      ...stagingSaveDeps({ hasStaging: true, stale: false }),
      refreshCanvas,
      readStagedCanvas,
    });

    expect(readStagedCanvas).not.toHaveBeenCalled();
    expect(refreshCanvas).toHaveBeenCalledTimes(1);
    expect(stageYaml).toHaveBeenCalledTimes(2);
    expect(stageYaml.mock.calls[0][0].versionId).toBe("version-live");
    expect(stageYaml.mock.calls[0][0].canvasYaml).not.toContain("published-elsewhere");
    expect(stageYaml.mock.calls[1][0]).toEqual({
      versionId: "version-newer",
      canvasYaml: expect.stringContaining("published-elsewhere"),
      replaceIfStale: true,
    });
    expect(stageYaml.mock.calls[1][0].canvasYaml).toContain("opus");
    expect(commit).toHaveBeenCalledTimes(1);
    expect(showSuccessToast).toHaveBeenCalledWith(
      "Agent saved. Earlier canvas edits were discarded because the live canvas changed.",
    );
  });

  it("does not retry a stale stage more than once", async () => {
    const { showErrorToast, showSuccessToast } = await import("@/lib/toast");
    const stageYaml = vi.fn().mockRejectedValue(new Error("stale staging cannot be updated"));
    const commit = vi.fn();
    const refreshCanvas = vi.fn().mockResolvedValue(canvasWithPublishedNode("version-newer"));
    const readStagedCanvas = vi.fn();

    await expect(
      persistColumnAgent({
        appId: "app-refund-implementer",
        canvas,
        agentNodeId: "implementation-agent",
        draft,
        stageYaml,
        commit,
        invalidate: vi.fn(),
        ...stagingSaveDeps({ hasStaging: false }),
        refreshCanvas,
        readStagedCanvas,
      }),
    ).rejects.toThrow("stale staging cannot be updated");

    expect(readStagedCanvas).not.toHaveBeenCalled();
    expect(refreshCanvas).toHaveBeenCalledTimes(1);
    expect(stageYaml).toHaveBeenCalledTimes(2);
    expect(stageYaml.mock.calls[1][0].replaceIfStale).toBe(true);
    expect(stageYaml.mock.calls[1][0].versionId).toBe("version-newer");
    expect(commit).not.toHaveBeenCalled();
    expect(showSuccessToast).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalled();
  });
});
