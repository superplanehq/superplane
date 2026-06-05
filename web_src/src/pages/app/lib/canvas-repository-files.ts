import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function encodeRepositoryFileContent(content: string): string {
  const bytes = new TextEncoder().encode(content);
  let binary = "";
  for (let index = 0; index < bytes.length; index += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(index, index + 0x8000));
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
