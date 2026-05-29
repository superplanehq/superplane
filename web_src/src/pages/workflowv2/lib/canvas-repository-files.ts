import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import type { WorkflowFile } from "../WorkflowFilesOverlayLayer";

export const PROTECTED_REPOSITORY_PATHS = new Set(["canvas.yaml", "console.yaml"]);

export function isProtectedRepositoryPath(path: string): boolean {
  return PROTECTED_REPOSITORY_PATHS.has(path);
}

export function encodeRepositoryFileContent(content: string): string {
  const bytes = new TextEncoder().encode(content);
  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

export function buildRepositoryFileUrl(canvasId: string, path: string): string {
  const params = new URLSearchParams({ path });
  return `/api/v1/canvases/${encodeURIComponent(canvasId)}/repository/file?${params.toString()}`;
}

export async function fetchCanvasRepositoryFileContent(canvasId: string, path: string): Promise<string> {
  const response = await fetch(buildRepositoryFileUrl(canvasId, path), {
    credentials: "include",
    headers: withOrganizationHeader().headers,
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Failed to load ${path}`);
  }

  return response.text();
}

export function mergeWorkflowFilePaths(virtualFiles: WorkflowFile[], repositoryPaths: string[]): string[] {
  const paths: string[] = [];
  const seen = new Set<string>();

  for (const file of virtualFiles) {
    paths.push(file.path);
    seen.add(file.path);
  }

  for (const path of repositoryPaths) {
    if (seen.has(path) || isProtectedRepositoryPath(path)) {
      continue;
    }
    paths.push(path);
    seen.add(path);
  }

  return paths;
}
