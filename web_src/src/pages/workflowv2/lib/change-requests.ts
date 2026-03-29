import type { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { versionSortValue } from "./canvas-versions";

export type ChangeRequestVersionRow = {
  version: CanvasesCanvasVersion;
  changeRequest: CanvasesCanvasChangeRequest;
};

function buildVisibleVersionIndex(visibleCanvasVersions: CanvasesCanvasVersion[]) {
  const indexedVisibleVersions = new Map<string, CanvasesCanvasVersion>();
  visibleCanvasVersions.forEach((version) => {
    const id = version.metadata?.id;
    if (!id) {
      return;
    }

    indexedVisibleVersions.set(id, version);
  });

  return indexedVisibleVersions;
}

function resolveChangeRequestVersion(
  changeRequest: CanvasesCanvasChangeRequest,
  indexedVisibleVersions: Map<string, CanvasesCanvasVersion>,
) {
  const versionFromRequest = changeRequest.version;
  const versionId = versionFromRequest?.metadata?.id || changeRequest.metadata?.versionId || "";
  const hasEmbeddedVersion = !!(versionFromRequest?.metadata?.id || versionFromRequest?.spec);

  return {
    versionId,
    resolvedVersion: hasEmbeddedVersion ? versionFromRequest : indexedVisibleVersions.get(versionId),
  };
}

function isDuplicateResolvedVersion(resolvedVersionId: string, seenVersionIds: Set<string>) {
  if (!resolvedVersionId) {
    return true;
  }

  return seenVersionIds.has(resolvedVersionId);
}

/**
 * Builds one sidebar row per version for change requests whose status matches
 * the requested filter. The API can return multiple requests for the same
 * version, and some requests only include the version ID, so this resolves the
 * version from `visibleCanvasVersions` when needed and keeps the newest request
 * for each version after sorting.
 */
export function buildChangeRequestVersionRowsForStatus(
  canvasChangeRequests: CanvasesCanvasChangeRequest[],
  visibleCanvasVersions: CanvasesCanvasVersion[],
  statusFilter: string,
): ChangeRequestVersionRow[] {
  const matchingChangeRequests = canvasChangeRequests
    .filter((changeRequest) => (changeRequest.metadata?.status || "").toLowerCase().includes(statusFilter))
    .sort(
      (left, right) =>
        versionSortValue(right.metadata?.updatedAt || right.metadata?.createdAt) -
        versionSortValue(left.metadata?.updatedAt || left.metadata?.createdAt),
    );

  const indexedVisibleVersions = buildVisibleVersionIndex(visibleCanvasVersions);

  const seenVersionIds = new Set<string>();
  const rows: ChangeRequestVersionRow[] = [];

  matchingChangeRequests.forEach((changeRequest) => {
    const { versionId, resolvedVersion } = resolveChangeRequestVersion(changeRequest, indexedVisibleVersions);
    if (!resolvedVersion) {
      return;
    }

    const resolvedVersionId = resolvedVersion?.metadata?.id || versionId;
    if (isDuplicateResolvedVersion(resolvedVersionId, seenVersionIds)) {
      return;
    }

    seenVersionIds.add(resolvedVersionId);
    rows.push({ version: resolvedVersion, changeRequest });
  });

  return rows;
}
