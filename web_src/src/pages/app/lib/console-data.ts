import type { CanvasesCanvasVersion } from "@/api-client";

import { consoleSpecFromVersion } from "./repository-spec-files";
import { dematerializeConsoleSpec } from "./workflow-spec-files";

export interface ConsolePanel {
  id: string;
  type: string;
  content: Record<string, unknown>;
}

export interface ConsoleLayoutItem {
  i: string;
  x: number;
  y: number;
  w: number;
  h: number;
  minW?: number;
  minH?: number;
}

export type CanvasConsoleData = {
  canvasId: string;
  versionId?: string;
  updatedAt?: string;
  panels: ConsolePanel[];
  layout: ConsoleLayoutItem[];
  consoleYaml: string;
};

export function parseConsoleDataFromVersion(
  canvasId: string,
  version: CanvasesCanvasVersion | undefined,
): CanvasConsoleData | undefined {
  const spec = consoleSpecFromVersion(canvasId, version);
  if (!spec) {
    return undefined;
  }

  return {
    canvasId,
    versionId: version?.metadata?.id,
    updatedAt: version?.metadata?.updatedAt,
    panels: spec.panels,
    layout: spec.layout,
    consoleYaml: spec.consoleYaml,
  };
}

export function parseConsoleDataFromYaml(
  canvasId: string,
  versionId: string | undefined,
  consoleYaml: string,
): CanvasConsoleData | undefined {
  const parsed = dematerializeConsoleSpec(consoleYaml);
  if (!parsed) {
    return undefined;
  }

  return {
    canvasId,
    versionId,
    panels: parsed.panels,
    layout: parsed.layout,
    consoleYaml,
  };
}
