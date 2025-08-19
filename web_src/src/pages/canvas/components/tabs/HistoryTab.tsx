import { ExecutionTimeline } from '../ExecutionTimeline';
import { ExecutionWithEvent } from "../../store/types";

interface HistoryTabProps {
  allExecutions: ExecutionWithEvent[];
  organizationId: string;
}

export const HistoryTab = ({ allExecutions, organizationId }: HistoryTabProps) => {
  return (
    <div className="p-6">
      <h3 className="font-bold text-left text-sm text-gray-500 dark:text-gray-400 uppercase tracking-wide">Historical Runs ({allExecutions.length})</h3>

      <div className="mb-8 mt-5">
        <ExecutionTimeline
          executions={allExecutions}
          organizationId={organizationId}
        />
      </div>
    </div>
  );
};