import { canvasesDescribeCanvasVersion, type CanvasesCanvasVersion } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { dematerializeCanvasSpec, dematerializeConsoleSpec } from "./workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./workflow-spec-paths";

export async function fetchRepositorySpecFileContent(
  canvasId: string,
  path: string,
  versionId?: string,
): Promise<string> {
  const params = new URLSearchParams({ path });
  if (versionId) {
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

export async function fetchCanvasVersionWithSpec(
  canvasId: string,
  versionId: string,
): Promise<CanvasesCanvasVersion | undefined> {
  const [describeResponse, canvasYaml] = await Promise.all([
    canvasesDescribeCanvasVersion(
      withOrganizationHeader({
        path: { canvasId, versionId },
      }),
    ),
    fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, versionId),
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
): Promise<ConsoleSpecData | undefined> {
  const consoleYaml = await fetchRepositorySpecFileContent(canvasId, CONSOLE_YAML_PATH, versionId);
  if (!consoleYaml.trim()) {
    return undefined;
  }
  return consoleSpecFromYaml(consoleYaml);
}
