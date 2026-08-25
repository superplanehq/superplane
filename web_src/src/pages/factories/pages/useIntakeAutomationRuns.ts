import { useEventExecutionsBatch, useInfiniteNodeEvents } from "@/hooks/useCanvasData";
import { useMemo } from "react";

import { automationRunsFromIntakeEvents } from "./intakeRunModel";
import type { IntakeAutomationRun } from "./intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "./lineIntakeModel";

interface LiveIntakeRuns {
  runs: IntakeAutomationRun[];
  isLoading: boolean;
  isError: boolean;
  retry: () => void;
}

export function useIntakeAutomationRuns(configuredSource: ConfiguredLineIntakeSource | undefined): LiveIntakeRuns {
  const eventsQuery = useInfiniteNodeEvents(
    configuredSource?.appId ?? "",
    configuredSource?.triggerNodeId ?? "",
    Boolean(configuredSource),
  );
  const events = useMemo(
    () => eventsQuery.data?.pages.flatMap((page) => page?.events ?? []) ?? [],
    [eventsQuery.data?.pages],
  );
  const eventIds = useMemo(() => events.flatMap((event) => (event.id ? [event.id] : [])), [events]);
  const executions = useEventExecutionsBatch(configuredSource?.appId ?? "", eventIds);

  const runs = useMemo(() => {
    if (!configuredSource) {
      return [];
    }
    return automationRunsFromIntakeEvents(
      {
        appId: configuredSource.appId,
        sourceId: configuredSource.source.id,
        analysisNodeId: configuredSource.analysisNodeId,
        createWorkOrderNodeId: configuredSource.createWorkOrderNodeId,
      },
      events,
      executions.queries.map((query) => query.data),
    );
  }, [configuredSource, events, executions.queries]);

  return {
    runs,
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
