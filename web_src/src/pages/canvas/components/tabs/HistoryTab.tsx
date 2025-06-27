import { SuperplaneExecution } from "@/api-client";
import { ExecutionTimeline } from '../ExecutionTimeline';

interface HistoryTabProps {
  allExecutions: SuperplaneExecution[];
}

export const HistoryTab = ({ allExecutions }: HistoryTabProps) => {
  return (
    <div className="p-6">
      <div className="mb-8">
        <ExecutionTimeline 
          executions={allExecutions} 
          title="Execution Timeline"
        />
      </div>
    </div>
  );
};