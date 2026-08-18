import type { QueryClient } from "@tanstack/react-query";
import { canvasesListCanvases, type CanvasesCanvasSummary } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { isCanvasNameAlreadyExistsError, uniqueCanvasName } from "./uniqueCanvasName";

export const MAX_NAME_RETRY_ATTEMPTS = 20;

export async function listExistingCanvasNames(organizationId: string, queryClient: QueryClient): Promise<string[]> {
  const cached = queryClient.getQueryData<CanvasesCanvasSummary[]>(canvasKeys.list(organizationId));
  if (cached) {
    return cached.map((canvas) => canvas.name).filter((name): name is string => Boolean(name));
  }

  const response = await canvasesListCanvases(withOrganizationHeader({ organizationId }));
  return (response.data?.canvases ?? []).map((canvas) => canvas.name).filter((name): name is string => Boolean(name));
}

/**
 * Calls `createCanvas` with a name derived from `title`, retrying under a
 * suffixed name (via uniqueCanvasName) whenever the create call rejects with
 * a name-collision error. Handles both a stale client-side name cache and a
 * fresh server-side collision on the same attempt.
 */
export async function createCanvasWithUniqueName<T extends { canvasId: string }>(args: {
  title: string;
  existingNames: Set<string>;
  createCanvas: (name: string) => Promise<T>;
  isNameCollisionError?: (error: unknown) => boolean;
  failureMessage?: string;
}): Promise<T & { canvasName: string }> {
  const isCollision = args.isNameCollisionError ?? isCanvasNameAlreadyExistsError;
  let canvasName = uniqueCanvasName(args.title, args.existingNames);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt++) {
    try {
      const result = await args.createCanvas(canvasName);
      return { ...result, canvasName };
    } catch (error) {
      if (!isCollision(error)) {
        throw error;
      }
      args.existingNames.add(canvasName);
      canvasName = uniqueCanvasName(args.title, args.existingNames);
    }
  }

  throw new Error(args.failureMessage ?? "Failed to create canvas with a unique name");
}
