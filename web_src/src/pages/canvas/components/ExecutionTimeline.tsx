import { ExecutionWithEvent } from "../store/types";
import { RunItem } from "./tabs/RunItem";
import { useOrganizationUsersForCanvas } from "../../../hooks/useCanvasData";
import { useMemo } from "react";

interface ExecutionTimelineProps {
  executions: ExecutionWithEvent[];
  organizationId: string;
}

export const ExecutionTimeline = ({
  executions,
  organizationId,
}: ExecutionTimelineProps) => {
  // Fetch organization users to resolve user IDs to names
  const { data: orgUsers = [] } = useOrganizationUsersForCanvas(organizationId);

  // Create a lookup map for user IDs to display names
  const userDisplayNames = useMemo(() => {
    const map: Record<string, string> = {};
    orgUsers.forEach(user => {
      if (user.metadata?.id) {
        map[user.metadata.id] = user.spec?.displayName || user.metadata?.email || user.metadata.id;
      }
    });
    return map;
  }, [orgUsers]);

  const getMinApprovedAt = (execution: ExecutionWithEvent) => {
    if (!execution.event.approvals?.length)
      return undefined;

    return execution.event.approvals.reduce((min, approval) => {
      if (approval.approvedAt && new Date(approval.approvedAt).getTime() < new Date(min).getTime()) {
        return approval.approvedAt;
      }
      return min;
    }, execution.event.approvals[0].approvedAt!);
  }


  const getApprovalsNames = (execution: ExecutionWithEvent) => {
    const names: string[] = [];
    execution.event.approvals?.forEach(approval => {
      if (approval.approvedBy) {
        names.push(userDisplayNames[approval.approvedBy]);
      }
    });
    return names.join(', ');
  };


  if (executions.length === 0) {
    return (
      <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
        <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3 " aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>inbox</span>
        <p data-slot="text" className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No recent runs</p>
      </div>
    );
  }




  const formatDuration = (startedAt?: string, finishedAt?: string) => {
    if (!startedAt || !finishedAt) {
      return "-";
    }
    const duration = new Date(finishedAt).getTime() - new Date(startedAt).getTime();
    const hours = Math.floor(duration / (1000 * 60 * 60));
    const prefixHours = hours >= 10 ? `${hours}h ` : `0${hours}h`;
    const minutes = Math.floor((duration % (1000 * 60 * 60)) / (1000 * 60));
    const prefixMinutes = minutes >= 10 ? `${minutes}m ` : `0${minutes}m`;
    const seconds = Math.floor((duration % (1000 * 60)) / 1000);
    const prefixSeconds = seconds >= 10 ? `${seconds}s` : `0${seconds}s`;
    return `${prefixHours} ${prefixMinutes} ${prefixSeconds}`;
  };

  const mapExecutionOutputs = (execution: ExecutionWithEvent) => {
    const map: Record<string, string> = {};
    execution.outputs?.forEach((output) => {
      if (!output.name) {
        return;
      }

      map[output.name!] = output.value!;
    });

    return map;
  };

  const mapExecutionEventInputs = (execution: ExecutionWithEvent) => {
    const map: Record<string, string> = {};
    execution.event.inputs?.forEach((input) => {
      if (!input.name) {
        return;
      }

      map[input.name!] = input.value!;
    });

    return map;
  };

  return (
    <div className="space-y-3">
      {
        executions.map((execution) => (
          <RunItem
            key={execution.id!}
            title={execution.event.label || execution.id || 'Execution'}
            runId={execution.id}
            inputs={mapExecutionEventInputs(execution)}
            outputs={mapExecutionOutputs(execution)}
            state={execution.state || 'STATE_UNKNOWN'}
            result={execution.result || 'RESULT_UNKNOWN'}
            timestamp={execution.createdAt || new Date().toISOString()}
            executionDuration={formatDuration(execution.startedAt, execution.finishedAt)}
            approvedOn={getMinApprovedAt(execution)}
            approvedBy={getApprovalsNames(execution)}
            queuedOn={execution.event.createdAt}
            eventId={execution.event.id}
          />
        ))
      }
    </div>
  );
};