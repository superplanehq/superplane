import { SuperplaneStage } from "@/api-client";
import { ExecutionTimeline } from '../ExecutionTimeline';
import { ExecutionWithEvent } from "../../store/types";

interface HistoryTabProps {
  allExecutions: ExecutionWithEvent[];
  selectedStage: SuperplaneStage;
}

export const HistoryTab = ({ allExecutions, selectedStage }: HistoryTabProps) => {
  return (
    <div className="p-6">
      <div className="mb-8">
        <ExecutionTimeline
          selectedStage={selectedStage}
          executions={allExecutions} 
        />
      </div>
    </div>
  );
};