import { useCallback, useMemo, useRef } from "react";
import type { CanvasesDashboardLayoutItem, CanvasesDashboardPanel } from "@/api-client";

import type { SuperplaneComponentsNode } from "@/api-client";
import type {
  CanvasDashboardQueryResult,
  DashboardLayoutItem,
  DashboardPanel,
  UpdateCanvasDashboardMutationResult,
} from "@/hooks/useCanvasData";

import { DashboardView } from "./DashboardView";
import { DashboardYamlModal } from "./DashboardYamlModal";
import { DashboardContextProvider, type DashboardNodeStatus } from "./DashboardContext";

export type DashboardOverlayProps = {
  /**
   * Aggregated read-only flag: true when the viewer cannot edit the dashboard
   * structure (panels, layout, markdown body) for any reason — missing the
   * `canvases:update` permission, viewing a template, or the canvas being
   * deleted remotely. Mirrors the server-side guard in
   * `UpdateCanvasDashboard`; the server remains the source of truth.
   */
  readOnly: boolean;
  /**
   * True when the viewer can import/replace the dashboard with YAML. Falls
   * back to false when `readOnly` is true. Server-side authorization remains
   * the source of truth; this is for UX only.
   */
  canImportYaml: boolean;
  /**
   * True when the viewer can invoke runtime actions on the underlying canvas:
   * trigger chips, run-trigger row actions, approve/cancel/push-through
   * execution hooks. This maps 1:1 to the `canvases:update` permission used
   * by the `InvokeNodeTriggerHook` / `InvokeNodeExecutionHook` interceptor
   * rules; the same backend rules apply even if the UI is bypassed.
   */
  canRunNodes: boolean;
  dashboardQuery: CanvasDashboardQueryResult;
  updateDashboardMutation: UpdateCanvasDashboardMutationResult;
  addPanelDialogOpen: boolean;
  onAddPanelDialogOpenChange: (open: boolean) => void;
  /** Controlled state for the YAML modal — owned by the canvas page header. */
  yamlModalOpen: boolean;
  onYamlModalOpenChange: (open: boolean) => void;
  canvasId?: string;
  canvasName?: string;
  /** Organization id for chip navigation. Required when chips should be live. */
  organizationId?: string;
  /** Canvas nodes used to resolve chip references by id or name. */
  canvasNodes?: SuperplaneComponentsNode[];
  /** Latest known node status per node id; powers the status chip. */
  nodeStatuses?: Record<string, DashboardNodeStatus | undefined>;
  /** Callback invoked when a manual-run chip is clicked. */
  onTriggerNode?: (nodeId: string, options?: { templateName?: string; triggerName?: string }) => void;
};

export function DashboardOverlay({
  readOnly,
  canImportYaml,
  canRunNodes,
  dashboardQuery,
  updateDashboardMutation,
  addPanelDialogOpen,
  onAddPanelDialogOpenChange,
  yamlModalOpen,
  onYamlModalOpenChange,
  canvasId,
  canvasName,
  organizationId,
  canvasNodes,
  nodeStatuses,
  onTriggerNode,
}: DashboardOverlayProps) {
  const updateDashboardMutationRef = useRef(updateDashboardMutation);
  updateDashboardMutationRef.current = updateDashboardMutation;

  const panels: DashboardPanel[] = useMemo(
    () =>
      (dashboardQuery.data?.panels || []).map((panel: CanvasesDashboardPanel) => ({
        id: panel.id || "",
        type: panel.type || "markdown",
        content: (panel.content as Record<string, unknown>) || {},
      })),
    [dashboardQuery.data?.panels],
  );

  const layout: DashboardLayoutItem[] = useMemo(
    () =>
      (dashboardQuery.data?.layout || []).map((item: CanvasesDashboardLayoutItem) => ({
        i: item.i || "",
        x: item.x || 0,
        y: item.y || 0,
        w: item.w || 12,
        h: item.h || 6,
        ...(item.minW !== undefined ? { minW: item.minW } : {}),
        ...(item.minH !== undefined ? { minH: item.minH } : {}),
      })),
    [dashboardQuery.data?.layout],
  );

  const handleChange = useCallback((next: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => {
    updateDashboardMutationRef.current.mutate(next);
  }, []);

  const handleImportYaml = useCallback(async (next: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => {
    // Use mutateAsync so the modal can await before closing/showing toasts.
    await updateDashboardMutationRef.current.mutateAsync(next);
  }, []);

  const contextNodes = canvasNodes ?? [];
  // The YAML modal is opened from the canvas page header (next to "Add panel").
  // The dashboard overlay no longer renders its own toolbar so the white
  // strip is gone and the grid fills the available area.
  const overlayContent = (
    <div
      className="absolute inset-x-0 bottom-0 z-10 flex flex-col bg-slate-100 top-[calc(2.75rem+3rem)]"
      data-testid="dashboard-overlay"
    >
      <div className="min-h-0 flex-1 overflow-auto">
        <DashboardView
          panels={panels}
          layout={layout}
          isLoading={dashboardQuery.isLoading}
          errorMessage={dashboardQuery.error ? String(dashboardQuery.error) : undefined}
          readOnly={readOnly}
          onChange={handleChange}
          addPanelDialogOpen={addPanelDialogOpen}
          onAddPanelDialogOpenChange={onAddPanelDialogOpenChange}
        />
      </div>

      <DashboardYamlModal
        open={yamlModalOpen}
        onOpenChange={onYamlModalOpenChange}
        panels={panels}
        layout={layout}
        canvasId={canvasId}
        canvasName={canvasName}
        onImport={canImportYaml ? handleImportYaml : undefined}
        isImporting={updateDashboardMutation.isPending}
      />
    </div>
  );

  if (!canvasId || !organizationId) {
    return overlayContent;
  }

  return (
    <DashboardContextProvider
      canvasId={canvasId}
      organizationId={organizationId}
      nodes={contextNodes}
      nodeStatuses={nodeStatuses}
      canRunNodes={canRunNodes}
      onTriggerNode={canRunNodes ? onTriggerNode : undefined}
    >
      {overlayContent}
    </DashboardContextProvider>
  );
}
