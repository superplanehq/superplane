import { useCallback, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { useAppDashboard, useUpdateAppDashboard } from "@/hooks/useAppData";
import { DashboardView } from "@/pages/workflowv2/dashboard/DashboardView";
import type { DashboardPanel, DashboardLayoutItem } from "@/hooks/useCanvasData";
import { Plus } from "lucide-react";

interface AppDashboardTabProps {
  appId: string;
  readOnly?: boolean;
}

export function AppDashboardTab({ appId, readOnly = false }: AppDashboardTabProps) {
  const [addPanelDialogOpen, setAddPanelDialogOpen] = useState(false);

  const dashboardQuery = useAppDashboard(appId);
  const updateDashboardMutation = useUpdateAppDashboard(appId);
  const updateMutationRef = useRef(updateDashboardMutation);
  updateMutationRef.current = updateDashboardMutation;

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
    updateMutationRef.current.mutate(next);
  }, []);

  return (
    <div className="flex flex-col h-full">
      {!readOnly && (
        <div className="flex items-center justify-end px-4 py-2 border-b border-slate-200 dark:border-slate-700">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setAddPanelDialogOpen(true)}
            className="flex items-center gap-1"
          >
            <Plus className="h-4 w-4" />
            Add Panel
          </Button>
        </div>
      )}
      <div className="flex-1 overflow-auto bg-slate-100 dark:bg-slate-900">
        <DashboardView
          panels={panels}
          layout={layout}
          isLoading={dashboardQuery.isLoading}
          errorMessage={dashboardQuery.error ? String(dashboardQuery.error) : undefined}
          readOnly={readOnly}
          onChange={handleChange}
          addPanelDialogOpen={addPanelDialogOpen}
          onAddPanelDialogOpenChange={setAddPanelDialogOpen}
        />
      </div>
    </div>
  );
}
