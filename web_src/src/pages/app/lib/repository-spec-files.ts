import {
  canvasesDescribeCanvasVersion,
  canvasesGetCanvasStaging,
  type CanvasesCanvasVersion,
  type CanvasesCanvasVersionMetadata,
  type CanvasesStagingSummary,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { isNotFoundError } from "@/lib/errors";

import { dematerializeCanvasSpec, dematerializeConsoleSpec } from "./workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./workflow-spec-paths";
import { toCanvasVersionShell } from "./canvas-versions";

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

// fetchCanvasStagingSummary returns the uncommitted staging summary for the
// current user on a canvas.
export async function fetchCanvasStagingSummary(canvasId: string): Promise<CanvasesStagingSummary | undefined> {
  const response = await canvasesGetCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
    }),
  );
  return response.data?.stagingSummary;
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

// fetchStagedCanvasVersionWithSpec reads the canvas-scoped staging layer. Staging
// is not keyed by version id; version metadata is only used as a display shell.
export async function fetchStagedCanvasVersionWithSpec(
  canvasId: string,
  versionMetadata?: CanvasesCanvasVersion | CanvasesCanvasVersionMetadata,
): Promise<CanvasesCanvasVersion | undefined> {
  const shell = toCanvasVersionShell(versionMetadata);
  const canvasYaml = await fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, undefined, true);
  return canvasVersionWithSpecFromYaml(shell, canvasYaml);
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

export async function fetchConsoleSpecFromRepository(
  canvasId: string,
  versionId?: string,
  stage = false,
): Promise<ConsoleSpecData | undefined> {
  const consoleYaml = await fetchRepositorySpecFileContent(canvasId, CONSOLE_YAML_PATH, versionId, stage);
  if (!consoleYaml.trim()) {
    return undefined;
  }
  return consoleSpecFromYaml(consoleYaml);
}
