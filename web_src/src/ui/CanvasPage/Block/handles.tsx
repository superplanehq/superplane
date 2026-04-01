import React from "react";
import { Handle, Position } from "@xyflow/react";
import { getOutputChannels } from "./data";
import type { BlockConnectionState, BlockEdgeState, BlockProps, CanvasBlockData } from "./types";

const HANDLE_STYLE = {
  width: 12,
  height: 12,
  borderRadius: 100,
  border: "3px solid #C9D5E1",
  background: "transparent",
};

function isAlreadyConnectedToNode(
  edges: BlockEdgeState[],
  connection: BlockConnectionState | undefined,
  sourceNodeId: string | undefined,
  targetNodeId: string | undefined,
  sourceHandle?: string | null,
) {
  if (!connection || !sourceNodeId || !targetNodeId) {
    return false;
  }

  return edges.some(
    (edge) =>
      edge.source === sourceNodeId &&
      edge.sourceHandle === (sourceHandle ?? connection.handleId) &&
      edge.target === targetNodeId,
  );
}

function getBlockEdges(data: CanvasBlockData): BlockEdgeState[] {
  return data._allEdges || [];
}

function shouldHideLeftHandle(data: CanvasBlockData) {
  return data.type === "trigger" || data.type === "annotation" || data.type === "group";
}

function shouldHideRightHandle(data: CanvasBlockData) {
  return data.isTemplate || data.isPendingConnection || data.type === "annotation" || data.type === "group";
}

function getSingleChannelHighlight(args: {
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  channel: string;
  allEdges: BlockEdgeState[];
}) {
  const { hoveredEdge, connectingFrom, nodeId, channel, allEdges } = args;
  const isAlreadyConnected = isAlreadyConnectedToNode(allEdges, connectingFrom, nodeId, connectingFrom?.nodeId, channel);

  return (
    (hoveredEdge && hoveredEdge.source === nodeId && hoveredEdge.sourceHandle === channel) ||
    (connectingFrom && connectingFrom.nodeId === nodeId && connectingFrom.handleId === channel) ||
    (connectingFrom &&
      connectingFrom.nodeId !== nodeId &&
      connectingFrom.handleType === "target" &&
      !isAlreadyConnected)
  );
}

function SingleRightHandle({
  channel,
  hoveredEdge,
  connectingFrom,
  nodeId,
  allEdges,
}: {
  channel: string;
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
}) {
  const isHighlighted = getSingleChannelHighlight({
    hoveredEdge,
    connectingFrom,
    nodeId,
    channel,
    allEdges,
  });

  return (
    <Handle
      type="source"
      position={Position.Right}
      id={channel}
      style={{
        ...HANDLE_STYLE,
        right: -15,
        top: 18,
      }}
      className={isHighlighted ? "highlighted" : undefined}
    />
  );
}

function getChannelHighlightResolver(args: {
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
}) {
  const { hoveredEdge, connectingFrom, nodeId, allEdges } = args;

  return (channel: string) => {
    const isAlreadyConnected = isAlreadyConnectedToNode(
      allEdges,
      connectingFrom,
      nodeId,
      connectingFrom?.nodeId,
      channel,
    );

    return (
      (hoveredEdge && hoveredEdge.source === nodeId && hoveredEdge.sourceHandle === channel) ||
      (connectingFrom && connectingFrom.nodeId === nodeId && connectingFrom.handleId === channel) ||
      (connectingFrom &&
        connectingFrom.nodeId !== nodeId &&
        connectingFrom.handleType === "target" &&
        !isAlreadyConnected)
    );
  };
}

