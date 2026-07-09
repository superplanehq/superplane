import { useCallback, useMemo, useRef } from "react";
import type { QueryClient } from "@tanstack/react-query";
import type { CanvasesCanvasRun, CanvasesListRunsResponse } from "@/api-client";
import { canvasesListRuns } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import {
  buildRunLookupIndex,
  buildRunLookupFingerprintFromSources,
  collectCachedCanvasRuns,
  EMPTY_RUN_LOOKUP_INDEX,
  findRunIdInLookupIndex,
  findRunInListRunsResponse,
  getSidebarEventLookupKey,
  seedRunInInfiniteRunsCache,
  shouldContinueRunLookupPagination,
} from "@/pages/app/sidebarRunLookup";

const RUN_LOOKUP_PAGE_LIMIT = 25;
/** Safety cap so a broken pagination cursor cannot loop forever. */
const RUN_LOOKUP_MAX_PAGES = 100;

type UseSidebarEventRunLookupOptions = {
  enabled?: boolean;
  canvasId?: string;
  organizationId?: string | null;
  queryClient: QueryClient;
  runs: CanvasesCanvasRun[];
  infiniteRunsPages?: Array<CanvasesListRunsResponse | undefined>;
};

type FetchRunLookupOptions = {
  maxPages?: number;
};

export function useSidebarEventRunLookup({
  enabled = true,
  canvasId,
  organizationId,
  queryClient,
  runs,
  infiniteRunsPages,
}: UseSidebarEventRunLookupOptions) {
  const unfilteredRunPages =
    enabled && canvasId
      ? queryClient.getQueryData<{
          pages?: Array<CanvasesListRunsResponse | undefined>;
        }>(canvasKeys.infiniteRuns(canvasId, {}))?.pages
      : undefined;

  const lookupFingerprint = useMemo(() => {
    if (!enabled) {
      return "";
    }

    return buildRunLookupFingerprintFromSources({
      primaryRuns: runs,
      pages: [...(infiniteRunsPages ?? []), ...(unfilteredRunPages ?? [])],
    });
  }, [enabled, infiniteRunsPages, runs, unfilteredRunPages]);

  const lookupIndex = useMemo(() => {
    if (!enabled || !lookupFingerprint) {
      return EMPTY_RUN_LOOKUP_INDEX;
    }

    const cachedRuns = collectCachedCanvasRuns({
      primaryRuns: runs,
      pages: [...(infiniteRunsPages ?? []), ...(unfilteredRunPages ?? [])],
    });

    return buildRunLookupIndex(cachedRuns);
  }, [enabled, infiniteRunsPages, lookupFingerprint, runs, unfilteredRunPages]);

  const inFlightRef = useRef(new Map<string, Promise<string | null>>());

  const resolveRunIdForSidebarEvent = useCallback(
    (event: SidebarEvent) => {
      if (!enabled) {
        return null;
      }

      return findRunIdInLookupIndex(lookupIndex, event);
    },
    [enabled, lookupIndex],
  );

  const fetchRunIdForSidebarEvent = useCallback(
    async (event: SidebarEvent, options: FetchRunLookupOptions = {}) => {
      if (!enabled || !canvasId) {
        return null;
      }

      const lookupKey = getSidebarEventLookupKey(event);
      if (!lookupKey) {
        return null;
      }

      const scopedLookupKey = `${canvasId}:${lookupKey}:${options.maxPages ?? "all"}`;

      const resolvedRunId = findRunIdInLookupIndex(lookupIndex, event);
      if (resolvedRunId) {
        return resolvedRunId;
      }

      const inFlight = inFlightRef.current.get(scopedLookupKey);
      if (inFlight) {
        return inFlight;
      }

      const fetchPromise = (async () => {
        let before: string | undefined;
        let loadedCount = 0;
        const maxPages = Math.min(options.maxPages ?? RUN_LOOKUP_MAX_PAGES, RUN_LOOKUP_MAX_PAGES);

        for (let page = 0; page < maxPages; page += 1) {
          const response = await canvasesListRuns(
            withOrganizationHeader({
              organizationId: organizationId ?? undefined,
              path: { canvasId },
              query: {
                limit: RUN_LOOKUP_PAGE_LIMIT,
                ...(before ? { before } : {}),
              },
            }),
          );

          const data = response.data;
          const pageRuns = data?.runs ?? [];
          const match = findRunInListRunsResponse(pageRuns, event);
          if (match) {
            if (match.run.id) {
              seedRunInInfiniteRunsCache(queryClient, canvasId, match.run);
            }
            return match.runId;
          }

          loadedCount += pageRuns.length;

          if (!shouldContinueRunLookupPagination({ pageRuns, loadedCount, response: data })) {
            break;
          }

          before = data!.lastTimestamp;
        }

        return null;
      })();

      inFlightRef.current.set(scopedLookupKey, fetchPromise);

      try {
        return await fetchPromise;
      } finally {
        inFlightRef.current.delete(scopedLookupKey);
      }
    },
    [canvasId, enabled, lookupIndex, organizationId, queryClient],
  );

  return {
    resolveRunIdForSidebarEvent,
    fetchRunIdForSidebarEvent,
  };
}
