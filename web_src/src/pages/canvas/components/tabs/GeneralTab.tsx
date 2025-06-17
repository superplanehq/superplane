import { StageWithEventQueue } from "../../store/types";
import { SuperplaneExecution } from "@/api-client";
import { StageOverviewCard } from '../StageOverviewCard';
import { EventSection } from '../EventSection';
import { ExecutionTimeline } from '../ExecutionTimeline';
import { SuperplaneStageEvent } from "@/api-client";

interface GeneralTabProps {
  selectedStage: StageWithEventQueue;
  pendingEvents: SuperplaneStageEvent[];
  waitingEvents: SuperplaneStageEvent[];
  processedEvents: SuperplaneStageEvent[];
  allExecutions: SuperplaneExecution[];
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  executionRunning: boolean;
}

export const GeneralTab = ({
  selectedStage,
  pendingEvents,
  waitingEvents,
  processedEvents,
  allExecutions,
  approveStageEvent,
  executionRunning
}: GeneralTabProps) => {
  return (
    <div className="p-6 space-y-6">
      {/* Stage Overview Card */}
      <StageOverviewCard
        totalEvents={selectedStage.queue?.length || 0}
        pendingCount={pendingEvents.length}
        waitingCount={waitingEvents.length}
        processedCount={processedEvents.length}
      />

      {/* Pending Runs Section */}
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

      {/* Recent Activity */}
      <ExecutionTimeline 
        executions={allExecutions.slice(0, 5)} 
        title="Recent Activity"
      />
    </div>
  );
};