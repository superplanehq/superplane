import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { factoriesDescribeFactoryVelocity, factoriesSyncFactoryVelocity } from "@/api-client";
import type {
  DescribeFactoryVelocityRequestPeopleSort,
  DescribeFactoryVelocityRequestSortDirection,
  FactoriesDescribeFactoryVelocityResponse,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import {
  PEOPLE_PAGE_SIZE,
  PEOPLE_SORT_DEFAULT_DIRECTION,
  PEOPLE_SORT_DEFAULT_KEY,
  peopleSortDirectionParam,
  peopleSortParam,
  type PeopleSortDirection,
  type PeopleSortKey,
} from "@/pages/factories/lib/velocityPeopleSort";

import { factoryQueryKeys } from "./useFactoryData";

export interface FactoryVelocityParams {
  periodDays: number;
  repository?: string;
  /** Column the People table sorts on. Defaults to `"total"`. */
  peopleSort?: PeopleSortKey;
  /** Defaults to `"desc"`. */
  peopleSortDirection?: PeopleSortDirection;
  /** 0-based row offset into the sorted People table. Defaults to 0. */
  peopleOffset?: number;
  /** Rows per page of the People table. Defaults to `PEOPLE_PAGE_SIZE`. */
  peoplePageSize?: number;
}

export function useFactoryVelocity(organizationId: string, factoryId: string, params: FactoryVelocityParams) {
  const peopleSort = params.peopleSort ?? PEOPLE_SORT_DEFAULT_KEY;
  const peopleSortDirection = params.peopleSortDirection ?? PEOPLE_SORT_DEFAULT_DIRECTION;
  const peopleOffset = params.peopleOffset ?? 0;
  const peoplePageSize = params.peoplePageSize ?? PEOPLE_PAGE_SIZE;

  return useQuery({
    queryKey: factoryQueryKeys.velocity(organizationId, factoryId, {
      periodDays: params.periodDays,
      repository: params.repository ?? "",
      peopleSort,
      peopleSortDirection,
      peopleOffset,
      peoplePageSize,
    }),
    queryFn: async (): Promise<FactoriesDescribeFactoryVelocityResponse> => {
      const query: Record<
        string,
        string | number | DescribeFactoryVelocityRequestPeopleSort | DescribeFactoryVelocityRequestSortDirection
      > = {
        periodDays: params.periodDays,
        peopleSort: peopleSortParam(peopleSort),
        peopleSortDirection: peopleSortDirectionParam(peopleSortDirection),
        peopleOffset,
        peoplePageSize,
      };
      if (params.repository) query.repository = params.repository;

      const response = await factoriesDescribeFactoryVelocity(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query,
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId && factoryId),
    staleTime: 60 * 1000,
  });
}

/** How often the report is read back while a sync runs. */
const VELOCITY_SYNC_POLL_MS = 3000;

/**
 * How long to follow a sync before giving up on it.
 *
 * A sync rebuilds sixty days of history, so how long it takes depends on the
 * repository. Giving up only stops the progress indicator; the sync itself
 * carries on and the next read picks up its rows.
 */
const VELOCITY_SYNC_TIMEOUT_MS = 120_000;

/**
 * Asks for a fresh read of the repository merges and follows it to completion.
 *
 * Velocity reads rows a background sync collects, so a merge is only visible
 * after a sync picked it up. The request returns as soon as the sync is handed
 * to the worker, which is why the report is polled rather than read once: the
 * duration belongs to the repository, not to a delay this code can guess.
 */
export function useSyncFactoryVelocity(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const syncedBefore = newestSyncedAt(queryClient, organizationId, factoryId);

      const response = await factoriesSyncFactoryVelocity(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {},
        }),
      );
      return { started: response.data?.started ?? false, syncedBefore };
    },
    onSuccess: async ({ started, syncedBefore }) => {
      if (!started) return;
      await waitForFreshReport(queryClient, organizationId, factoryId, syncedBefore);
    },
  });
}

/**
 * The newest sync time the cached report knows about, as milliseconds.
 *
 * The page can hold a report per period, and any of them proves the sync
 * finished. A report with no sync yet counts as zero.
 */
function newestSyncedAt(queryClient: QueryClient, organizationId: string, factoryId: string): number {
  const cached = queryClient.getQueriesData<FactoriesDescribeFactoryVelocityResponse>({
    queryKey: factoryQueryKeys.velocityAll(organizationId, factoryId),
  });

  return cached.reduce((newest, [, report]) => {
    const syncedAt = report?.peopleSyncedAt ? Date.parse(report.peopleSyncedAt) : 0;
    return Number.isNaN(syncedAt) ? newest : Math.max(newest, syncedAt);
  }, 0);
}

async function waitForFreshReport(
  queryClient: QueryClient,
  organizationId: string,
  factoryId: string,
  syncedBefore: number,
): Promise<void> {
  const queryKey = factoryQueryKeys.velocityAll(organizationId, factoryId);
  const deadline = Date.now() + VELOCITY_SYNC_TIMEOUT_MS;

  while (Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, VELOCITY_SYNC_POLL_MS));
    await queryClient.invalidateQueries({ queryKey });

    if (newestSyncedAt(queryClient, organizationId, factoryId) > syncedBefore) return;
  }
}
