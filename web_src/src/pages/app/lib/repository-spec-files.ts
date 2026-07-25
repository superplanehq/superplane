import {
  canvasesDescribeCanvasVersion,
  canvasesGetCanvasStaging,
  type CanvasesCanvasSpec,
  type CanvasesCanvasVersion,
  type CanvasesCanvasVersionMetadata,
  type CanvasesStagingSummary,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { dematerializeCanvasSpec, dematerializeConsoleSpec, materializeConsoleSpec } from "./workflow-spec-files";
import { toCanvasVersionShell } from "./canvas-versions";
import { isNotFoundError } from "../workflowPageHelpers";

// Confirms whether a canvas version still exists via DescribeCanvasVersion.
export async function canvasVersionExists(canvasId: string, versionId: string): Promise<boolean> {
  try {
    const response = await canvasesDescribeCanvasVersion(withOrganizationHeader({ path: { canvasId, versionId } }));
    return Boolean(response.data?.version?.metadata?.id);
  } catch (error) {
    return !isNotFoundError(error);
  }
}

// fetchRepositorySpecFileContent reads a repository file. The server treats
// `version_id` and `stage` as mutually exclusive query modes:
// - stage=true: effective staged content (or live committed when nothing is staged)
// - version_id: committed content for a historical version
// - neither: committed content for the live version
export async function fetchRepositorySpecFileContent(
  canvasId: string,
  path: string,
  versionId?: string,
  stage = false,
): Promise<string> {
  const params = new URLSearchParams({ path });
  if (stage) {
    params.set("stage", "true");
  } else if (versionId) {
    params.set("version_id", versionId);
  }

  const response = await fetch(`/api/v1/canvases/${encodeURIComponent(canvasId)}/repository/file?${params}`, {
    credentials: "include",
    headers: withOrganizationHeader().headers,
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Failed to load ${path}`);
  }

  return response.text();
}

// fetchCanvasStaging returns the current user's effective staged spec and summary.
export type CanvasStagingData = {
  stagingSummary: CanvasesStagingSummary;
  spec?: CanvasesCanvasSpec;
};

export async function fetchCanvasStaging(canvasId: string): Promise<CanvasStagingData> {
  const response = await canvasesGetCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
    }),
  );

  return {
    stagingSummary: response.data?.stagingSummary ?? { hasStaging: false, stagedPaths: [] },
    spec: response.data?.spec,
  };
}

// fetchCanvasStagingSummary returns the uncommitted staging summary for the
// current user on a canvas.
export async function fetchCanvasStagingSummary(canvasId: string): Promise<CanvasesStagingSummary | undefined> {
  const staging = await fetchCanvasStaging(canvasId);
  return staging.stagingSummary;
}

export function canvasVersionWithSpecFromYaml(
  version: CanvasesCanvasVersion | undefined,
  canvasYaml: string | undefined,
): CanvasesCanvasVersion | undefined {
  if (!version) {
    return version;
  }

  if (!canvasYaml) {
    return version;
  }

  const spec = dematerializeCanvasSpec(canvasYaml);
  if (!spec) {
    return version;
  }

  return { ...version, spec };
}

export function canvasVersionWithStagingSpec(
  version: CanvasesCanvasVersion | CanvasesCanvasVersionMetadata | undefined,
  spec: CanvasesCanvasSpec | undefined,
): CanvasesCanvasVersion | undefined {
  const shell = toCanvasVersionShell(version);
  if (!shell || !spec) {
    return shell;
  }

  return { ...shell, spec };
}

// fetchStagedCanvasVersionWithSpec reads the canvas-scoped staging layer. Staging
// is not keyed by version id; version metadata is only used as a display shell.
export async function fetchStagedCanvasVersionWithSpec(
  canvasId: string,
  versionMetadata?: CanvasesCanvasVersion | CanvasesCanvasVersionMetadata,
): Promise<CanvasesCanvasVersion | undefined> {
  const staging = await fetchCanvasStaging(canvasId);
  return canvasVersionWithStagingSpec(versionMetadata, staging.spec);
}

// fetchCanvasVersionWithSpec loads a version from DescribeCanvasVersion, including
// nodes and edges from the version row. Versions never change after publish, so
// callers should cache aggressively and avoid invalidating these reads.
export async function fetchCanvasVersionWithSpec(
  canvasId: string,
  versionId: string,
): Promise<CanvasesCanvasVersion | undefined> {
  const describeResponse = await canvasesDescribeCanvasVersion(
    withOrganizationHeader({
      path: { canvasId, versionId },
    }),
  );
  return describeResponse.data?.version;
}

export type ConsoleSpecData = {
  panels: NonNullable<ReturnType<typeof dematerializeConsoleSpec>>["panels"];
  layout: NonNullable<ReturnType<typeof dematerializeConsoleSpec>>["layout"];
  consoleYaml: string;
};

function protoConsolePanels(panels: NonNullable<CanvasesCanvasVersion["spec"]>["panels"]): ConsoleSpecData["panels"] {
  return (panels ?? []).map((panel) => ({
    id: panel.id ?? "",
    type: panel.type ?? "",
    content: (panel.content as Record<string, unknown> | undefined) ?? {},
  }));
}

function protoConsoleLayout(layout: NonNullable<CanvasesCanvasVersion["spec"]>["layout"]): ConsoleSpecData["layout"] {
  return (layout ?? []).map((item) => ({
    i: item.i ?? "",
    x: item.x ?? 0,
    y: item.y ?? 0,
    w: item.w ?? 0,
    h: item.h ?? 0,
    ...(item.minW !== undefined ? { minW: item.minW } : {}),
    ...(item.minH !== undefined ? { minH: item.minH } : {}),
  }));
}

export function consoleSpecFromVersion(
  canvasId: string,
  version: CanvasesCanvasVersion | undefined,
): ConsoleSpecData | undefined {
  const spec = version?.spec;
  if (!spec?.panels && !spec?.layout) {
    return undefined;
  }

  const panels = protoConsolePanels(spec.panels);
  const layout = protoConsoleLayout(spec.layout);
  const consoleYaml = materializeConsoleSpec({ panels, layout, canvasId });
  if (!consoleYaml.trim()) {
    return undefined;
  }

  return {
    panels,
    layout,
    consoleYaml,
  };
}

export function consoleSpecFromYaml(consoleYaml: string): ConsoleSpecData | undefined {
  const parsed = dematerializeConsoleSpec(consoleYaml);
  if (!parsed) {
    return undefined;
  }

  return {
    panels: parsed.panels,
    layout: parsed.layout,
    consoleYaml,
  };
}
