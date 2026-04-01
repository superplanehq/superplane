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
  targetNodeId: string | undefined,
  sourceHandle?: string | null,
) {
  if (!connection) {
    return false;
  }

  return edges.some(
    (edge) =>
      edge.source === connection.nodeId &&
      edge.sourceHandle === (sourceHandle ?? connection.handleId) &&
      edge.target === targetNodeId,
  );
}

function getBlockEdges(data: CanvasBlockData): BlockEdgeState[] {
  return data._allEdges || [];
}

export function LeftHandle({ data, nodeId }: Pick<BlockProps, "data" | "nodeId">) {
  if (data.type === "trigger" || data.type === "annotation" || data.type === "group") return null;

  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;
  const isAlreadyConnected = isAlreadyConnectedToNode(getBlockEdges(data), connectingFrom, nodeId);

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

// eslint-disable-next-line max-lines-per-function, max-statements, complexity
export function RightHandle({ data, nodeId }: Pick<BlockProps, "data" | "nodeId">) {
  const isTemplate = data.isTemplate;
  const isPendingConnection = data.isPendingConnection;
  if (isTemplate || isPendingConnection || data.type === "annotation" || data.type === "group") return null;

  const channels = getOutputChannels(data);
  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;
  const allEdges = getBlockEdges(data);

  if (channels.length === 1) {
    const isAlreadyConnected = isAlreadyConnectedToNode(allEdges, connectingFrom, connectingFrom?.nodeId, channels[0]);

    const isHighlighted =
      (hoveredEdge && hoveredEdge.source === nodeId && hoveredEdge.sourceHandle === channels[0]) ||
      (connectingFrom && connectingFrom.nodeId === nodeId && connectingFrom.handleId === channels[0]) ||
      (connectingFrom &&
        connectingFrom.nodeId !== nodeId &&
        connectingFrom.handleType === "target" &&
        !isAlreadyConnected);

    return (
      <Handle
        type="source"
        position={Position.Right}
        id={channels[0]}
        style={{
          ...HANDLE_STYLE,
          right: -15,
          top: 18,
        }}
        className={isHighlighted ? "highlighted" : undefined}
      />
    );
  }

  const getChannelHighlight = (channel: string) => {
    const isAlreadyConnected = isAlreadyConnectedToNode(allEdges, connectingFrom, connectingFrom?.nodeId, channel);

    return (
      (hoveredEdge && hoveredEdge.source === nodeId && hoveredEdge.sourceHandle === channel) ||
      (connectingFrom && connectingFrom.nodeId === nodeId && connectingFrom.handleId === channel) ||
      (connectingFrom &&
        connectingFrom.nodeId !== nodeId &&
        connectingFrom.handleType === "target" &&
        !isAlreadyConnected)
    );
  };

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
