import { useFactoryPullRequests, useWorkOrderArtifacts, useWorkOrderEvents } from "@/hooks/useFactoryData";
import { useMemo } from "react";

import { flattenWorkOrderEventsPages } from "../../lib/workOrderEventsPagination";
import { streamArtifactIndexFromEvents, type StreamArtifactIndex } from "./attachStreamArtifacts";

const EMPTY_INDEX: StreamArtifactIndex = {
  byRun: new Map(),
};

export function useSplitRunStreamArtifacts(
  organizationId: string | undefined,
  factoryId: string | undefined,
  orderId: string | undefined,
): StreamArtifactIndex {
  const eventsQuery = useWorkOrderEvents(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const artifactsQuery = useWorkOrderArtifacts(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const pullRequestsQuery = useFactoryPullRequests(
    organizationId ?? "",
    factoryId ?? "",
    orderId ? { workOrderIds: [orderId] } : undefined,
  );
  const events = useMemo(() => flattenWorkOrderEventsPages(eventsQuery.data?.pages), [eventsQuery.data?.pages]);

  return useMemo(() => {
    if (!organizationId || !factoryId || !orderId) {
      return EMPTY_INDEX;
    }
    return streamArtifactIndexFromEvents(events, artifactsQuery.data, pullRequestsQuery.data);
  }, [artifactsQuery.data, events, factoryId, orderId, organizationId, pullRequestsQuery.data]);
}
