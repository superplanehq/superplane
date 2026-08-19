import { describe, expect, it, vi } from "vitest";

import { runFactoryConfigureSave, stagingSummaryHasChanges, withCanvasMetadataName } from "./factoryConfigureActions";

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

describe("stagingSummaryHasChanges", () => {
  it("detects staged paths and hasStaging", () => {
    expect(stagingSummaryHasChanges(undefined)).toBe(false);
    expect(stagingSummaryHasChanges({ hasStaging: false, stagedPaths: [] })).toBe(false);
    expect(stagingSummaryHasChanges({ hasStaging: true, stagedPaths: [] })).toBe(true);
    expect(stagingSummaryHasChanges({ hasStaging: false, stagedPaths: ["canvas.yaml"] })).toBe(true);
  });
});

describe("runFactoryConfigureSave", () => {
  function baseDeps(overrides: Partial<Parameters<typeof runFactoryConfigureSave>[0]> = {}) {
    return {
      canStageCanvasVersion: true,
      activeCanvasVersionIdRef: { current: "ver-1" },
      activeCanvasVersionId: "ver-1",
      editSessionActive: true,
      setEditSessionActive: vi.fn(),
      getCurrentWorkflowSnapshot: () => ({
        metadata: { id: "c1", name: "Old" },
        spec: { nodes: [{ id: "n1", name: "Node", component: "noop", type: "TYPE_ACTION" }], edges: [] },
      }),
      setSavePending: vi.fn(),
      updateCanvasVersionMutation: {
        mutateAsync: vi.fn().mockResolvedValue({
          data: { stagingSummary: { hasStaging: true, stagedPaths: ["canvas.yaml"] } },
        }),
      },
      draftCanvasSpecsRef: { current: new Map() },
      setDraftCanvasSpec: vi.fn(),
      setLastSavedWorkflowSnapshot: vi.fn(),
      handleCommitStaging: vi.fn().mockResolvedValue(true),
      onDone: vi.fn(),
      onAfterCommit: vi.fn(),
      ...overrides,
    } satisfies Parameters<typeof runFactoryConfigureSave>[0];
  }

  it("materializes canvas.yaml with the overlaid rename", async () => {
    const updateCanvasVersionMutation = {
      mutateAsync: vi.fn().mockResolvedValue({
        data: { stagingSummary: { hasStaging: true, stagedPaths: ["canvas.yaml"] } },
      }),
    };
    const deps = baseDeps({ updateCanvasVersionMutation, canvasName: "Renamed" });

    await runFactoryConfigureSave(deps);

    expect(updateCanvasVersionMutation.mutateAsync).toHaveBeenCalledTimes(1);
    const stagedYaml = updateCanvasVersionMutation.mutateAsync.mock.calls[0]?.[0]?.canvasYaml as string;
    expect(stagedYaml).toContain("name: Renamed");
    expect(stagedYaml).not.toContain("name: Old");
    expect(deps.handleCommitStaging).toHaveBeenCalled();
    expect(deps.onAfterCommit).toHaveBeenCalled();
    expect(deps.onDone).toHaveBeenCalled();
    expect(deps.setLastSavedWorkflowSnapshot).toHaveBeenCalledWith(
      expect.objectContaining({ metadata: expect.objectContaining({ name: "Renamed" }) }),
    );
  });

  it("skips commit and finishes when staging is empty", async () => {
    const updateCanvasVersionMutation = {
      mutateAsync: vi.fn().mockResolvedValue({
        data: { stagingSummary: { hasStaging: false, stagedPaths: [] } },
      }),
    };
    const deps = baseDeps({ updateCanvasVersionMutation });

    await runFactoryConfigureSave(deps);

    expect(deps.handleCommitStaging).not.toHaveBeenCalled();
    expect(deps.onAfterCommit).not.toHaveBeenCalled();
    expect(deps.onDone).toHaveBeenCalled();
  });
});
