import type { QueryClient } from "@tanstack/react-query";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { fetchCanvasVersionWithSpec } from "./repository-spec-files";

export async function syncCommittedCanvasDraftState({
  queryClient,
  organizationId,
  canvasId,
  versionId,
}: {
  queryClient: QueryClient;
  organizationId: string;
  canvasId: string;
  versionId: string;
}): Promise<CanvasesCanvasVersion | undefined> {
  const committedVersion = await fetchCanvasVersionWithSpec(canvasId, versionId, false);
  if (!committedVersion) {
    return undefined;
  }

  queryClient.setQueryData(canvasKeys.versionStagedDetail(canvasId, versionId), committedVersion);
  queryClient.setQueryData(canvasKeys.versionDetail(canvasId, versionId), committedVersion);

  queryClient.setQueryData(canvasKeys.versionList(canvasId), (current: CanvasesCanvasVersion[] | undefined) => {
    if (!current) {
      return current;
    }

    return current.map((item) => (item.metadata?.id === versionId ? { ...item, spec: committedVersion.spec } : item));
  });

  if (committedVersion.spec) {
    queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) => {
      if (!current) {
        return current;
      }

      return {
        ...current,
        spec: { ...current.spec, ...committedVersion.spec },
      };
    });
  }

  return committedVersion;
}
