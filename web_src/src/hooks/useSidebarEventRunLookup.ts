import { useCallback, useMemo, useRef } from "react";
import type { QueryClient } from "@tanstack/react-query";
import type { CanvasesCanvasRun, CanvasesListRunsResponse } from "@/api-client";
import { canvasesListRuns } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import { getSidebarEventRunLookupBefore } from "@/pages/app/utils";
import {
  buildRunLookupIndex,
  buildRunLookupFingerprintFromSources,
  collectCachedCanvasRuns,
  EMPTY_RUN_LOOKUP_INDEX,
  findRunIdInLookupIndex,
  findRunInListRunsResponse,
  getSidebarEventLookupKey,
  seedRunInInfiniteRunsCache,
} from "@/pages/app/sidebarRunLookup";

const RUN_LOOKUP_PAGE_LIMIT = 25;
const RUN_LOOKUP_MAX_PAGES = 3;

type UseSidebarEventRunLookupOptions = {
  enabled?: boolean;
  canvasId?: string;
  organizationId?: string | null;
  queryClient: QueryClient;
  runs: CanvasesCanvasRun[];
  infiniteRunsPages?: Array<CanvasesListRunsResponse | undefined>;
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

  const fetchedRunIdsRef = useRef(new Map<string, string | null>());
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
    async (event: SidebarEvent) => {
      if (!enabled || !canvasId) {
        return null;
      }

      const lookupKey = getSidebarEventLookupKey(event);
      if (!lookupKey) {
        return null;
      }

      const cachedRunId = fetchedRunIdsRef.current.get(lookupKey);
      if (cachedRunId) {
        return cachedRunId;
      }

      const resolvedRunId = findRunIdInLookupIndex(lookupIndex, event);
      if (resolvedRunId) {
        fetchedRunIdsRef.current.set(lookupKey, resolvedRunId);
        return resolvedRunId;
      }

      const inFlight = inFlightRef.current.get(lookupKey);
      if (inFlight) {
        return inFlight;
      }

      const fetchPromise = (async () => {
        let before: string | undefined = getSidebarEventRunLookupBefore(event);

        for (let page = 0; page < RUN_LOOKUP_MAX_PAGES; page += 1) {
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

          const pageRuns = response.data?.runs ?? [];
          const match = findRunInListRunsResponse(pageRuns, event);
          if (match) {
            if (match.run.id) {
              seedRunInInfiniteRunsCache(queryClient, canvasId, match.run);
            }
            fetchedRunIdsRef.current.set(lookupKey, match.runId);
            return match.runId;
          }

          if (!response.data?.lastTimestamp || pageRuns.length === 0) {
            break;
          }

          before = response.data.lastTimestamp;
        }

        return null;
      })();

      inFlightRef.current.set(lookupKey, fetchPromise);

      try {
        return await fetchPromise;
      } finally {
        inFlightRef.current.delete(lookupKey);
      }
    },
    [canvasId, enabled, lookupIndex, organizationId, queryClient],
  );

  return {
    resolveRunIdForSidebarEvent,
    fetchRunIdForSidebarEvent,
  };
}