function MultiRightHandle({
  channels,
  hoveredEdge,
  connectingFrom,
  nodeId,
  allEdges,
}: {
  channels: string[];
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
}) {
  const getChannelHighlight = getChannelHighlightResolver({
    hoveredEdge,
    connectingFrom,
    nodeId,
    allEdges,
  });

  const channelSpacing = 24;
  const handleSize = 12;
  const edgeColor = "#C9D5E1";
  const trunkLength = 16;
  const branchEndX = trunkLength + 24;
  const labelStartX = branchEndX + 4;
  const lineEndX = branchEndX + 62;
  const handleGap = 4;
  const handleLeftX = lineEndX + handleGap;
  const totalHeight = (channels.length - 1) * channelSpacing;
  const svgHeight = totalHeight + 40;
  const svgCenterY = svgHeight / 2;
  const channelPositions = channels.map((channel, index) => ({
    channel,
    offsetY: (index - (channels.length - 1) / 2) * channelSpacing,
    isHighlighted: getChannelHighlight(channel),
  }));

  return (
    <div
      className="absolute"
      style={{
        left: "100%",
        top: 0,
        bottom: 0,
        pointerEvents: "none",
      }}
    >
      <svg
        width={lineEndX}
        height={svgHeight}
        style={{
          position: "absolute",
          left: 0,
          top: `calc(50% - ${svgCenterY}px)`,
          overflow: "visible",
        }}
      >
        <line x1={0} y1={svgCenterY} x2={trunkLength} y2={svgCenterY} stroke={edgeColor} strokeWidth={3} />

        {channelPositions.map(({ channel, offsetY }) => {
          const y = svgCenterY + offsetY;
          return (
            <g key={channel}>
              <line x1={trunkLength} y1={svgCenterY} x2={branchEndX} y2={y} stroke={edgeColor} strokeWidth={3} />
              <line x1={branchEndX} y1={y} x2={lineEndX} y2={y} stroke={edgeColor} strokeWidth={3} />
            </g>
          );
        })}
      </svg>

      {channelPositions.map(({ channel, offsetY, isHighlighted }) => (
        <React.Fragment key={channel}>
          <span
            className="text-xs font-medium whitespace-nowrap absolute bg-slate-100"
            style={{
              left: labelStartX,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
              color: "#8B9AAC",
              lineHeight: `${handleSize}px`,
              paddingLeft: 4,
              paddingRight: 4,
            }}
          >
            {channel}
          </span>

          <Handle
            type="source"
            position={Position.Right}
            id={channel}
            style={{
              ...HANDLE_STYLE,
              left: handleLeftX,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
              pointerEvents: "auto",
            }}
            className={isHighlighted ? "highlighted" : undefined}
          />
        </React.Fragment>
      ))}
    </div>
  );
}

export function LeftHandle({ data, nodeId }: Pick<BlockProps, "data" | "nodeId">) {
  if (shouldHideLeftHandle(data)) return null;

  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;
  const isAlreadyConnected = isAlreadyConnectedToNode(
    getBlockEdges(data),
    connectingFrom,
    connectingFrom?.nodeId,
    nodeId,
  );
  const isHighlighted =
    (hoveredEdge && hoveredEdge.target === nodeId) ||
    (connectingFrom &&
      connectingFrom.nodeId !== nodeId &&
      connectingFrom.handleType === "source" &&
      !isAlreadyConnected);

  return (
    <Handle
      type="target"
      position={Position.Left}
      style={{
        ...HANDLE_STYLE,
        left: -15,
        top: 18,
      }}
      className={isHighlighted ? "highlighted" : undefined}
    />
  );
}

export function RightHandle({ data, nodeId }: Pick<BlockProps, "data" | "nodeId">) {
  if (shouldHideRightHandle(data)) return null;

  const channels = getOutputChannels(data);
  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;
  const allEdges = getBlockEdges(data);

  if (channels.length === 1) {
    return (
      <SingleRightHandle
        channel={channels[0]}
        hoveredEdge={hoveredEdge}
        connectingFrom={connectingFrom}
        nodeId={nodeId}
        allEdges={allEdges}
      />
    );
  }

  return (
    <MultiRightHandle
      channels={channels}
      hoveredEdge={hoveredEdge}
      connectingFrom={connectingFrom}
      nodeId={nodeId}
      allEdges={allEdges}
    />
  );
}
