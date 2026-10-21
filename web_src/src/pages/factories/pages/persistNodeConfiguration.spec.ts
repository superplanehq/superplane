import { describe, expect, it, vi } from "bun:test";
import { dematerializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";

import { agentCanvas, stagingSaveDeps } from "./useColumnCanvasAgentEditor.testHelpers";
import { persistNodeConfiguration } from "./useColumnCanvasAgentEditor";

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

describe("persistNodeConfiguration", () => {
  it("stages every changed node on the same canvas", async () => {
    const stageYaml = vi.fn().mockResolvedValue({});
    const commit = vi.fn().mockResolvedValue({});

    await persistNodeConfiguration({
      appId: "app-refund-implementer",
      canvas: agentCanvas,
      updates: [
        {
          nodeId: "onrun-implement",
          name: "On work order",
          configuration: { repository: "operately/website" },
        },
        {
          nodeId: "implementation-agent",
          name: "Implement From Task Description",
          configuration: { model: "opus" },
          concurrency: { max: 3 },
        },
      ],
      stageYaml,
      commit,
      invalidate: vi.fn(),
      notify: false,
      ...stagingSaveDeps(),
    });

    const canvasYaml = stageYaml.mock.calls[0][0].canvasYaml as string;
    const spec = dematerializeCanvasSpec(canvasYaml);
    const trigger = spec?.nodes?.find((node) => node.id === "onrun-implement");
    const agent = spec?.nodes?.find((node) => node.id === "implementation-agent");
    expect(stageYaml).toHaveBeenCalledTimes(1);
    expect(trigger?.name).toBe("On work order");
    expect(trigger?.configuration).toMatchObject({ repository: "operately/website" });
    expect(agent?.configuration).toMatchObject({ model: "opus" });
    expect(agent?.concurrency).toMatchObject({ max: 3 });
    expect(commit).toHaveBeenCalledTimes(1);
  });
});
