import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { CanvasesCanvas } from "@/api-client";
import { useCanvasYaml } from "./useCanvasYaml";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

vi.mock("@/lib/analytics", () => ({
  analytics: {
    yamlExport: vi.fn(),
  },
}));

const baseCanvas = {
  metadata: { id: "canvas-1", name: "Preview" },
  spec: {
    nodes: [{ id: "old-node" }],
    edges: [],
  },
} as unknown as CanvasesCanvas;

function renderUseCanvasYaml(
  overrides: Partial<Parameters<typeof useCanvasYaml>[0]> = {},
  handleSaveWorkflow = vi.fn().mockResolvedValue({ status: "saved" }),
) {
  const onWorkflowImported = vi.fn();

  const result = renderHook(() =>
    useCanvasYaml({
      canvasId: "canvas-1",
      organizationId: "org-1",
      open: true,
      onOpenChange: vi.fn(),
      isImporting: false,
      nodes: [],
      getYamlExportPayload: vi.fn().mockReturnValue({ yamlText: "spec: {}", filename: "canvas.yaml" }),
      canvas: baseCanvas,
      isReadOnly: false,
      handleSaveWorkflow,
      onWorkflowImported,
      ...overrides,
    }),
  );

  return { ...result, handleSaveWorkflow, onWorkflowImported };
}

describe("useCanvasYaml", () => {
  it("syncs the imported workflow after a successful save", async () => {
    const { result, handleSaveWorkflow, onWorkflowImported } = renderUseCanvasYaml();

    await result.current.modalProps.onImport?.({
      nodes: [{ id: "new-node" }],
      edges: [{ source: "new-node", target: "next-node" }],
    });

    expect(handleSaveWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({
        spec: expect.objectContaining({
          nodes: [{ id: "new-node" }],
          edges: [{ source: "new-node", target: "next-node" }],
        }),
      }),
    );
    expect(onWorkflowImported).toHaveBeenCalledWith(
      expect.objectContaining({
        spec: expect.objectContaining({
          nodes: [{ id: "new-node" }],
          edges: [{ source: "new-node", target: "next-node" }],
        }),
      }),
    );
  });

  it("rejects failed saves so the import dialog stays open", async () => {
    const handleSaveWorkflow = vi.fn().mockResolvedValue({ status: "stale" });
    const { result, onWorkflowImported } = renderUseCanvasYaml({}, handleSaveWorkflow);

    await expect(result.current.modalProps.onImport?.({ nodes: [], edges: [] })).rejects.toThrow(
      "The canvas changed while importing. Refresh and try again.",
    );
    expect(onWorkflowImported).not.toHaveBeenCalled();
  });
});
