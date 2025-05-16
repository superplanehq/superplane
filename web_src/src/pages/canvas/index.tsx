import { StrictMode, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { FlowRenderer } from "./components/FlowRenderer";
import { useCanvasStore } from "./store/canvasStore";
import { superplaneDescribeCanvas, superplaneListStages, superplaneListEventSources } from "../../api-client";
import { Stage, EventSource } from "./types";
import { ConnectionFilterOperator, ConditionType, RunTemplateType } from "./types/flow";

// No props needed as we'll get the ID from the URL params

export function Canvas() {
  // Get the canvas ID from the URL params
  const { id } = useParams<{ id: string }>();
  const { initialize, setupLiveViewHandlers } = useCanvasStore();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
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
        
        // Map API response to internal Stage type
        const mappedStages: Stage[] = (stagesResponse.data?.stages || []).map(apiStage => {
          // Create proper Connection objects from API connections if they exist
          const connections = apiStage.connections?.map(conn => {
            // Map the string filter operator to the enum value
            let filterOp = ConnectionFilterOperator.AND; // Default to AND
            if (conn.filterOperator === 'FILTER_OPERATOR_OR') {
              filterOp = ConnectionFilterOperator.OR;
            }
            
            return {
              name: conn.name || '',
              type: conn.type || '',
              filters: conn.filters?.map(f => JSON.stringify(f)) || [],
              filter_operator: filterOp
            };
          }) || [];
          
          // Map API conditions to internal Condition type
          const conditions = apiStage.conditions?.map(cond => {
            // Map the condition type
            const condType = cond.type === 'CONDITION_TYPE_APPROVAL' ? 
              ConditionType.APPROVAL : ConditionType.TIME_WINDOW;
            
            return {
              type: condType,
              approval: cond.approval ? {
                count: cond.approval.count || 0
              } : { count: 0 },
              time_window: cond.timeWindow ? {
                start: cond.timeWindow.start || '',
                end: cond.timeWindow.end || '',
                timezone: 'UTC', // Default timezone since our type requires it
                week_days: cond.timeWindow?.weekDays || []
              } : {
                start: '',
                end: '',
                timezone: 'UTC',
                week_days: []
              }
            };
          }) || [];
          
          return {
            id: apiStage.id || '',
            name: apiStage.name || '',
            timestamp: apiStage.createdAt,
            connections,
            conditions,
            // Add any other required fields for Stage type
            status: '',
            labels: [],
            icon: '',
            queue: [],
            run_template: {
              type: RunTemplateType.SEMAPHORE,
              semaphore: {
                project_id: '',
                branch: '',
                pipeline_file: '',
                task_id: '',
                parameters: []
              }
            }
          };
        });
        
        // Map API response to internal EventSource type
        const mappedEventSources: EventSource[] = (eventSourcesResponse.data?.eventSources || []).map(apiEventSource => ({
          id: apiEventSource.id || '',
          name: apiEventSource.name || '',
          timestamp: apiEventSource.createdAt,
          type: 'default',  // Default type since it's required by our internal type
          release: '',      // Required by our internal type
          // Map other properties as needed
          ...apiEventSource
        }));
        
        // Initialize the store with the mapped data
        const initialData = {
          canvas: canvasResponse.data?.canvas || {},
          stages: mappedStages,
          event_sources: mappedEventSources,
          handleEvent: () => {},
          removeHandleEvent: () => {},
          pushEvent: () => {},
        };
        
        initialize(initialData);
        
        // Set up LiveView event handlers and get cleanup function
        const cleanup = setupLiveViewHandlers(initialData);
        setIsLoading(false);
        
        // Return cleanup function to remove event handlers on unmount
        return cleanup;
      } catch (err) {
        console.error('Error fetching canvas data:', err);
        setError('Failed to load canvas data');
        setIsLoading(false);
      }
    };
    
    fetchCanvasData();
  }, [id, initialize, setupLiveViewHandlers]);
  
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
