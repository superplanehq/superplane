import { useWorkOrder, useWorkOrderArtifacts, useWorkOrderEvents } from "@/hooks/useFactoryData";
import { useMemo } from "react";

import { flattenWorkOrderEventsPages } from "../../lib/workOrderEventsPagination";
import { streamArtifactIndexFromEvents, type StreamArtifactIndex } from "./attachStreamArtifacts";

const EMPTY_INDEX: StreamArtifactIndex = {
  byNodeId: new Map(),
  byNodeName: new Map(),
  pullRequestsByNodeId: new Map(),
  pullRequestsByNodeName: new Map(),
};

export function useSplitRunStreamArtifacts(
  organizationId: string | undefined,
  factoryId: string | undefined,
  orderId: string | undefined,
): StreamArtifactIndex {
  const eventsQuery = useWorkOrderEvents(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const artifactsQuery = useWorkOrderArtifacts(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const workOrderQuery = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const events = useMemo(() => flattenWorkOrderEventsPages(eventsQuery.data?.pages), [eventsQuery.data?.pages]);

  return useMemo(() => {
    if (!organizationId || !factoryId || !orderId) {
      return EMPTY_INDEX;
    }
    return streamArtifactIndexFromEvents(events, artifactsQuery.data, workOrderQuery.data?.pullRequests);
  }, [artifactsQuery.data, events, factoryId, orderId, organizationId, workOrderQuery.data?.pullRequests]);
}
