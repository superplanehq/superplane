import { useCallback, useRef } from "react";
import type { CanvasesDashboardLayoutItem, CanvasesDashboardPanel } from "@/api-client";

import type {
  CanvasDashboardQueryResult,
  DashboardLayoutItem,
  DashboardPanel,
  UpdateCanvasDashboardMutationResult,
} from "@/hooks/useCanvasData";

import { DashboardView } from "./DashboardView";

export type DashboardOverlayProps = {
  readOnly: boolean;
  dashboardQuery: CanvasDashboardQueryResult;
  updateDashboardMutation: UpdateCanvasDashboardMutationResult;
  addPanelDialogOpen: boolean;
  onAddPanelDialogOpenChange: (open: boolean) => void;
};

export function DashboardOverlay({
  readOnly,
  dashboardQuery,
  updateDashboardMutation,
  addPanelDialogOpen,
  onAddPanelDialogOpenChange,
}: DashboardOverlayProps) {
  const updateDashboardMutationRef = useRef(updateDashboardMutation);
  updateDashboardMutationRef.current = updateDashboardMutation;

  const panels: DashboardPanel[] = (dashboardQuery.data?.panels || []).map((panel: CanvasesDashboardPanel) => ({
    id: panel.id || "",
    type: panel.type || "markdown",
    content: (panel.content as Record<string, unknown>) || {},
  }));
  const layout: DashboardLayoutItem[] = (dashboardQuery.data?.layout || []).map(
    (item: CanvasesDashboardLayoutItem) => ({
      i: item.i || "",
      x: item.x || 0,
      y: item.y || 0,
      w: item.w || 12,
      h: item.h || 6,
      ...(item.minW !== undefined ? { minW: item.minW } : {}),
      ...(item.minH !== undefined ? { minH: item.minH } : {}),
    }),
  );

  const handleChange = useCallback((next: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => {
    updateDashboardMutationRef.current.mutate(next);
  }, []);

  return (
    <div
      className="absolute inset-x-0 bottom-0 z-10 overflow-auto bg-slate-100 top-[calc(2.75rem+3rem)]"
      data-testid="dashboard-overlay"
    >
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
  );
}
