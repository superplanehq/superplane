import { StrictMode, useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useLocation, useNavigate } from "react-router-dom";
import { FlowRenderer } from "./components/FlowRenderer";
import { useCanvasStore } from "./store/canvasStore";
import { useWebsocketEvents } from "./hooks/useWebsocketEvents";
import { superplaneDescribeCanvas, superplaneListStages, superplaneListEventSources, superplaneListStageEvents, SuperplaneStageEvent, superplaneListConnectionGroups, superplaneListEvents, SuperplaneStage, SuperplaneEventSource, SuperplaneCanvas, SuperplaneConnectionGroup, SuperplaneEvent } from "@/api-client";
import { ConnectionGroupWithEvents, EventSourceWithEvents, StageWithEventQueue } from "./store/types";
import { Sidebar } from "./components/SideBar";
import { EventSourceSidebar } from "./components/EventSourceSidebar";
import { ComponentSidebar } from "./components/ComponentSidebar";
import { CanvasNavigation, CanvasNavigationContent, type CanvasView } from "../../components/CanvasNavigation";
import { useNodeHandlers } from "./utils/nodeHandlers";
import { NodeType } from "./utils/nodeFactories";
import { withOrganizationHeader } from "../../utils/withOrganizationHeader";
import { useAutoLayout } from "./hooks/useAutoLayout";

