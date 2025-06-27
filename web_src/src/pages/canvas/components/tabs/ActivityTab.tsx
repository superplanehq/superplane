import { StageWithEventQueue } from "../../store/types";
import { SuperplaneExecution } from "@/api-client";
import { EventSection } from '../EventSection';
import { ExecutionTimeline } from '../ExecutionTimeline';
import { SuperplaneStageEvent } from "@/api-client";

interface ActivityTabProps {
  selectedStage: StageWithEventQueue;
  pendingEvents: SuperplaneStageEvent[];
  waitingEvents: SuperplaneStageEvent[];
  allExecutions: SuperplaneExecution[];
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  executionRunning: boolean;
}

export const ActivityTab = ({
  selectedStage,
  pendingEvents,
  waitingEvents,
  allExecutions,
  approveStageEvent,
  executionRunning
}: ActivityTabProps) => {
  return (
    <div className="p-6 space-y-6">
      <p className="text-xs text-left w-full font-medium text-gray-700">RECENT RUNS</p>
      <ExecutionTimeline 
        executions={allExecutions.slice(0, 2)} 
      />
      <p className="text-xs text-left w-full font-medium text-gray-700">QUEUE ({pendingEvents.length + waitingEvents.length})</p>
      <EventSection
        title="Pending Runs"
        icon="pending"
        iconColor="text-amber-600"
        events={pendingEvents}
        variant="pending"
        maxVisible={3}
        emptyMessage="No pending runs"
        emptyIcon="pending"
      />

      {/* Waiting for Approval Section */}
      {waitingEvents.length > 0 && (
        <EventSection
          title="Waiting for Approval"
          icon="hourglass_empty"
          iconColor="text-blue-600"
          events={waitingEvents}
          variant="waiting"
          maxVisible={2}
          emptyMessage="No events waiting for approval"
          emptyIcon="hourglass_empty"
          onApprove={approveStageEvent}
          stageId={selectedStage.metadata!.id}
          executionRunning={executionRunning}
        />
      )}
    </div>
  );
};