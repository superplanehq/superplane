import type { DashboardLayoutItem, DashboardPanel } from "@/hooks/useCanvasData";

import { dashboardToYaml } from "../dashboard/dashboardYaml";
import type { WorkflowFile } from "../workflow-files-types";

type DashboardPanelInput = {
  id?: string;
  type?: string;
  content?: unknown;
};

type DashboardLayoutInput = {
  i?: string;
  x?: number;
  y?: number;
  w?: number;
  h?: number;
  minW?: number;
  minH?: number;
};

type CanvasYamlPayload = {
  yamlText: string;
} | null;

export function buildWorkflowFiles({
  canvasYamlPayload,
  panels,
  layout,
  canvasId,
  canvasName,
  consoleLoading,
  consoleError,
}: {
  canvasYamlPayload: CanvasYamlPayload;
  panels: DashboardPanelInput[] | undefined;
  layout: DashboardLayoutInput[] | undefined;
  canvasId: string | null | undefined;
  canvasName: string | undefined;
  consoleLoading: boolean;
  consoleError: unknown;
}): WorkflowFile[] {
  const consoleYamlText = dashboardToYaml({
    panels: normalizePanels(panels),
    layout: normalizeLayout(layout),
    canvasId: canvasId || undefined,
    canvasName,
  });

  return [
    {
      path: "canvas.yaml",
      content: canvasYamlPayload?.yamlText || "",
      language: "yaml",
      loading: !canvasYamlPayload,
    },
    {
      path: "console.yaml",
      content: consoleYamlText,
      language: "yaml",
      loading: consoleLoading,
      errorMessage: consoleError ? String(consoleError) : undefined,
    },
  ];
}

function normalizePanels(panels: DashboardPanelInput[] | undefined): DashboardPanel[] {
  return (panels || []).map((panel) => ({
    id: panel.id || "",
    type: panel.type || "markdown",
    content: (panel.content as Record<string, unknown>) || {},
  }));
}

function normalizeLayout(layout: DashboardLayoutInput[] | undefined): DashboardLayoutItem[] {
  return (layout || []).map((item) => ({
    i: item.i || "",
    x: item.x || 0,
    y: item.y || 0,
    w: item.w || 12,
    h: item.h || 6,
    ...(item.minW !== undefined ? { minW: item.minW } : {}),
    ...(item.minH !== undefined ? { minH: item.minH } : {}),
  }));
}
