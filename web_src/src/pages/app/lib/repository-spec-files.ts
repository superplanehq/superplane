import {
  canvasesDescribeCanvasVersion,
  canvasesGetCanvasStaging,
  canvasesListCanvasVersions,
  type CanvasesCanvasVersion,
  type CanvasesStagingSummary,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { dematerializeCanvasSpec, dematerializeConsoleSpec } from "./workflow-spec-files";
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

export type RepositoryFileReadTarget =
  | { source: "live" }
  | { source: "staging" }
  | { source: "version"; versionId: string };

function repositoryFileReadTargetFromArgs(versionId?: string, stage = false): RepositoryFileReadTarget {
  if (stage) {
    return { source: "staging" };
  }

  if (versionId) {
    return { source: "version", versionId };
  }

  return { source: "live" };
}

function repositoryFileQueryParams(path: string, target: RepositoryFileReadTarget): URLSearchParams {
  const params = new URLSearchParams({ path });

  switch (target.source) {
    case "staging":
      params.set("stage", "true");
      break;
    case "version":
      params.set("version_id", target.versionId);
      break;
    case "live":
      break;
  }

  return params;
}

// fetchRepositorySpecFileContent reads a repository file using one of three
// mutually exclusive modes: live (default), staging (stage=true), or a specific
// committed version (version_id). Staging reads never send version_id.
export async function fetchRepositorySpecFileContent(
  canvasId: string,
  path: string,
  versionId?: string,
  stage = false,
): Promise<string> {
  const params = repositoryFileQueryParams(path, repositoryFileReadTargetFromArgs(versionId, stage));

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

  return { ...version, spec };
}

export async function fetchCanvasVersionWithSpec(
  canvasId: string,
  versionId: string,
  stage = false,
): Promise<CanvasesCanvasVersion | undefined> {
  const describeResponse = await canvasesDescribeCanvasVersion(
    withOrganizationHeader({
      path: { canvasId, versionId },
    }),
  );

  try {
    const canvasYaml = await fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, versionId, stage);
    return canvasVersionWithSpecFromYaml(describeResponse.data?.version, canvasYaml);
  } catch (error) {
    if (stage) {
      throw error;
    }

    // Some historical versions may not have canvas.yaml in the repository yet.
    // Fall back to the spec stored on the version row from the list API.
    const listResponse = await canvasesListCanvasVersions(
      withOrganizationHeader({
        path: { canvasId },
        query: { limit: 50 },
      }),
    );
    const listVersion = listResponse.data?.versions?.find((item) => item.metadata?.id === versionId);
    if (listVersion?.spec) {
      return listVersion;
    }

    throw error;
  }
}

// fetchLiveCommittedCanvasVersionWithSpec loads the current live version's
// committed canvas.yaml without pinning a version_id on the repository read.
// After a remote commit the previously-active version id is stale, but the live
// file endpoint still resolves to the new live version automatically.
export async function fetchLiveCommittedCanvasVersionWithSpec(
  canvasId: string,
): Promise<CanvasesCanvasVersion | undefined> {
  const [listResponse, canvasYaml] = await Promise.all([
    canvasesListCanvasVersions(
      withOrganizationHeader({
        path: { canvasId },
        query: { limit: 1 },
      }),
    ),
    fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH),
  ]);

  return canvasVersionWithSpecFromYaml(listResponse.data?.versions?.[0], canvasYaml);
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
