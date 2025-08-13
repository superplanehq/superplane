import { ExecutionWithEvent } from "../store/types";
import { RunItem } from "./tabs/RunItem";

interface ExecutionTimelineProps {
  executions: ExecutionWithEvent[];
}

export const ExecutionTimeline = ({
  executions,
}: ExecutionTimelineProps) => {
  if (executions.length === 0) {
    return (
      <div className="bg-gray-50 rounded-lg">
        <div className="p-4">
          <div className="text-center py-6 text-gray-500">
            <div className="text-sm">No recent activity</div>
          </div>
        </div>
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
            title={execution.id || 'Execution'}
            inputs={mapExecutionEventInputs(execution)}
            outputs={mapExecutionOutputs(execution)}
            state={execution.state || 'STATE_UNKNOWN'}
            result={execution.result || 'RESULT_UNKNOWN'}
            timestamp={execution.createdAt || new Date().toISOString()}
            executionDuration={formatDuration(execution.startedAt, execution.finishedAt)}
          />
        ))
      }
    </div>
  );
};