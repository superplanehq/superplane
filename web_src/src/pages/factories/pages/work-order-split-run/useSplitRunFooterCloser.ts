import type { FactoriesWorkOrder } from "@/api-client";
import { useWorkOrderEvents } from "@/hooks/useFactoryData";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { useMemo } from "react";

import { flattenWorkOrderEventsPages } from "../../lib/workOrderEventsPagination";
import { getWorkOrderDisplayStatus } from "../../lib/workOrderProgress";
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
  return useMemo(
    () => footerCloserFromEvents(events, displayStatus, resolveUser),
    [displayStatus, events, resolveUser],
  );
}
