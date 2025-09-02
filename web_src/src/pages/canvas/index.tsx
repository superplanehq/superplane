import { StrictMode, useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useLocation, useNavigate } from "react-router-dom";
import { FlowRenderer } from "./components/FlowRenderer";
import { useCanvasStore } from "./store/canvasStore";
import { useWebsocketEvents } from "./hooks/useWebsocketEvents";
import { superplaneDescribeCanvas, superplaneListStages, superplaneListEventSources, SuperplaneStageEvent, superplaneListConnectionGroups, superplaneBulkListEvents, superplaneBulkListStageEvents, SuperplaneStage, SuperplaneEventSource, SuperplaneCanvas, SuperplaneConnectionGroup, SuperplaneEvent, SuperplaneConnection, SuperplaneConnectionType } from "@/api-client";
import { ConnectionGroupWithEvents, EventSourceWithEvents, StageWithEventQueue } from "./store/types";
import { Sidebar } from "./components/SideBar";
import { EventSourceSidebar } from "./components/EventSourceSidebar";
import { ComponentSidebar } from "./components/ComponentSidebar";
import { CanvasNavigation, CanvasNavigationContent, type CanvasView } from "../../components/CanvasNavigation";
import { useNodeHandlers } from "./utils/nodeHandlers";
import { NodeType } from "./utils/nodeFactories";
import { withOrganizationHeader } from "../../utils/withOrganizationHeader";
import { useAutoLayout } from "./hooks/useAutoLayout";

const EVENTS_LIMIT = 3;
const STAGE_EVENTS_LIMIT = 2;

