import type { CanvasesCanvas } from "@/api-client";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";
import type { CanvasNode } from "@/ui/CanvasPage";

import { consoleToYaml, parseConsoleYaml } from "../console/consoleYaml";

import {
  buildCanvasYamlFromWorkflow,
  parseCanvasYamlMetadata,
  parseCanvasYamlToSpec,
  type ParsedCanvasYamlMetadata,
} from "./canvas-yaml-staging";

export function applyRenderedNodePositions(workflow: CanvasesCanvas, canvasNodes?: CanvasNode[]): CanvasesCanvas {
  if (!canvasNodes?.length) {
    return workflow;
  }

  const updatedNodes =
    workflow.spec?.nodes?.map((node) => {
      const canvasNode = canvasNodes.find((cn) => cn.id === node.id);
      if (!canvasNode) {
        return node;
      }

      const componentType = (canvasNode.data?.type as string) || "";
      return {
        ...node,
        position: {
          x: Math.round(canvasNode.position.x),
          y: Math.round(canvasNode.position.y),
        },
        isCollapsed: (canvasNode.data[componentType] as { collapsed: boolean } | undefined)?.collapsed || false,
      };
    }) || [];

  return {
    ...workflow,
    spec: {
      ...workflow.spec,
      nodes: updatedNodes,
    },
  };
}

export function materializeCanvasSpec(workflow: CanvasesCanvas, canvasNodes?: CanvasNode[]): string {
  return buildCanvasYamlFromWorkflow(applyRenderedNodePositions(workflow, canvasNodes));
}

export function dematerializeCanvasSpec(yamlText: string): CanvasesCanvas["spec"] | null {
  return parseCanvasYamlToSpec(yamlText);
}

export function dematerializeCanvasMetadata(yamlText: string): ParsedCanvasYamlMetadata | null {
  return parseCanvasYamlMetadata(yamlText);
}

export function materializeConsoleSpec(input: {
  panels: ConsolePanel[];
  layout: ConsoleLayoutItem[];
  canvasId?: string;
  canvasName?: string;
}): string {
  return consoleToYaml(input);
}

export function dematerializeConsoleSpec(
  yamlText: string,
): { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] } | null {
  const result = parseConsoleYaml(yamlText);
  if (!result.ok) {
    return null;
  }

  return {
    panels: result.data.spec.panels,
    layout: result.data.spec.layout,
  };
}

export function canvasYamlDownloadFilename(canvasName?: string): string {
  const safeName = (canvasName || "canvas")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
  return `${safeName || "canvas"}.yaml`;
}

export function buildCanvasYamlExportPayload(
  workflow: CanvasesCanvas | null | undefined,
  canvasNodes?: CanvasNode[],
): { yamlText: string; filename: string } | null {
  if (!workflow) {
    return null;
  }

  return {
    yamlText: materializeCanvasSpec(workflow, canvasNodes),
    filename: canvasYamlDownloadFilename(workflow.metadata?.name),
  };
}

export function parseCanvasYamlForImport(
  text: string,
): { ok: true; spec: NonNullable<CanvasesCanvas["spec"]> } | { ok: false; error: string } {
  const trimmed = text.trim();
  if (!trimmed) {
    return { ok: false, error: "Please provide a YAML definition." };
  }

  const spec = dematerializeCanvasSpec(text);
  if (!spec) {
    return {
      ok: false,
      error: "YAML must contain a valid Canvas definition with apiVersion v1, kind Canvas, and a spec.nodes array.",
    };
  }

  return { ok: true, spec };
}

export function parseConsoleYamlForSave(
  text: string,
): { ok: true; panels: ConsolePanel[]; layout: ConsoleLayoutItem[] } | { ok: false; error: string } {
  const result = parseConsoleYaml(text);
  if (!result.ok) {
    return { ok: false, error: result.error };
  }

  return { ok: true, panels: result.data.spec.panels, layout: result.data.spec.layout };
}
