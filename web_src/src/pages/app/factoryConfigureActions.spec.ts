import { describe, expect, it, vi } from "vitest";

import { runFactoryConfigureSave, withCanvasMetadataName } from "./factoryConfigureActions";

describe("withCanvasMetadataName", () => {
  it("overlays metadata.name when a new name is provided", () => {
    const next = withCanvasMetadataName(
      {
        metadata: { id: "c1", name: "Old" },
        spec: { nodes: [], edges: [] },
      },
      "Renamed",
    );
    expect(next.metadata?.name).toBe("Renamed");
    expect(next.spec).toEqual({ nodes: [], edges: [] });
  });

  it("returns the same workflow when the name is unchanged", () => {
    const workflow = {
      metadata: { id: "c1", name: "Same" },
      spec: { nodes: [], edges: [] },
    };
    expect(withCanvasMetadataName(workflow, "Same")).toBe(workflow);
  });
});

describe("runFactoryConfigureSave", () => {
  it("materializes canvas.yaml with the overlaid rename", async () => {
    const updateCanvasVersionMutation = {
      mutateAsync: vi.fn().mockResolvedValue({}),
    };
    const handleCommitStaging = vi.fn().mockResolvedValue(true);
    const onDone = vi.fn();
    const setSavePending = vi.fn();
    const setDraftCanvasSpec = vi.fn();
    const setLastSavedWorkflowSnapshot = vi.fn();
    const draftCanvasSpecsRef = { current: new Map() };
    const activeCanvasVersionIdRef = { current: "ver-1" };

    await runFactoryConfigureSave({
      canStageCanvasVersion: true,
      activeCanvasVersionIdRef,
      activeCanvasVersionId: "ver-1",
      editSessionActive: true,
      setEditSessionActive: vi.fn(),
      getCurrentWorkflowSnapshot: () => ({
        metadata: { id: "c1", name: "Old" },
        spec: { nodes: [{ id: "n1", name: "Node", component: "noop", type: "TYPE_ACTION" }], edges: [] },
      }),
      setSavePending,
      updateCanvasVersionMutation,
      draftCanvasSpecsRef,
      setDraftCanvasSpec,
      setLastSavedWorkflowSnapshot,
      handleCommitStaging,
      canvasName: "Renamed",
      onDone,
    });

    expect(updateCanvasVersionMutation.mutateAsync).toHaveBeenCalledTimes(1);
    const stagedYaml = updateCanvasVersionMutation.mutateAsync.mock.calls[0]?.[0]?.canvasYaml as string;
    expect(stagedYaml).toContain("name: Renamed");
    expect(stagedYaml).not.toContain("name: Old");
    expect(handleCommitStaging).toHaveBeenCalled();
    expect(onDone).toHaveBeenCalled();
    expect(setLastSavedWorkflowSnapshot).toHaveBeenCalledWith(
      expect.objectContaining({ metadata: expect.objectContaining({ name: "Renamed" }) }),
    );
  });
});