export function Canvas() {
  const { organizationId, canvasId } = useParams<{ organizationId: string, canvasId: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { initialize, selectedStageId, cleanSelectedStageId, selectedEventSourceId, cleanSelectedEventSourceId, editingStageId, stages, eventSources, approveStageEvent, fitViewNode, lockedNodes, setFocusedNodeId, setNodes } = useCanvasStore();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isComponentSidebarOpen, setIsComponentSidebarOpen] = useState(true);
  const [canvasName, setCanvasName] = useState<string>('');

  useEffect(() => {
    if (canvasName) {
      document.title = `${canvasName} - Superplane`;
    } else {
      document.title = 'Superplane';
    }
  }, [canvasName]);

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
    if (stages.length === 0) {
      return [];
    }

    const response = await superplaneBulkListStageEvents(
      withOrganizationHeader({
        path: { canvasIdOrName: canvasId! },
        body: {
          stages: stages.map(stage => ({
            stageIdOrName: stage.metadata!.id!
          })),
          limitPerStage: STAGE_EVENTS_LIMIT
        }
      })
    );

    const results = response.data?.results || [];

    return stages.map(stage => {
      const result = results.find(r => r.stageId === stage.metadata!.id);
      return {
        stage,
        stageEvents: result?.events || []
      };
    });
  }, [canvasId]);

  const fetchAllPlainEvents = useCallback(async (
    stages: SuperplaneStage[],
    connectionGroups: SuperplaneConnectionGroup[],
    eventSources: SuperplaneEventSource[]
  ) => {
    const sources = [
      ...stages.map(stage => ({
        sourceType: 'EVENT_SOURCE_TYPE_STAGE' as const,
        sourceId: stage.metadata!.id!,
        limitPerSource: EVENTS_LIMIT,
        entityType: 'stage' as const,
        entity: stage
      })),
      ...connectionGroups.map(connectionGroup => ({
        sourceType: 'EVENT_SOURCE_TYPE_CONNECTION_GROUP' as const,
        sourceId: connectionGroup.metadata!.id!,
        limitPerSource: 50,
        entityType: 'connectionGroup' as const,
        entity: connectionGroup
      })),
      ...eventSources.map(eventSource => ({
        sourceType: 'EVENT_SOURCE_TYPE_EVENT_SOURCE' as const,
        sourceId: eventSource.metadata!.id!,
        limitPerSource: EVENTS_LIMIT,
        entityType: 'eventSource' as const,
        entity: eventSource
      }))
    ];

    if (sources.length === 0) {
      return {
        stagePlainEventsResults: [],
        connectionGroupEventsResults: [],
        eventSourceEventsResults: []
      };
    }

    const response = await superplaneBulkListEvents(
      withOrganizationHeader({
        path: { canvasIdOrName: canvasId! },
        body: {
          sources: sources.map(({ sourceType, sourceId, limitPerSource }) => ({
            sourceType,
            sourceId,
            limitPerSource
          }))
        }
      })
    );

    const results = response.data?.results || [];
    const resultsBySourceId = results.reduce((acc, result) => {
      acc[result.sourceId || ''] = result;
      return acc;
    }, {} as Record<string, typeof results[0]>);

    const stagePlainEventsResults = sources
      .filter(s => s.entityType === 'stage')
      .map(s => ({
        stage: s.entity as SuperplaneStage,
        events: resultsBySourceId[s.sourceId]?.events || []
      }));

    const connectionGroupEventsResults = sources
      .filter(s => s.entityType === 'connectionGroup')
      .map(s => ({
        connectionGroup: s.entity as SuperplaneConnectionGroup,
        events: resultsBySourceId[s.sourceId]?.events || []
      }));

    const eventSourceEventsResults = sources
      .filter(s => s.entityType === 'eventSource')
      .map(s => ({
        eventSource: s.entity as SuperplaneEventSource,
        events: resultsBySourceId[s.sourceId]?.events || []
      }));

    return {
      stagePlainEventsResults,
      connectionGroupEventsResults,
      eventSourceEventsResults
    };
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

        const [stageEventsResults, plainEventsResults] = await Promise.all([
          fetchStageEvents(basicData.stages),
          fetchAllPlainEvents(basicData.stages, basicData.connectionGroups, basicData.eventSources)
        ]);

        const { stagePlainEventsResults, connectionGroupEventsResults, eventSourceEventsResults } = plainEventsResults;

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
  }, [canvasId, organizationId, fetchCanvasBasicData, fetchStageEvents, fetchAllPlainEvents, processAndInitializeStore]);

  if (isLoading) {
    return <div className="loading-state">Loading canvas...</div>;
  }

  if (error) {
    return <div className="error-state">Error: {error}</div>;
  }

  const handleAddNodeByType = async (nodeType: NodeType, executorType?: string, eventSourceType?: string, focusedNodeInfo?: { name: string; type: string } | null) => {
    try {
      const config = getNodeConfig(nodeType, executorType, eventSourceType, focusedNodeInfo);
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

  const getNodeConfig = (nodeType: NodeType, executorType?: string, eventSourceType?: string, focusedNodeInfo?: { name: string; type: string } | null) => {
    const baseConfig: { connections?: Array<SuperplaneConnection> } = {};

    if (focusedNodeInfo && (nodeType !== 'event_source')) {
      baseConfig.connections = [{
        name: focusedNodeInfo.name,
        type: focusedNodeInfo.type as SuperplaneConnectionType,
        filters: [],
        filterOperator: "FILTER_OPERATOR_AND"
      }];
    }

    switch (nodeType) {
      case 'stage':
        return executorType ? {
          name: '',
          executorType,
          ...baseConfig
        } : baseConfig;

      case 'event_source':
        return eventSourceType ? {
          name: '',
          eventSourceType
        } : undefined;

      case 'connection_group':
        return {
          name: '',
          ...baseConfig
        };

      default:
        return undefined;
    }
  };

  return (
    <StrictMode>
      <div className="h-[100vh] overflow-hidden">

        <CanvasNavigation
          canvasName={canvasName}
          activeView={activeView}
          onViewChange={handleViewChange}
          organizationId={organizationId!}
        />

        {activeView === 'editor' ? (
          <div className="relative" style={{ height: "calc(100vh - 2.6rem)", overflow: "hidden" }}>
            <ComponentSidebar
              isOpen={isComponentSidebarOpen}
              onClose={() => setIsComponentSidebarOpen(false)}
              onNodeAdd={(nodeType: NodeType, executorType?: string, eventSourceType?: string, focusedNodeInfo?: { name: string; type: string } | null) => {
                handleAddNodeByType(nodeType, executorType, eventSourceType, focusedNodeInfo);
              }}
            />

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