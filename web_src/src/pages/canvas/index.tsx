import { StrictMode, useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { FlowRenderer } from "./components/FlowRenderer";
import { useCanvasStore } from "./store/canvasStore";
import { useWebsocketEvents } from "./hooks/useWebsocketEvents";
import { superplaneDescribeCanvas, superplaneListStages, superplaneListEventSources, superplaneListStageEvents, SuperplaneStageEvent } from "@/api-client";
import { EventSourceWithEvents, StageWithEventQueue } from "./store/types";
import { Sidebar } from "./components/SideBar";

// No props needed as we'll get the ID from the URL params

export function Canvas() {
  // Get the canvas ID from the URL params
  const { orgId, canvasId } = useParams<{ orgId: string, canvasId: string }>();
  const { initialize, selectedStageId, cleanSelectedStageId, stages, approveStageEvent } = useCanvasStore();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Custom hook for setting up event handlers - must be called at top level
  useWebsocketEvents(canvasId!);

  const selectedStage = useMemo(() => stages.find(stage => stage.metadata!.id === selectedStageId), [stages, selectedStageId]);

  useEffect(() => {
    // Return early if no ID is available
    if (!canvasId) {
      setError("No canvas ID provided");
      setIsLoading(false);
      return;
    }

    const fetchCanvasData = async () => {
      try {
        setIsLoading(true);

        // Fetch canvas details
        const canvasResponse = await superplaneDescribeCanvas({
          path: { id: canvasId },
          query: { organizationId: orgId }
        });

        if (!canvasResponse.data?.canvas) {
          throw new Error('Failed to fetch canvas data');
        }

        // Fetch stages for the canvas
        const stagesResponse = await superplaneListStages({
          path: { canvasIdOrName: canvasId },
        });

        // Check if stages data was fetched successfully
        if (!stagesResponse.data?.stages) {
          throw new Error('Failed to fetch stages data');
        }

        // Fetch event sources for the canvas
        const eventSourcesResponse = await superplaneListEventSources({
          path: { canvasIdOrName: canvasId }
        });

        // Check if event sources data was fetched successfully
        if (!eventSourcesResponse.data?.eventSources) {
          throw new Error('Failed to fetch event sources data');
        }

        // Use the API stages directly with minimal adaptation
        const mappedStages = stagesResponse.data?.stages || [];
        
        // Collect all events from all stages
        const allEvents: Record<string, SuperplaneStageEvent> = {};
        const stagesWithQueues: StageWithEventQueue[] = [];

        // Fetch events for each stage
        for (const stage of mappedStages) {
          const stageEventsResponse = await superplaneListStageEvents({
            path: { canvasIdOrName: canvasId!, stageIdOrName: stage.metadata!.id! }
          });

          const stageEvents = stageEventsResponse.data?.events || [];
          
          // Add events to the collection
          for (const event of stageEvents) {
            allEvents[event.id!] = event;
          }

          stagesWithQueues.push({
            ...stage,
            queue: stageEvents
          });
        }

        // Group events by source ID
        const eventsBySourceId = Object.values(allEvents).reduce((acc, event) => {
          const sourceId = event.sourceId;
          if (sourceId) {
            if (!acc[sourceId]) {
              acc[sourceId] = [];
            }
            acc[sourceId].push(event);
          }
          return acc;
        }, {} as Record<string, SuperplaneStageEvent[]>);

        // Assign events to their corresponding event sources
        const eventSourcesWithEvents: EventSourceWithEvents[] = (eventSourcesResponse.data?.eventSources || []).map(eventSource => ({
          ...eventSource,
          events: eventSource.metadata?.id ? eventsBySourceId[eventSource.metadata.id] : []
        }));

        // Initialize the store with the mapped data
        const initialData = {
          canvas: canvasResponse.data?.canvas || {},
          stages: stagesWithQueues,
          event_sources: eventSourcesWithEvents,
          handleEvent: () => { },
          removeHandleEvent: () => { },
          pushEvent: () => { },
        };

        initialize(initialData);
        setIsLoading(false);

      } catch (err) {
        console.error('Error fetching canvas data:', err);
        setError('Failed to load canvas data');
        setIsLoading(false);
      }
    };

    fetchCanvasData();
  }, [canvasId, initialize, orgId]);

  if (isLoading) {
    return <div className="loading-state">Loading canvas...</div>;
  }

  if (error) {
    return <div className="error-state">Error: {error}</div>;
  }

  return (
    <StrictMode>
      <FlowRenderer />
      {selectedStage && <Sidebar approveStageEvent={approveStageEvent} selectedStage={selectedStage} onClose={() => cleanSelectedStageId()} />}
    </StrictMode>
  );
}
