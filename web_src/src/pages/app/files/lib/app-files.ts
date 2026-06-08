import type { CanvasesCanvas } from "@/api-client";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";

import { materializeCanvasSpec, materializeConsoleSpec } from "../../lib/workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "../../lib/workflow-spec-paths";
import type { AppFile } from "../types";

type ConsolePanelInput = {
  id?: string;
  type?: string;
  content?: unknown;
};

type ConsoleLayoutInput = {
  i?: string;
  x?: number;
  y?: number;
  w?: number;
  h?: number;
  minW?: number;
  minH?: number;
};

export function buildAppFiles({
  canvas,
  canvasNodes,
  panels,
  layout,
  canvasId,
  canvasName,
  consoleLoading,
  consoleError,
}: {
  canvas: CanvasesCanvas | null | undefined;
  canvasNodes?: Parameters<typeof materializeCanvasSpec>[1];
  panels: ConsolePanelInput[] | undefined;
  layout: ConsoleLayoutInput[] | undefined;
  canvasId: string | null | undefined;
  canvasName: string | undefined;
  consoleLoading: boolean;
  consoleError: unknown;
}): AppFile[] {
  const canvasYamlText = canvas ? materializeCanvasSpec(canvas, canvasNodes) : "";
  const consoleYamlText = materializeConsoleSpec({
    panels: normalizePanels(panels),
    layout: normalizeLayout(layout),
    canvasId: canvasId || undefined,
    canvasName,
  });

  return [
    {
      path: CANVAS_YAML_PATH,
      content: canvasYamlText,
      language: "yaml",
      loading: !canvas,
    },
    {
      path: CONSOLE_YAML_PATH,
      content: consoleYamlText,
      language: "yaml",
      loading: consoleLoading,
      errorMessage: consoleError ? String(consoleError) : undefined,
    },
  ];
}

function normalizePanels(panels: ConsolePanelInput[] | undefined): ConsolePanel[] {
  return (panels || []).map((panel) => ({
    id: panel.id || "",
    type: panel.type || "markdown",
    content: (panel.content as Record<string, unknown>) || {},
  }));
}

function normalizeLayout(layout: ConsoleLayoutInput[] | undefined): ConsoleLayoutItem[] {
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
