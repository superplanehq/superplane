import { ExecutionWithEvent } from "../store/types";
import { RunItem } from "./tabs/RunItem";
import { useOrganizationUsersForCanvas } from "../../../hooks/useCanvasData";
import { useMemo } from "react";
import { SuperplaneEvent } from "@/api-client";
import {
  formatDuration,
  getMinApprovedAt,
  getApprovalsNames,
  mapExecutionOutputs,
  mapExecutionEventInputs,
  createUserDisplayNames
} from "../utils/stageEventUtils";

interface ExecutionTimelineProps {
  executions: ExecutionWithEvent[];
  organizationId: string;
  connectionEventsById: Record<string, SuperplaneEvent>;
  eventsByExecutionId: Record<string, SuperplaneEvent>;
}

export const ExecutionTimeline = ({
  executions,
  organizationId,
  connectionEventsById,
  eventsByExecutionId,
}: ExecutionTimelineProps) => {
  // Fetch organization users to resolve user IDs to names
  const { data: orgUsers = [] } = useOrganizationUsersForCanvas(organizationId);

  const userDisplayNames = useMemo(() => createUserDisplayNames(orgUsers), [orgUsers]);


  if (executions.length === 0) {
    return (
      <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
        <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3 " aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>inbox</span>
        <p data-slot="text" className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No recent runs</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {
        executions.map((execution) => {
          const sourceEvent = connectionEventsById[execution.event.eventId || ''];
          const emmitedEvent = eventsByExecutionId[execution.id || ''];

          return (
            <RunItem
              key={execution.id!}
              title={execution.event.name || execution.id || 'Execution'}
              runId={execution.id}
              inputs={mapExecutionEventInputs(execution)}
              outputs={mapExecutionOutputs(execution)}
              state={execution.state || 'STATE_UNKNOWN'}
              result={execution.result || 'RESULT_UNKNOWN'}
              timestamp={execution.createdAt || new Date().toISOString()}
              executionDuration={formatDuration(execution.startedAt || execution.createdAt, execution.finishedAt)}
              approvedOn={getMinApprovedAt(execution)}
              approvedBy={getApprovalsNames(execution, userDisplayNames)}
              queuedOn={execution.event.createdAt}
              eventId={sourceEvent?.id}
              sourceEvent={sourceEvent}
              emmitedEvent={emmitedEvent}
            />
          );
        })
      }
    </div>
  );
};