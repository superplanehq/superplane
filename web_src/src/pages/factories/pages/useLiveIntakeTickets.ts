import { useEventExecutionsBatch, useInfiniteNodeEvents } from "@/hooks/useCanvasData";
import { useMemo } from "react";

import { analyzingTicketsFromIntakeRuns } from "./intakeRunModel";
import type { ConfiguredLineIntakeSource, LineIntakeAnalyzingTicket } from "./lineIntakeModel";

interface LiveIntakeTickets {
  tickets: LineIntakeAnalyzingTicket[];
  isLoading: boolean;
  isError: boolean;
  retry: () => void;
}

export function useLiveIntakeTickets(
  configuredSource: ConfiguredLineIntakeSource | undefined,
  enabled: boolean,
): LiveIntakeTickets {
  const eventsQuery = useInfiniteNodeEvents(
    configuredSource?.appId ?? "",
    configuredSource?.triggerNodeId ?? "",
    enabled && Boolean(configuredSource),
  );
  const events = useMemo(
    () => eventsQuery.data?.pages.flatMap((page) => page?.events ?? []) ?? [],
    [eventsQuery.data?.pages],
  );
  const eventIds = useMemo(() => events.flatMap((event) => (event.id ? [event.id] : [])), [events]);
  const executions = useEventExecutionsBatch(configuredSource?.appId ?? "", eventIds);

  const tickets = useMemo(() => {
    if (!configuredSource) {
      return [];
    }
    return analyzingTicketsFromIntakeRuns(
      configuredSource.source.id,
      configuredSource.analysisNodeId,
      events,
      executions.queries.map((query) => query.data),
    ).map((ticket) => ({ ...ticket, appId: configuredSource.appId }));
  }, [configuredSource, events, executions.queries]);

  return {
    tickets,
    isLoading: eventsQuery.isLoading || executions.isLoading,
    isError: eventsQuery.isError || executions.queries.some((query) => query.isError),
    retry: () => {
      void eventsQuery.refetch();
      for (const query of executions.queries) {
        void query.refetch();
      }
    },
  };
}
