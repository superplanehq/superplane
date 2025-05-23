import { StrictMode, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { FlowRenderer } from "./components/FlowRenderer";
import { useCanvasStore } from "./store/canvasStore";
import { useSetupEventHandlers } from "./store/handlers/setup";
import { superplaneDescribeCanvas, superplaneListStages, superplaneListEventSources, superplaneListStageEvents } from "@/api-client";
import { StageWithEventQueue } from "./store/types";

// No props needed as we'll get the ID from the URL params

export function Canvas() {
  
  // Get the canvas ID from the URL params
  const { id } = useParams<{ id: string }>();
  const { initialize, markEventHandlersAsSetup } = useCanvasStore();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Custom hook for setting up event handlers - must be called at top level
  useSetupEventHandlers(id!);
  
  useEffect(() => {
    // Return early if no ID is available
    if (!id) {
      setError("No canvas ID provided");
      setIsLoading(false);
      return;
    }
    
    const fetchCanvasData = async () => {
      try {
        setIsLoading(true);
        
        // Fetch canvas details
        const canvasResponse = await superplaneDescribeCanvas({
          path: { id }
        });
        
        // Check if canvas data was fetched successfully
        if (!canvasResponse.data?.canvas) {
          throw new Error('Failed to fetch canvas data');
        }
        
        // Fetch stages for the canvas
        const stagesResponse = await superplaneListStages({
          path: { canvasId: id }
        });
        
        // Check if stages data was fetched successfully
        if (!stagesResponse.data?.stages) {
          throw new Error('Failed to fetch stages data');
        }
        
        // Fetch event sources for the canvas
        const eventSourcesResponse = await superplaneListEventSources({
          path: { canvasId: id }
        });
        
        // Check if event sources data was fetched successfully
        if (!eventSourcesResponse.data?.eventSources) {
          throw new Error('Failed to fetch event sources data');
        }
        
        // Use the API stages directly with minimal adaptation
        const mappedStages = stagesResponse.data?.stages || [];
        
        // Initialize queues array for each stage (for the real-time events)
        // Using for...of to properly handle async/await
        const stagesWithQueues: StageWithEventQueue[] = [];
        
        for (const stage of mappedStages) {
          // fetch stage events
          const stageEventsResponse = await superplaneListStageEvents({
            path: { canvasId: id!, stageId: stage.id! }
          });
          
          stagesWithQueues.push({
            ...stage,
            queue: stageEventsResponse.data?.events || [] // Add stage events from API
          });
        }
        
        // Use the API event sources directly
        const mappedEventSources = eventSourcesResponse.data?.eventSources || [];
        
        // Initialize the store with the mapped data
        const initialData = {
          canvas: canvasResponse.data?.canvas || {},
          stages: stagesWithQueues,
          event_sources: mappedEventSources,
          handleEvent: () => {},
          removeHandleEvent: () => {},
          pushEvent: () => {},
        };
        
        initialize(initialData);
        
        // Mark event handlers as ready to be set up
        markEventHandlersAsSetup();
        setIsLoading(false);
        
        // No cleanup function needed here, it will be handled by the useSetupEventHandlers hook
      } catch (err) {
        console.error('Error fetching canvas data:', err);
        setError('Failed to load canvas data');
        setIsLoading(false);
      }
    };
    
    fetchCanvasData();
  }, [id, initialize, markEventHandlersAsSetup]);
  
  if (isLoading) {
    return <div className="loading-state">Loading canvas...</div>;
  }
  
  if (error) {
    return <div className="error-state">Error: {error}</div>;
  }
  
  return (
    <StrictMode>
        <FlowRenderer />
    </StrictMode>
  );
}
