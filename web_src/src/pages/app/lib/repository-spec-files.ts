import {
  canvasesDescribeCanvasVersion,
  canvasesGetCanvasStaging,
  type CanvasesCanvasVersion,
  type CanvasesStagingSummary,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { dematerializeCanvasMetadata, dematerializeCanvasSpec, dematerializeConsoleSpec } from "./workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./workflow-spec-paths";
import { isNotFoundError } from "../workflowPageHelpers";

// Confirms whether a canvas version still exists. Used to distinguish a deleted
// version from an incidental repository-file 404, since fetchCanvasVersionWithSpec
// loads the version and its canvas.yaml in parallel. A non-404 describe error is
// treated as "exists" so the original failure is surfaced as transient.
export async function canvasVersionExists(canvasId: string, versionId: string): Promise<boolean> {
  try {
    const response = await canvasesDescribeCanvasVersion(withOrganizationHeader({ path: { canvasId, versionId } }));
    return Boolean(response.data?.version?.metadata?.id);
  } catch (error) {
    return !isNotFoundError(error);
  }
}

// fetchRepositorySpecFileContent reads a canvas.yaml/console.yaml file. When
// `stage` is set (only meaningful for a draft version), the server returns the
// effective staged content (staged edits overlaid on the committed version).
export async function fetchRepositorySpecFileContent(
  canvasId: string,
  path: string,
  versionId?: string,
  stage = false,
): Promise<string> {
  const params = new URLSearchParams({ path });
  if (versionId) {
    params.set("version_id", versionId);
  }
  if (stage && versionId) {
    params.set("stage", "true");
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

/** @deprecated Use fetchCanvasStagingSummary */
export async function fetchCanvasVersionStagingSummary(
  canvasId: string,
  _versionId: string,
): Promise<CanvasesStagingSummary | undefined> {
  return fetchCanvasStagingSummary(canvasId);
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

  // Carry the canvas-level name/description from canvas.yaml onto the version
  // metadata so edit mode can render the canvas without the DescribeCanvas
  // response. Only override when the file actually provides a value.
  const yamlMetadata = dematerializeCanvasMetadata(canvasYaml);
  const metadata = version.metadata
    ? {
        ...version.metadata,
        ...(yamlMetadata?.name ? { name: yamlMetadata.name } : {}),
        ...(yamlMetadata?.description !== undefined ? { description: yamlMetadata.description } : {}),
      }
    : version.metadata;

  return { ...version, metadata, spec };
}

export async function fetchCanvasVersionWithSpec(
  canvasId: string,
  versionId: string,
  stage = false,
): Promise<CanvasesCanvasVersion | undefined> {
  const [describeResponse, canvasYaml] = await Promise.all([
    canvasesDescribeCanvasVersion(
      withOrganizationHeader({
        path: { canvasId, versionId },
      }),
    ),
    fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, versionId, stage),
  ]);

  return canvasVersionWithSpecFromYaml(describeResponse.data?.version, canvasYaml);
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
