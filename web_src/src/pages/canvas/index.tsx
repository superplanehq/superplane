import { StrictMode, useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useLocation, useNavigate } from "react-router-dom";
import { FlowRenderer } from "./components/FlowRenderer";
import { useCanvasStore } from "./store/canvasStore";
import { useWebsocketEvents } from "./hooks/useWebsocketEvents";
import { superplaneDescribeCanvas, superplaneListStages, superplaneListEventSources, superplaneListConnectionGroups, SuperplaneStage, SuperplaneEventSource, SuperplaneCanvas, SuperplaneConnectionGroup, SuperplaneConnection, SuperplaneConnectionType } from "@/api-client";
import { ConnectionGroupWithEvents, EventSourceWithEvents, Stage } from "./store/types";
import { Sidebar } from "./components/SideBar";
import { EventSourceSidebar } from "./components/EventSourceSidebar";
import { ComponentSidebar, ConnectionInfo } from "./components/ComponentSidebar";
import { CanvasNavigation, CanvasNavigationContent, type CanvasView } from "../../components/CanvasNavigation";
import { useNodeHandlers } from "./utils/nodeHandlers";
import { NodeType } from "./utils/nodeFactories";
import { withOrganizationHeader } from "../../utils/withOrganizationHeader";
import { useAutoLayout } from "./hooks/useAutoLayout";
import { DEFAULT_SIDEBAR_WIDTH } from "./utils/constants";


export function Canvas() {
  const { organizationId, canvasId } = useParams<{ organizationId: string, canvasId: string }>();
  const location = useLocation();
  const navigate = useNavigate();
  const { initialize, selectedStageId, cleanSelectedStageId, selectedEventSourceId, cleanSelectedEventSourceId, editingStageId, stages, eventSources, connectionGroups, approveStageEvent, discardStageEvent, cancelStageExecution, lockedNodes, setFocusedNodeId, setNodes } = useCanvasStore();
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isComponentSidebarOpen, setIsComponentSidebarOpen] = useState(false);
  const [canvasName, setCanvasName] = useState<string>('');

  useEffect(() => {
    if (canvasName) {
      document.title = `${canvasName} - Superplane`;
    } else {
      document.title = 'Superplane';
    }
  }, [canvasName]);

  // Set initial component sidebar state based on canvas contents
  useEffect(() => {
    if (!isLoading) {
      const hasCanvasComponents = stages.length > 0 || eventSources.length > 0 || connectionGroups.length > 0;
      setIsComponentSidebarOpen(!hasCanvasComponents);
    }
  }, [isLoading, stages.length, eventSources.length, connectionGroups.length]);

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


  const processAndInitializeStore = useCallback((
    canvas: SuperplaneCanvas,
    rawStages: SuperplaneStage[],
    rawEventSources: SuperplaneEventSource[],
    rawConnectionGroups: SuperplaneConnectionGroup[]
  ) => {
    const stages: Stage[] = rawStages.map(stage => {
      const queue = stage.status?.queue?.items || [];

      const executions = [];
      if (stage.status?.lastExecution) {
        executions.push(stage.status.lastExecution);
      }

      return {
        ...stage,
        queue,
        executions
      };
    });

    const eventSourcesWithEvents: EventSourceWithEvents[] = rawEventSources.map(eventSource => ({
      ...eventSource,
      events: eventSource.status?.history?.recentItems || [],
      eventFilters: []
    }));

    const connectionGroupsWithEvents: ConnectionGroupWithEvents[] = rawConnectionGroups.map(connectionGroup => ({
      ...connectionGroup,
      events: []
    }));

    const initialData = {
      canvas: canvas || {},
      stages: stages,
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

        processAndInitializeStore(
          basicData.canvas,
          basicData.stages,
          basicData.eventSources,
          basicData.connectionGroups
        );

        setIsLoading(false);

      } catch (err) {
        console.error('Error fetching canvas data:', err);
        setError('Failed to load canvas data');
        setIsLoading(false);
      }
    };

    fetchCanvasData();
  }, [canvasId, organizationId, fetchCanvasBasicData, processAndInitializeStore]);

  if (isLoading) {
    return <div className="loading-state">Loading canvas...</div>;
  }

  if (error) {
    return <div className="error-state">Error: {error}</div>;
  }

  const handleAddNodeByType = async (nodeType: NodeType, executorType?: string, eventSourceType?: string, focusedNodeInfo?: ConnectionInfo | null) => {
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
        }, 50);
      }
    } catch (error) {
      console.error(`Failed to add node of type ${nodeType}:`, error);
    }
  };

  const getNodeConfig = (nodeType: NodeType, executorType?: string, eventSourceType?: string, focusedNodeInfo?: ConnectionInfo | null) => {
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
              onNodeAdd={(nodeType: NodeType, executorType?: string, eventSourceType?: string, focusedNodeInfo?: ConnectionInfo | null) => {
                handleAddNodeByType(nodeType, executorType, eventSourceType, focusedNodeInfo);
              }}
            />

            {!isComponentSidebarOpen && (
              <button
                onClick={() => setIsComponentSidebarOpen(true)}
                className="fixed top-16 left-4 z-30 px-4 py-2 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded-md shadow-md hover:bg-gray-50 dark:hover:bg-zinc-800 transition-all duration-300 flex items-center gap-2"
                title="Open Components"
              >
                <span className="text-medium font-semibold text-gray-700 dark:text-zinc-100">Components</span>
                <span style={{ fontSize: '1.2rem' }} className="material-symbols-outlined text-gray-600 dark:text-zinc-300 -scale-x-100">menu_open</span>
              </button>
            )}

            <FlowRenderer />
            {selectedStage && !editingStageId && <Sidebar approveStageEvent={approveStageEvent} discardStageEvent={discardStageEvent} cancelStageExecution={cancelStageExecution} selectedStage={selectedStage} onClose={() => cleanSelectedStageId()} initialWidth={DEFAULT_SIDEBAR_WIDTH} />}
            {selectedEventSource && <EventSourceSidebar selectedEventSource={selectedEventSource} onClose={() => cleanSelectedEventSourceId()} initialWidth={DEFAULT_SIDEBAR_WIDTH} />}
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