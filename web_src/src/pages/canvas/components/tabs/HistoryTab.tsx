import { SuperplaneExecution } from "@/api-client";
import { ExecutionTimeline } from '../ExecutionTimeline';

interface HistoryTabProps {
  allExecutions: SuperplaneExecution[];
}

export const HistoryTab = ({ allExecutions }: HistoryTabProps) => {
  return (
    <div className="p-6">
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-2">Run History</h2>
        <p className="text-gray-600">Complete execution history for this stage</p>
      </div>

      <ExecutionTimeline 
        executions={allExecutions} 
        title="Execution Timeline"
        showStats={true}
      />
    </div>
  );
};