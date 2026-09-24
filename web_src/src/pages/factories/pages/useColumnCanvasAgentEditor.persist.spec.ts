import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { CanvasesCanvas } from "@/api-client";
import { dematerializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";

import { PR_FEEDBACK_DISCUSSION_AGENT_NODE_IDS } from "../lib/columnCanvasAgent";
import {
  agentCanvas,
  agentDraft,
  canvasWithPublishedNode,
  prFeedbackCanvas,
  stagingSaveDeps,
} from "./useColumnCanvasAgentEditor.testHelpers";
import { persistColumnAgent } from "./useColumnCanvasAgentEditor";

vi.mock("@/hooks/useCanvasData", () => ({
  canvasKeys: { detail: (organizationId: string, canvasId: string) => ["canvas", organizationId, canvasId] },
  useCanvas: () => ({ data: undefined, isPending: false }),
  useCanvasStaging: () => ({ refetch: vi.fn() }),
  useCommitCanvasStaging: () => ({ mutateAsync: vi.fn() }),
  useDiscardCanvasStaging: () => ({ mutateAsync: vi.fn() }),
  useUpdateCanvasVersion: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

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
      canvas: agentCanvas,
      agentNodeId: "implementation-agent",
      draft: agentDraft,
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
    expect(stageYaml.mock.calls[0][0].canvasYaml).toContain("includeVisualEvidence: true");
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
        canvas: agentCanvas,
        agentNodeId: "implementation-agent",
        draft: agentDraft,
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
      draft: agentDraft,
      stageYaml,
      commit,
      invalidate,
      ...stagingSaveDeps(),
    });

    const serialized = dematerializeCanvasSpec(stageYaml.mock.calls[0][0].canvasYaml);
    if (!serialized) {
      throw new Error("staged canvas yaml is not a canvas");
    }
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
    const stageYaml = vi.fn(async (input: { versionId: string; canvasYaml: string; replaceIfStale?: boolean }) => {
      order.push("stage");
      return input;
    });
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
      canvas: agentCanvas,
      agentNodeId: "implementation-agent",
      draft: agentDraft,
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
    expect(staged.canvasYaml).not.toContain("model: sonnet");
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
    const readStagedCanvas = vi.fn().mockResolvedValue({
      canvas: currentDraft,
      canvasYaml: "current-draft-yaml",
    });

    await persistColumnAgent({
      appId: "app-refund-implementer",
      canvas: agentCanvas,
      agentNodeId: "implementation-agent",
      draft: agentDraft,
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
    expect(stageYaml.mock.calls[1][0].expectedCanvasYaml).toBe("current-draft-yaml");
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
      canvas: agentCanvas,
      agentNodeId: "implementation-agent",
      draft: agentDraft,
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
      canvas: agentCanvas,
      agentNodeId: "implementation-agent",
      draft: agentDraft,
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
        canvas: agentCanvas,
        agentNodeId: "implementation-agent",
        draft: agentDraft,
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

  it("does not save when the current canvas has no agent node", async () => {
    const { showErrorToast, showSuccessToast } = await import("@/lib/toast");
    const canvasWithoutAgent: CanvasesCanvas = {
      metadata: { id: "app-refund-implementer", liveVersionId: "version-newer" },
      spec: {
        nodes: [{ id: "published-elsewhere", name: "Published elsewhere", type: "TYPE_ACTION", component: "http" }],
        edges: [],
      },
    };
    const stageYaml = vi.fn();
    const commit = vi.fn();

    await expect(
      persistColumnAgent({
        appId: "app-refund-implementer",
        canvas: agentCanvas,
        agentNodeId: "implementation-agent",
        draft: agentDraft,
        stageYaml,
        commit,
        invalidate: vi.fn(),
        readStagingSummary: vi.fn().mockResolvedValue({ hasStaging: true, stale: true }),
        refreshCanvas: vi.fn().mockResolvedValue(canvasWithoutAgent),
        readStagedCanvas: vi.fn(),
      }),
    ).rejects.toThrow("The agent is not on this canvas.");

    expect(stageYaml).not.toHaveBeenCalled();
    expect(commit).not.toHaveBeenCalled();
    expect(showSuccessToast).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalled();
  });

  it("does not report a save when the current draft removed the agent", async () => {
    const { showErrorToast, showSuccessToast } = await import("@/lib/toast");
    const draftWithoutAgent: CanvasesCanvas = {
      metadata: { id: "app-refund-implementer", liveVersionId: "version-live" },
      spec: {
        nodes: [{ id: "published-elsewhere", name: "Published elsewhere", type: "TYPE_ACTION", component: "http" }],
        edges: [],
      },
    };
    const stageYaml = vi.fn().mockRejectedValueOnce(new Error("current staging cannot be discarded"));
    const commit = vi.fn();

    await expect(
      persistColumnAgent({
        appId: "app-refund-implementer",
        canvas: agentCanvas,
        agentNodeId: "implementation-agent",
        draft: agentDraft,
        stageYaml,
        commit,
        invalidate: vi.fn(),
        readStagingSummary: vi.fn().mockResolvedValue({ hasStaging: true, stale: true }),
        refreshCanvas: vi.fn().mockResolvedValue(canvasWithPublishedNode("version-newer")),
        readStagedCanvas: vi.fn().mockResolvedValue({
          canvas: draftWithoutAgent,
          canvasYaml: "draft-without-agent",
        }),
      }),
    ).rejects.toThrow("The agent is not on this canvas.");

    expect(stageYaml).toHaveBeenCalledTimes(1);
    expect(commit).not.toHaveBeenCalled();
    expect(showSuccessToast).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalled();
  });

  it("retries a current draft save when the draft changes after the read", async () => {
    const { showSuccessToast } = await import("@/lib/toast");
    const firstDraft = canvasWithPublishedNode("version-live");
    const laterDraft: CanvasesCanvas = {
      ...firstDraft,
      spec: {
        nodes: [
          ...(firstDraft.spec?.nodes ?? []),
          { id: "later-edit", name: "Later edit", type: "TYPE_ACTION", component: "http" },
        ],
        edges: firstDraft.spec?.edges ?? [],
      },
    };
    const stageYaml = vi
      .fn()
      .mockRejectedValueOnce(new Error("current staging cannot be discarded"))
      .mockRejectedValueOnce(new Error("staged canvas changed"))
      .mockResolvedValueOnce({});
    const readStagedCanvas = vi
      .fn()
      .mockResolvedValueOnce({ canvas: firstDraft, canvasYaml: "yaml-1" })
      .mockResolvedValueOnce({ canvas: laterDraft, canvasYaml: "yaml-2" });

    await persistColumnAgent({
      appId: "app-refund-implementer",
      canvas: agentCanvas,
      agentNodeId: "implementation-agent",
      draft: agentDraft,
      stageYaml,
      commit: vi.fn().mockResolvedValue({}),
      invalidate: vi.fn().mockResolvedValue({}),
      readStagingSummary: vi.fn().mockResolvedValue({ hasStaging: true, stale: true }),
      refreshCanvas: vi.fn().mockResolvedValue(canvasWithPublishedNode("version-newer")),
      readStagedCanvas,
    });

    expect(readStagedCanvas).toHaveBeenCalledTimes(2);
    expect(stageYaml).toHaveBeenCalledTimes(3);
    expect(stageYaml.mock.calls[1][0].expectedCanvasYaml).toBe("yaml-1");
    expect(stageYaml.mock.calls[2][0].expectedCanvasYaml).toBe("yaml-2");
    expect(stageYaml.mock.calls[2][0].canvasYaml).toContain("later-edit");
    expect(stageYaml.mock.calls[2][0].canvasYaml).toContain("opus");
    expect(showSuccessToast).toHaveBeenCalledWith("Agent saved.");
  });

  it("does not overwrite a current draft that changes again", async () => {
    const { showErrorToast, showSuccessToast } = await import("@/lib/toast");
    const stageYaml = vi
      .fn()
      .mockRejectedValueOnce(new Error("current staging cannot be discarded"))
      .mockRejectedValue(new Error("staged canvas changed"));
    const readStagedCanvas = vi.fn().mockResolvedValue({
      canvas: canvasWithPublishedNode("version-live"),
      canvasYaml: "yaml-1",
    });

    await expect(
      persistColumnAgent({
        appId: "app-refund-implementer",
        canvas: agentCanvas,
        agentNodeId: "implementation-agent",
        draft: agentDraft,
        stageYaml,
        commit: vi.fn(),
        invalidate: vi.fn(),
        readStagingSummary: vi.fn().mockResolvedValue({ hasStaging: true, stale: true }),
        refreshCanvas: vi.fn().mockResolvedValue(canvasWithPublishedNode("version-newer")),
        readStagedCanvas,
      }),
    ).rejects.toThrow("The canvas changed. Save the agent again.");

    expect(readStagedCanvas).toHaveBeenCalledTimes(2);
    expect(stageYaml).toHaveBeenCalledTimes(3);
    expect(stageYaml.mock.calls[1][0].expectedCanvasYaml).toBe("yaml-1");
    expect(stageYaml.mock.calls[2][0].expectedCanvasYaml).toBe("yaml-1");
    expect(showSuccessToast).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalled();
  });
});