export function Canvas() {
  const { organizationId, canvasId } = useParams<{ organizationId: string, canvasId: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { initialize, selectedStageId, cleanSelectedStageId, selectedEventSourceId, cleanSelectedEventSourceId, editingStageId, stages, eventSources, approveStageEvent, fitViewNode, lockedNodes, setFocusedNodeId, setNodes } = useCanvasStore();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isComponentSidebarOpen, setIsComponentSidebarOpen] = useState(true);
  const [canvasName, setCanvasName] = useState<string>('');

  const getActiveViewFromHash = (): CanvasView => {
    const hash = location.hash.substring(1);
    switch (hash) {
      case 'secrets':
        return 'secrets';
      case 'integrations':
        return 'integrations';
      case 'members':
        return 'members';
      case 'delete':
        return 'delete';
      default:
        return 'editor';
    }
  };

  const activeView = getActiveViewFromHash();

  const handleViewChange = (view: CanvasView) => {
    if (view === 'editor') {
      navigate(location.pathname);
    } else {
      navigate(`${location.pathname}#${view}`);
    }
  };

  useWebsocketEvents(canvasId!, organizationId!);

  const { handleAddNode } = useNodeHandlers(canvasId || '');

  const { applyElkAutoLayout } = useAutoLayout();

  const selectedStage = useMemo(() => stages.find(stage => stage.metadata!.id === selectedStageId), [stages, selectedStageId]);
  const selectedEventSource = useMemo(() => eventSources.find(eventSource => eventSource.metadata!.id === selectedEventSourceId), [eventSources, selectedEventSourceId]);

  const fetchCanvasBasicData = useCallback(async () => {
    const [canvasResponse, stagesResponse, connectionGroupsResponse, eventSourcesResponse] = await Promise.all([
      superplaneDescribeCanvas(
        withOrganizationHeader({
          path: { id: canvasId },
          query: { organizationId: organizationId }
        })
      ),
      superplaneListStages(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
        })
      ),
      superplaneListConnectionGroups(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId },
        })
      ),
      superplaneListEventSources(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId }
        })
      )
    ]);

    if (!canvasResponse.data?.canvas) {
      throw new Error('Failed to fetch canvas data');
    }
    if (!stagesResponse.data?.stages) {
      throw new Error('Failed to fetch stages data');
    }
    if (!connectionGroupsResponse.data?.connectionGroups) {
      throw new Error('Failed to fetch connection groups data');
    }
    if (!eventSourcesResponse.data?.eventSources) {
      throw new Error('Failed to fetch event sources data');
    }

    return {
      canvas: canvasResponse.data.canvas,
      stages: stagesResponse.data.stages,
      connectionGroups: connectionGroupsResponse.data.connectionGroups,
      eventSources: eventSourcesResponse.data.eventSources
    };
  }, [canvasId, organizationId]);

  const fetchStageEvents = useCallback(async (stages: SuperplaneStage[]) => {
    const stageEventsPromises = stages.map(stage =>
      superplaneListStageEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId!, stageIdOrName: stage.metadata!.id! }
        })
      ).then(response => ({
        stage,
        stageEvents: response.data?.events || []
      }))
    );

    return Promise.all(stageEventsPromises);
  }, [canvasId]);

  const fetchStagePlainEvents = useCallback(async (stages: SuperplaneStage[]) => {
    const stageEventsPromises = stages.map(stage =>
      superplaneListEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId! },
          query: {
            sourceType: 'EVENT_SOURCE_TYPE_STAGE' as const,
            sourceId: stage.metadata?.id
          }
        })
      ).then(response => ({
        stage,
        events: response.data?.events || []
      }))
    );

    return Promise.all(stageEventsPromises);
  }, [canvasId]);

  const fetchConnectionGroupEvents = useCallback(async (connectionGroups: SuperplaneConnectionGroup[]) => {
    const connectionGroupEventsPromises = connectionGroups.map(connectionGroup =>
      superplaneListEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId! },
          query: {
            sourceType: 'EVENT_SOURCE_TYPE_CONNECTION_GROUP' as const,
            sourceId: connectionGroup.metadata?.id
          }
        })
      ).then(response => ({
        connectionGroup,
        events: response.data?.events || []
      }))
    );

    return Promise.all(connectionGroupEventsPromises);
  }, [canvasId]);

  const fetchEventSourceEvents = useCallback(async (eventSources: SuperplaneEventSource[]) => {
    const eventSourceEventsPromises = eventSources.map(eventSource =>
      superplaneListEvents(
        withOrganizationHeader({
          path: { canvasIdOrName: canvasId! },
          query: {
            sourceType: 'EVENT_SOURCE_TYPE_EVENT_SOURCE' as const,
            sourceId: eventSource.metadata?.id
          }
        })
      ).then(response => ({
        eventSource,
        events: response.data?.events || []
      }))
    );

    return Promise.all(eventSourceEventsPromises);
  }, [canvasId]);

  const processAndInitializeStore = useCallback((
    canvas: SuperplaneCanvas,
    connectionGroupPlainEventsResults: Array<{ connectionGroup: SuperplaneConnectionGroup; events: SuperplaneEvent[] }>,
    stageEventsResults: Array<{ stage: SuperplaneStage; stageEvents: SuperplaneStageEvent[] }>,
    stagePlainEventsResults: Array<{ stage: SuperplaneStage; events: SuperplaneEvent[] }>,
    eventSourceEventsResults: Array<{ eventSource: SuperplaneEventSource; events: SuperplaneEvent[] }>
  ) => {
    const allStageEvents: Record<string, SuperplaneStageEvent> = {};

    const stagePlainEventsResultsByStageId = stagePlainEventsResults.reduce((acc, { stage, events }) => {
      acc[stage.metadata!.id!] = events;
      return acc;
    }, {} as Record<string, SuperplaneEvent[]>);

    const stagesWithQueues: StageWithEventQueue[] = stageEventsResults.map(({ stage, stageEvents }) => {
      for (const stageEvent of stageEvents) {
        allStageEvents[stageEvent.id!] = stageEvent;
      }

      return {
        ...stage,
        queue: stageEvents,
        events: stagePlainEventsResultsByStageId[stage.metadata!.id!] || []
      };
    });

    const eventSourcesWithEvents: EventSourceWithEvents[] = eventSourceEventsResults.map(({ eventSource, events }) => ({
      ...eventSource,
      events,
      eventFilters: events
    }));

    const connectionGroupsWithEvents: ConnectionGroupWithEvents[] = connectionGroupPlainEventsResults.map(({ connectionGroup, events }) => ({
      ...connectionGroup,
      events,
    }));

    const initialData = {
      canvas: canvas || {},
      stages: stagesWithQueues,
      eventSources: eventSourcesWithEvents,
      connectionGroups: connectionGroupsWithEvents,
      handleEvent: () => { },
      removeHandleEvent: () => { },
      pushEvent: () => { },
    };

    initialize(initialData);
  }, [initialize]);

  useEffect(() => {
    if (!canvasId || !organizationId) {
      if (!canvasId) {
        setError("No canvas ID provided");
        setIsLoading(false);
      }
      return;
    }

    const fetchCanvasData = async () => {
      try {
        setIsLoading(true);

        const basicData = await fetchCanvasBasicData();

        setCanvasName(basicData.canvas.metadata?.name || 'Unknown Canvas');

        const [stageEventsResults, stagePlainEventsResults, connectionGroupEventsResults, eventSourceEventsResults] = await Promise.all([
          fetchStageEvents(basicData.stages),
          fetchStagePlainEvents(basicData.stages),
          fetchConnectionGroupEvents(basicData.connectionGroups),
          fetchEventSourceEvents(basicData.eventSources)
        ]);

        processAndInitializeStore(
          basicData.canvas,
          connectionGroupEventsResults,
          stageEventsResults,
          stagePlainEventsResults,
          eventSourceEventsResults
        );

        setIsLoading(false);

      } catch (err) {
        console.error('Error fetching canvas data:', err);
        setError('Failed to load canvas data');
        setIsLoading(false);
      }
    };

    fetchCanvasData();
  }, [canvasId, organizationId, fetchCanvasBasicData, fetchStageEvents, fetchStagePlainEvents, fetchConnectionGroupEvents, fetchEventSourceEvents, processAndInitializeStore]);

  if (isLoading) {
    return <div className="loading-state">Loading canvas...</div>;
  }

  if (error) {
    return <div className="error-state">Error: {error}</div>;
  }

  const handleAddNodeByType = async (nodeType: NodeType, executorType?: string, eventSourceType?: string) => {
    try {
      const config = getNodeConfig(nodeType, executorType, eventSourceType);
      const nodeId = handleAddNode(nodeType, config);

      setFocusedNodeId(nodeId);

      setTimeout(() => {
        const currentNodes = useCanvasStore.getState().nodes;
        const updatedNodes = currentNodes.map(node => ({
          ...node,
          selected: node.id === nodeId
        }));
        setNodes(updatedNodes);
      }, 10);

      if (lockedNodes) {
        setTimeout(async () => {
          const { nodes: latestNodes, edges: latestEdges } = useCanvasStore.getState();
          await applyElkAutoLayout(latestNodes, latestEdges);
          setTimeout(() => {
            fitViewNode(nodeId);
          }, 200);
        }, 50);
      } else {
        setTimeout(() => {
          fitViewNode(nodeId);
        }, 100);
      }
    } catch (error) {
      console.error(`Failed to add node of type ${nodeType}:`, error);
    }
  };

  const getNodeConfig = (nodeType: NodeType, executorType?: string, eventSourceType?: string) => {
    switch (nodeType) {
      case 'stage':
        return executorType ? {
          name: '',
          executorType
        } : undefined;

      case 'event_source':
        return eventSourceType ? {
          name: '',
          eventSourceType
        } : undefined;

      default:
        return undefined;
    }
  };

  return (
    <StrictMode>
      {/* Canvas Navigation */}
      <div className="h-[100vh] overflow-hidden">

        <CanvasNavigation
          canvasName={canvasName}
          activeView={activeView}
          onViewChange={handleViewChange}
          organizationId={organizationId!}
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
            {selectedEventSource && <EventSourceSidebar selectedEventSource={selectedEventSource} onClose={() => cleanSelectedEventSourceId()} />}
          </div>
        ) : (
          <div className="h-[calc(100%-2.7rem)] p-6 bg-zinc-50 dark:bg-zinc-950" >
            <CanvasNavigationContent canvasId={canvasId!} activeView={activeView} organizationId={organizationId!} />
          </div>
        )}
      </div>
    </StrictMode>
  );
}