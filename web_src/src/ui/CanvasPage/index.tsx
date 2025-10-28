import {
  Background,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge as ReactFlowEdge,
  type Node as ReactFlowNode,
} from "@xyflow/react";

import { useCallback, useMemo } from "react";

import { AiSidebar } from "../ai";
import { ComponentSidebar } from "../componentSidebar";
import { ViewToggle } from "../ViewToggle";
import { Block, BlockData } from "./Block";
import "./canvas-reset.css";
import { Header, type BreadcrumbItem } from "./Header";
import { genCommit } from "./storybooks/commits";
import { Simulation } from "./storybooks/useSimulation";
import { CanvasPageState, useCanvasState } from "./useCanvasState";

export interface CanvasNode extends ReactFlowNode {
  __simulation?: Simulation;
}

export interface CanvasEdge extends ReactFlowEdge {}

export type OnApproveFn = (
  nodeId: string,
  approveId: string,
  artifact?: Record<string, string>
) => void;

export type OnRejectFn = (
  nodeId: string,
  rejectId: string,
  comment?: string
) => void;

export interface CanvasPageProps {
  nodes: CanvasNode[];
  edges: CanvasEdge[];

  startCollapsed?: boolean;
  title?: string;
  breadcrumbs?: BreadcrumbItem[];

  onNodeExpand?: (nodeId: string, nodeData: unknown) => void;
  onApprove?: OnApproveFn;
  onReject?: OnRejectFn;
}

const EDGE_STYLE = {
  type: "default",
  style: { stroke: "#C9D5E1", strokeWidth: 3 },
} as const;

function CanvasPage(props: CanvasPageProps) {
  const state = useCanvasState(props);

  return (
    <div className="h-[100vh] w-[100vw] overflow-hidden sp-canvas relative">
      <ReactFlowProvider>
        <CanvasContent state={state} />
      </ReactFlowProvider>

      <AiSidebar />
      <Sidebar state={state} />
    </div>
  );
}

function Sidebar({ state }: { state: CanvasPageState }) {
  const latestEvents = useMemo(
    () => [
      {
        title: genCommit().message,
        subtitle: "4m",
        state: "processed" as const,
        isOpen: false,
        receivedAt: new Date(),
        childEventsInfo: {
          count: 1,
          state: "processed" as const,
          waitingInfos: [],
        },
      },
      {
        title: genCommit().message,
        subtitle: "3h",
        state: "discarded" as const,
        isOpen: false,
        receivedAt: new Date(Date.now() - 1000 * 60 * 30),
        values: {
          Author: "Pedro Forestileao",
          Commit: "feat: update component sidebar",
          Branch: "feature/ui-update",
          Type: "merge",
          "Event ID": "abc123-def456-ghi789",
        },
        childEventsInfo: {
          count: 3,
          state: "processed" as const,
          waitingInfos: [
            {
              icon: "check",
              info: "Tests passed",
            },
            {
              icon: "check",
              info: "Deploy completed",
            },
          ],
        },
      },
    ],
    []
  );

  const nextInQueueEvents = useMemo(
    () => [
      {
        title: genCommit().message,
        state: "waiting" as const,
        isOpen: false,
        receivedAt: new Date(Date.now() + 1000 * 60 * 5),
        childEventsInfo: {
          count: 2,
          state: "waiting" as const,
          waitingInfos: [
            {
              icon: "clock",
              info: "Waiting for approval",
              futureTimeDate: new Date(Date.now() + 1000 * 60 * 15),
            },
          ],
        },
      },
      {
        title: genCommit().message,
        state: "waiting" as const,
        isOpen: false,
        receivedAt: new Date(Date.now() + 1000 * 60 * 10),
        childEventsInfo: {
          count: 1,
          state: "waiting" as const,
          waitingInfos: [],
        },
      },
    ],
    []
  );

  return (
    <ComponentSidebar
      isOpen={state.componentSidebar.isOpen}
      onClose={state.componentSidebar.close}
      latestEvents={latestEvents}
      nextInQueueEvents={nextInQueueEvents}
      metadata={[
        {
          icon: "book",
          label: "monarch-app",
        },
        {
          icon: "filter",
          label: "branch=main",
        },
      ]}
      title={"Build/Test/Deploy Stage"}
      moreInQueueCount={0}
    />
  );
}

function CanvasContent({ state }: { state: CanvasPageState }) {
  const { fitView } = useReactFlow();

  const handleNodeExpand = useCallback(
    (nodeId: string) => {
      const node = state.nodes?.find((n) => n.id === nodeId);
      if (node && state.onNodeExpand) {
        state.onNodeExpand(nodeId, node.data);
        fitView();
      }
    },
    [state.nodes, state.onNodeExpand, fitView]
  );

  const nodeTypes = useMemo(
    () => ({
      default: (nodeProps: { data: unknown; id: string }) => (
        <Block
          data={nodeProps.data as BlockData}
          onExpand={handleNodeExpand}
          onApprove={state.onApprove}
          onReject={state.onReject}
          nodeId={nodeProps.id}
          onClick={state.componentSidebar.open}
        />
      ),
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }),
    []
  );

  const edgeTypes = useMemo(() => ({}), []);
  const styledEdges = useMemo(
    () => state.edges?.map((e) => ({ ...e, ...EDGE_STYLE })),
    [state.edges]
  );

  return (
    <>
      {/* Header */}
      <Header breadcrumbs={state.breadcrumbs} />

      {/* Toggle button */}
      <div className="absolute top-14 left-1/2 transform -translate-x-1/2 z-10">
        <ViewToggle
          isCollapsed={state.isCollapsed}
          onToggle={state.toggleCollapse}
        />
      </div>

      <div className="pt-12 h-full">
        <div className="h-full w-full">
          <ReactFlow
            nodes={state.nodes}
            edges={styledEdges}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            fitView={true}
            minZoom={0.4}
            maxZoom={1.5}
            zoomOnScroll={true}
            zoomOnPinch={true}
            zoomOnDoubleClick={false}
            panOnScroll={true}
            panOnDrag={true}
            selectionOnDrag={false}
            panOnScrollSpeed={0.8}
            nodesDraggable={true}
            nodesConnectable={false}
            elementsSelectable={true}
            onNodesChange={state.onNodesChange}
            onEdgesChange={state.onEdgesChange}
            onNodeDoubleClick={(_, node) => state.toggleNodeCollapse(node.id)}
          >
            <Background bgColor="#F1F5F9" color="#F1F5F9" />
          </ReactFlow>
        </div>
      </div>
    </>
  );
}

export { CanvasPage };
