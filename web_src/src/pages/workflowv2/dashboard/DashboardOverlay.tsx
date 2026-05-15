import { useCallback, useRef } from "react";

import {
  useCanvasDashboard,
  useUpdateCanvasDashboard,
  type DashboardLayoutItem,
  type DashboardPanel,
} from "@/hooks/useCanvasData";

import { DashboardView } from "./DashboardView";

export type DashboardOverlayProps = {
  readOnly: boolean;
  dashboardQuery: ReturnType<typeof useCanvasDashboard>;
  updateDashboardMutation: ReturnType<typeof useUpdateCanvasDashboard>;
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

  const panels: DashboardPanel[] = (dashboardQuery.data?.panels || []).map((p) => ({
    id: p.id || "",
    type: p.type || "markdown",
    content: (p.content as Record<string, unknown>) || {},
  }));
  const layout: DashboardLayoutItem[] = (dashboardQuery.data?.layout || []).map((l) => ({
    i: l.i || "",
    x: l.x || 0,
    y: l.y || 0,
    w: l.w || 12,
    h: l.h || 6,
    ...(l.minW !== undefined ? { minW: l.minW } : {}),
    ...(l.minH !== undefined ? { minH: l.minH } : {}),
  }));

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
