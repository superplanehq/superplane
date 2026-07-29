import type { CanvasesCanvas } from "@/api-client";
import type { ConsoleLayoutItem, ConsolePage, ConsolePanel } from "@/hooks/useCanvasData";

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

type ConsolePageInput = {
  id?: string;
  name?: string;
  panels?: ConsolePanelInput[];
  layout?: ConsoleLayoutInput[];
};

export function buildAppFiles({
  canvas,
  canvasNodes,
  pages,
  canvasId,
  canvasName,
  consoleLoading,
  consoleError,
}: {
  canvas: CanvasesCanvas | null | undefined;
  canvasNodes?: Parameters<typeof materializeCanvasSpec>[1];
  pages: ConsolePageInput[] | undefined;
  canvasId: string | null | undefined;
  canvasName: string | undefined;
  consoleLoading: boolean;
  consoleError: unknown;
}): AppFile[] {
  const canvasYamlText = canvas ? materializeCanvasSpec(canvas, canvasNodes) : "";
  const consoleYamlText = materializeConsoleSpec({
    pages: normalizePages(pages),
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

function normalizePages(pages: ConsolePageInput[] | undefined): ConsolePage[] {
  return (pages || []).map((page) => ({
    id: page.id || "",
    ...(page.name ? { name: page.name } : {}),
    panels: normalizePanels(page.panels),
    layout: normalizeLayout(page.layout),
  }));
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
