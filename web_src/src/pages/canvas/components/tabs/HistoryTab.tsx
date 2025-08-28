import { ExecutionWithEvent, StageWithEventQueue } from "../../store/types";
import { SuperplaneStageEvent, SuperplaneEvent } from "@/api-client";
import MessageItem from '../MessageItem';
import { RunItem } from './RunItem';
import { useMemo } from 'react';
import { useOrganizationUsersForCanvas } from '@/hooks/useCanvasData';
import {
  formatDuration,
  getMinApprovedAt,
  getApprovalsNames,
  mapExecutionOutputs,
  mapExecutionEventInputs,
  createUserDisplayNames
} from '../../utils/stageEventUtils';

interface HistoryTabProps {
  allExecutions: ExecutionWithEvent[];
  selectedStage: StageWithEventQueue;
  allStageEvents: SuperplaneStageEvent[];
  organizationId: string;
  approveStageEvent: (stageEventId: string, stageId: string) => void;
  connectionEventsById?: Record<string, SuperplaneEvent>;
  eventsByExecutionId?: Record<string, SuperplaneEvent>;
}

export const HistoryTab = ({ allExecutions, selectedStage, allStageEvents, organizationId, approveStageEvent, connectionEventsById, eventsByExecutionId }: HistoryTabProps) => {
  // Create a unified timeline by merging executions, stage events, and discarded events
  type TimelineItem = {
    type: 'execution' | 'stage_event' | 'discarded_event';
    timestamp: string;
    data: ExecutionWithEvent | SuperplaneStageEvent | SuperplaneEvent;
  };

  const { data: orgUsers = [] } = useOrganizationUsersForCanvas(organizationId);


  const createTimeline = (): TimelineItem[] => {
    const items: TimelineItem[] = [];

    // Add executions
    allExecutions.forEach(execution => {
      if (execution?.createdAt) {
        items.push({
          type: 'execution',
          timestamp: execution.createdAt,
          data: execution
        });
      }
    });

    // Add orphaned stage events (events without executions)
    const executionEventIds = new Set(allExecutions.map(exec => exec.event?.id).filter(Boolean));
    const orphanedStageEvents = allStageEvents.filter(event => !executionEventIds.has(event.id));

    orphanedStageEvents.forEach(event => {
      if (event?.createdAt) {
        items.push({
          type: 'stage_event',
          timestamp: event.createdAt,
          data: event
        });
      }
    });

    // Add discarded events from connectionEventsById
    if (connectionEventsById) {
      Object.values(connectionEventsById).forEach(plainEvent => {
        if (plainEvent?.state === 'STATE_DISCARDED' && plainEvent?.receivedAt && plainEvent.sourceId !== selectedStage.metadata?.id) {
          items.push({
            type: 'discarded_event',
            timestamp: plainEvent.receivedAt,
            data: plainEvent
          });
        }
      });
    }

    return items.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  };

  const userDisplayNames = useMemo(() => createUserDisplayNames(orgUsers), [orgUsers]);


  const timeline = createTimeline();

  return (
    <div className="p-6">
      <h3 className="font-bold text-left text-sm text-gray-500 dark:text-gray-400 uppercase tracking-wide">
        History ({timeline.length} items)
      </h3>

      <div className="mb-8 mt-5 space-y-3">
        {timeline.length === 0 ? (
          <div className="text-center py-8 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
            <span className="material-symbols-outlined select-none inline-flex items-center justify-center !w-16 !h-16 !text-[64px] !leading-16 mx-auto text-zinc-400 dark:text-zinc-500 mb-3" aria-hidden="true" style={{ fontVariationSettings: "FILL 0, wght 400, GRAD 0, opsz 24" }}>history</span>
            <p className="text-zinc-600 dark:text-zinc-400 max-w-md mx-auto mb-6 !text-sm text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">No history available</p>
          </div>
        ) : (
          timeline.map((item) => {
            if (item.type === 'execution') {
              const execution = item.data as ExecutionWithEvent;
              const sourceEvent = connectionEventsById?.[execution.event.eventId || ''];
              const emmitedEvent = eventsByExecutionId?.[execution.id || ''];

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
            }
            if (item.type === 'stage_event') {
              const stageEvent = item.data as SuperplaneStageEvent;
              const sourceEvent = connectionEventsById?.[stageEvent.eventId || ''];
              const plainEventPayload = connectionEventsById?.[stageEvent.eventId || '']?.raw;
              const plainEventHeaders = connectionEventsById?.[stageEvent.eventId || '']?.headers;
              return (
                <MessageItem
                  key={stageEvent.id}
                  event={stageEvent}
                  selectedStage={selectedStage}
                  executionRunning={false}
                  onApprove={stageEvent.state === 'STATE_WAITING' ? (eventId) => approveStageEvent(eventId, selectedStage.metadata!.id!) : undefined}
                  plainEventPayload={plainEventPayload}
                  plainEventHeaders={plainEventHeaders}
                  sourceEvent={sourceEvent}
                />
              );
            }
            if (item.type === 'discarded_event') {
              const plainEvent = item.data as SuperplaneEvent;
              // Get payload from the plain event's raw data
              const plainEventPayload = plainEvent.raw;
              const plainEventHeaders = plainEvent.headers

              return (
                <MessageItem
                  key={plainEvent.id}
                  event={plainEvent}
                  executionRunning={false}
                  plainEventPayload={plainEventPayload}
                  plainEventHeaders={plainEventHeaders}
                />
              );
            }
            return null;
          })
        )}
      </div>
    </div>
  );
};