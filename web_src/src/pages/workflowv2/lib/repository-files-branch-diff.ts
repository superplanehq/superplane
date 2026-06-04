import { canvasesListCanvasRepositoryFiles } from "@/api-client";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/lib/canvas-staging";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { fetchCanvasRepositoryFileContent } from "./canvas-repository-files";

const RESERVED_REPOSITORY_PATHS = new Set([CANVAS_YAML_PATH, CONSOLE_YAML_PATH]);

async function listRepositoryPaths(canvasId: string, branch?: string): Promise<string[]> {
  const response = await canvasesListCanvasRepositoryFiles(
    withOrganizationHeader({
      path: { canvasId },
      query: branch ? { branch } : undefined,
    }),
  );

  return (response.data?.files ?? [])
    .map((file) => file.path)
    .filter((path): path is string => !!path && !RESERVED_REPOSITORY_PATHS.has(path))
    .sort();
}

/** True when non-canvas repository files on a draft branch differ from live (main). */
export async function branchHasCommittedRepositoryFilesVersusLive(
  canvasId: string,
  branchName: string,
): Promise<boolean> {
  const [draftPaths, livePaths] = await Promise.all([
    listRepositoryPaths(canvasId, branchName),
    listRepositoryPaths(canvasId),
  ]);
  const paths = Array.from(new Set([...draftPaths, ...livePaths])).sort();
  if (paths.length === 0) {
    return false;
  }

  const comparisons = await Promise.all(
    paths.map(async (path) => {
      const [draftContent, liveContent] = await Promise.all([
        fetchCanvasRepositoryFileContent(canvasId, path, branchName).catch(() => ""),
        fetchCanvasRepositoryFileContent(canvasId, path).catch(() => ""),
      ]);
      return draftContent !== liveContent;
    }),
  );

  return comparisons.some(Boolean);
}
