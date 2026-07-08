import {
  canvasesDescribeCanvasVersion,
  type CanvasesCanvasSpec,
  type CanvasesCanvasVersion,
  type CanvasesStaging,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { dematerializeCanvasSpec, dematerializeConsoleSpec, materializeConsoleSpec } from "./workflow-spec-files";
import { isNotFoundError } from "../workflowPageHelpers";

export const emptyCanvasStaging = (): CanvasesStaging => ({
  hasStaging: false,
  stagedPaths: [],
});

// Confirms whether a canvas version still exists.
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

export type ConsoleSpecData = {
  panels: NonNullable<ReturnType<typeof dematerializeConsoleSpec>>["panels"];
  layout: NonNullable<ReturnType<typeof dematerializeConsoleSpec>>["layout"];
  consoleYaml: string;
};

export function consoleSpecFromCanvasSpec(canvasId: string, spec: CanvasesCanvasSpec | undefined): ConsoleSpecData {
  const panels = (spec?.panels ?? []) as ConsoleSpecData["panels"];
  const layout = (spec?.layout ?? []) as ConsoleSpecData["layout"];
  const consoleYaml = materializeConsoleSpec({
    panels,
    layout,
    canvasId,
  });

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
