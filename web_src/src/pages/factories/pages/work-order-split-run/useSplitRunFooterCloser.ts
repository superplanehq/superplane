import type { FactoriesWorkOrder } from "@/api-client";
import { useWorkOrderEvents } from "@/hooks/useFactoryData";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import { flattenWorkOrderEventsPages } from "../../lib/workOrderEventsPagination";
import { getWorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import {
  closerFromGitHubMergedBy,
  githubPullRequestApiHref,
  latestMergedGitHubPullRequest,
} from "./githubPullRequestMerger";
import { footerCloserFromEvents, type SplitRunFooterCloser } from "./splitRunFooterActor";

export function useSplitRunFooterCloser(
  organizationId: string,
  factoryId: string,
  order: FactoriesWorkOrder,
): SplitRunFooterCloser {
  const eventsQuery = useWorkOrderEvents(organizationId, factoryId, order.id ?? "");
  const { resolveUser } = useOrgUserLookup(organizationId);
  const events = useMemo(() => flattenWorkOrderEventsPages(eventsQuery.data?.pages), [eventsQuery.data?.pages]);
  const displayStatus = getWorkOrderDisplayStatus(order);
  const fromEvents = useMemo(
    () => footerCloserFromEvents(events, displayStatus, resolveUser),
    [displayStatus, events, resolveUser],
  );
  const apiHref = useMemo(() => {
    if (displayStatus !== "completed" || fromEvents.actor || fromEvents.automationHref) {
      return undefined;
    }
    return githubPullRequestApiHref(latestMergedGitHubPullRequest(order.pullRequests)?.url);
  }, [displayStatus, fromEvents.actor, fromEvents.automationHref, order.pullRequests]);
  const githubQuery = useQuery({
    queryKey: ["github-pull-request-merged-by", apiHref],
    queryFn: () => fetchGitHubMergedBy(apiHref ?? ""),
    enabled: Boolean(apiHref),
    staleTime: Infinity,
    retry: false,
  });
  return useMemo(() => {
    const fromGitHub = closerFromGitHubMergedBy(githubQuery.data);
    if (!fromGitHub.automationName) {
      return fromEvents;
    }
    return { ...fromEvents, ...fromGitHub };
  }, [fromEvents, githubQuery.data]);
}

async function fetchGitHubMergedBy(
  apiHref: string,
): Promise<{ login?: string; name?: string; html_url?: string } | null> {
  const response = await fetch(apiHref);
  if (!response.ok) {
    return null;
  }
  const body = (await response.json()) as { merged_by?: { login?: string; name?: string; html_url?: string } };
  return body.merged_by ?? null;
}
