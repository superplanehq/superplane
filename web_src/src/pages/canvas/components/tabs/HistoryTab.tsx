import { ExecutionTimeline } from '../ExecutionTimeline';
import { ExecutionWithEvent } from "../../store/types";

interface HistoryTabProps {
  allExecutions: ExecutionWithEvent[];
}

export const HistoryTab = ({ allExecutions }: HistoryTabProps) => {
  return (
    <div className="p-6">
      <div className="mb-8">
        <ExecutionTimeline
          executions={allExecutions}
        />
      </div>
    </div>
  );
};