import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvas } from "@/api-client";

import type { CanvasSpecNode } from "../lib/columnCanvasAgent";
import { persistColumnAgent } from "./useColumnCanvasAgentEditor";
import type { PlanningReviewDraft } from "./planningReviewMockup";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

vi.mock("@/pages/app/lib/workflow-spec-files", () => ({
  materializeCanvasSpec: (canvas: CanvasesCanvas) => JSON.stringify(canvas.spec),
}));

const implementerNode: CanvasSpecNode = {
  id: "implementation-agent",
  name: "Agent - Implement from order description",
  type: "TYPE_ACTION",
  component: "runnerClaudeCode",
  concurrency: { max: 5 },
  configuration: { model: "sonnet", steps: [{ name: "Clone Repo", type: "bash", command: "git clone" }] },
};

const canvas: CanvasesCanvas = {
  metadata: { id: "app-refund-implementer", liveVersionId: "version-live" },
  spec: { nodes: [implementerNode], edges: [] },
};

const draft: PlanningReviewDraft = {
  title: "Agent - Implement from order description",
  components: [
    {
      id: "implementation-agent",
      title: "Agent - Implement from order description",
      description: "",
      expanded: true,
      configuration: { model: "opus", steps: [{ name: "Clone Repo", type: "bash", command: "git clone --depth 1" }] },
      concurrency: { max: "5", key: "" },
    },
  ],
};

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
