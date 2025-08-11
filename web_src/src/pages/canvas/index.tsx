import { StrictMode, useEffect, useMemo, useState } from "react";
import { useParams, useLocation, useNavigate } from "react-router-dom";
import { FlowRenderer } from "./components/FlowRenderer";
import { useCanvasStore } from "./store/canvasStore";
import { useWebsocketEvents } from "./hooks/useWebsocketEvents";
import { superplaneDescribeCanvas, superplaneListStages, superplaneListEventSources, superplaneListStageEvents, SuperplaneStageEvent, superplaneListConnectionGroups } from "@/api-client";
import { EventSourceWithEvents, StageWithEventQueue } from "./store/types";
import { Sidebar } from "./components/SideBar";
import { ComponentSidebar } from "./components/ComponentSidebar";
import { CanvasNavigation } from "../../components/CanvasNavigation";
import { SettingsPage } from "../../components/SettingsPage";
import { useNodeHandlers } from "./utils/nodeHandlers";
import { NodeType } from "./utils/nodeFactories";
import { User } from "../../stores/userStore";

// No props needed as we'll get the ID from the URL params

export function Canvas() {
  // Get the canvas ID from the URL params
  const { canvasId } = useParams<{ canvasId: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { initialize, selectedStageId, cleanSelectedStageId, editingStageId, stages, approveStageEvent } = useCanvasStore();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isComponentSidebarOpen, setIsComponentSidebarOpen] = useState(true);
  const [canvasName, setCanvasName] = useState<string>('');
  const [user, setUser] = useState<User | null>(null);

  // Determine active view from URL hash or default to editor
  const activeView = location.hash.startsWith('#settings') ? 'settings' : 'editor';

  // Handle view changes by updating URL hash
  const handleViewChange = (view: 'editor' | 'settings') => {
    if (view === 'settings') {
      navigate(`${location.pathname}#settings`);
    } else {
      navigate(location.pathname);
    }
  };

  // Custom hook for setting up event handlers - must be called at top level
  useWebsocketEvents(canvasId!);

  // Use the modular node handlers
  const { handleAddNode } = useNodeHandlers(canvasId || '');

  const selectedStage = useMemo(() => stages.find(stage => stage.metadata!.id === selectedStageId), [stages, selectedStageId]);

  // Fetch user data first
  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetch('/api/v1/user/profile', {
          method: 'GET',
          credentials: 'include',
          headers: {
            'Content-Type': 'application/json',
          },
        });
        
        if (response.ok) {
          const userData = await response.json();
          setUser(userData);
        }
      } catch (error) {
        console.error('Failed to fetch user:', error);
        setError("Failed to authenticate user");
        setIsLoading(false);
      }
    };

    fetchUser();
  }, []);

  useEffect(() => {
    // Return early if no canvas ID or user is available
    if (!canvasId || !user?.organization_id) {
      if (!canvasId) {
        setError("No canvas ID provided");
        setIsLoading(false);
      }
      return;
    }

    const fetchCanvasData = async () => {
      try {
        setIsLoading(true);

        // Fetch canvas details
        const canvasResponse = await superplaneDescribeCanvas({
          path: { id: canvasId },
          query: { organizationId: user.organization_id }
        });

        if (!canvasResponse.data?.canvas) {
          throw new Error('Failed to fetch canvas data');
        }

        // Store canvas name for navigation
        setCanvasName(canvasResponse.data.canvas.metadata?.name || 'Unknown Canvas');

        // Fetch stages for the canvas
        const stagesResponse = await superplaneListStages({
          path: { canvasIdOrName: canvasId },
        });

        // Check if stages data was fetched successfully
        if (!stagesResponse.data?.stages) {
          throw new Error('Failed to fetch stages data');
        }

        // Fetch connection groups for the canvas
        const connectionGroupsResponse = await superplaneListConnectionGroups({
          path: { canvasIdOrName: canvasId },
        });

        if (!connectionGroupsResponse.data?.connectionGroups) {
          throw new Error('Failed to fetch connection groups data');
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
          eventSources: eventSourcesWithEvents,
          connectionGroups: connectionGroupsResponse.data?.connectionGroups || [],
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
  }, [canvasId, initialize, user?.organization_id]);

  if (isLoading) {
    return <div className="loading-state">Loading canvas...</div>;
  }

  if (error) {
    return <div className="error-state">Error: {error}</div>;
  }

  const handleAddNodeByType = (nodeType: NodeType, executorType?: string, eventSourceType?: string) => {
    try {
      const config = getNodeConfig(nodeType, executorType, eventSourceType);
      handleAddNode(nodeType, config);
    } catch (error) {
      console.error(`Failed to add node of type ${nodeType}:`, error);
    }
  };

  const getNodeConfig = (nodeType: NodeType, executorType?: string, eventSourceType?: string) => {
    const stageNames = {
      semaphore: 'Semaphore Stage',
      github: 'GitHub Stage',
      http: 'HTTP Stage'
    };

    const eventNames = {
      webhook: 'Webhook Event Source',
      semaphore: 'Semaphore Event Source',
      github: 'GitHub Event Source'
    };

    switch (nodeType) {
      case 'stage':
        return executorType ? {
          name: stageNames[executorType as keyof typeof stageNames] || 'New Stage',
          executorType
        } : undefined;

      case 'event_source':
        return eventSourceType ? {
          name: eventNames[eventSourceType as keyof typeof eventNames] || 'New Event Source',
          eventSourceType
        } : undefined;

      default:
        return undefined;
    }
  };


  return (
    <StrictMode>
      {/* Canvas Navigation */}
      <div className="h-[100vh]">

        <CanvasNavigation
          canvasId={canvasId!}
          canvasName={canvasName}
          activeView={activeView}
          onViewChange={handleViewChange}
          organizationId={user?.organization_id!}
        />

        {/* Content based on active view */}
        {activeView === 'editor' ? (
          <div className="relative" style={{ height: "calc(100vh - 2.6rem)", overflow: "hidden" }}>
            <ComponentSidebar
              isOpen={isComponentSidebarOpen}
              onClose={() => setIsComponentSidebarOpen(false)}
              onNodeAdd={(nodeType: NodeType, executorType?: string, eventSourceType?: string) => {
                handleAddNodeByType(nodeType, executorType, eventSourceType);
              }}
            />

            {/* Toggle Button for ComponentSidebar - Only show when closed */}
            {!isComponentSidebarOpen && (
              <button
                onClick={() => setIsComponentSidebarOpen(true)}
                className="fixed top-16 left-4 z-30 px-4 py-2 bg-white border border-gray-300 rounded-md shadow-md hover:bg-gray-50 transition-all duration-300 flex items-center gap-2"
                title="Open Components"
              >
                <span className="text-medium font-semibold text-gray-700">Components</span>
                <span style={{ fontSize: '1.2rem' }} className="material-symbols-outlined text-gray-600 -scale-x-100">menu_open</span>
              </button>
            )}

            <FlowRenderer />
            {selectedStage && !editingStageId && <Sidebar approveStageEvent={approveStageEvent} selectedStage={selectedStage} onClose={() => cleanSelectedStageId()} />}
          </div>
        ) : (
          <div className="h-[calc(100%-2.7rem)]" >
            <SettingsPage organizationId={user?.organization_id!} />
          </div>
        )}
      </div>
    </StrictMode>
  );
}