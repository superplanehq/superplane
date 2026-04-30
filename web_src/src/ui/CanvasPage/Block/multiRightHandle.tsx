import React from "react";
import { Handle, Position } from "@xyflow/react";
import { APPEND_CONNECTOR_COLOR, AppendHandleButton } from "./appendHandle";
import type { BlockConnectionState, BlockEdgeState } from "./types";

const HANDLE_STYLE = {
  width: 12,
  height: 12,
  borderRadius: 100,
  border: `3px solid ${APPEND_CONNECTOR_COLOR}`,
  background: "transparent",
};

type MultiRightHandleLayout = {
  handleSize: number;
  labelStartX: number;
  lineEndX: number;
  handleLeftX: number;
  trunkLength: number;
  branchEndX: number;
  svgHeight: number;
  svgCenterY: number;
};

type ChannelPosition = {
  channel: string;
  offsetY: number;
  isHighlighted: boolean | undefined;
  hasOutgoingForChannel: boolean;
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

function getMultiRightHandleLayout(channelCount: number): MultiRightHandleLayout {
  const channelSpacing = 34;
  const trunkLength = 16;
  const branchEndX = trunkLength + 24;
  const lineEndX = branchEndX + 62;
  const handleGap = 4;
  const totalHeight = (channelCount - 1) * channelSpacing;
  const svgHeight = totalHeight + 40;

  return {
    handleSize: 12,
    labelStartX: branchEndX + 4,
    lineEndX,
    handleLeftX: lineEndX + handleGap,
    trunkLength,
    branchEndX,
    svgHeight,
    svgCenterY: svgHeight / 2,
  };
}

function getChannelPositions({
  channels,
  allEdges,
  nodeId,
  isConnectionInteractive,
  getChannelHighlight,
}: {
  channels: string[];
  allEdges: BlockEdgeState[];
  nodeId?: string;
  isConnectionInteractive: boolean;
  getChannelHighlight: (channel: string) => boolean | undefined;
}): ChannelPosition[] {
  const channelSpacing = 34;

  return channels.map((channel, index) => ({
    channel,
    offsetY: (index - (channels.length - 1) / 2) * channelSpacing,
    isHighlighted: isConnectionInteractive && getChannelHighlight(channel),
    hasOutgoingForChannel: allEdges.some(
      (edge) => edge.source === nodeId && (edge.sourceHandle ?? "default") === channel,
    ),
  }));
}

function MultiRightHandleLines({ layout, channels }: { layout: MultiRightHandleLayout; channels: ChannelPosition[] }) {
  return (
    <svg
      width={layout.lineEndX}
      height={layout.svgHeight}
      style={{
        position: "absolute",
        left: 0,
        top: `calc(50% - ${layout.svgCenterY}px)`,
        overflow: "visible",
      }}
    >
      <line
        x1={0}
        y1={layout.svgCenterY}
        x2={layout.trunkLength}
        y2={layout.svgCenterY}
        stroke={APPEND_CONNECTOR_COLOR}
        strokeWidth={3}
      />

      {channels.map(({ channel, offsetY }) => {
        const y = layout.svgCenterY + offsetY;
        return (
          <g key={channel}>
            <line
              x1={layout.trunkLength}
              y1={layout.svgCenterY}
              x2={layout.branchEndX}
              y2={y}
              stroke={APPEND_CONNECTOR_COLOR}
              strokeWidth={3}
            />
            <line
              x1={layout.branchEndX}
              y1={y}
              x2={layout.lineEndX}
              y2={y}
              stroke={APPEND_CONNECTOR_COLOR}
              strokeWidth={3}
            />
          </g>
        );
      })}
    </svg>
  );
}

function MultiRightChannelControl({
  layout,
  channel,
  offsetY,
  isHighlighted,
  isConnectionInteractive,
  canAppend,
}: {
  layout: MultiRightHandleLayout;
  channel: string;
  offsetY: number;
  isHighlighted: boolean | undefined;
  isConnectionInteractive: boolean;
  canAppend: boolean;
}) {
  return (
    <React.Fragment>
      <span
        className="text-xs font-medium whitespace-nowrap absolute bg-slate-100"
        style={{
          left: layout.labelStartX,
          top: `calc(50% + ${offsetY}px)`,
          transform: "translateY(-50%)",
          color: "#8B9AAC",
          lineHeight: `${layout.handleSize}px`,
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
          left: layout.handleLeftX,
          top: `calc(50% + ${offsetY}px)`,
          transform: "translateY(-50%)",
          pointerEvents: isConnectionInteractive ? "auto" : "none",
        }}
        className={isHighlighted ? "highlighted" : undefined}
      />

      {canAppend ? (
        <>
          <div
            style={{
              position: "absolute",
              left: layout.handleLeftX + 12,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
              width: 24,
              height: 3,
              backgroundColor: APPEND_CONNECTOR_COLOR,
              pointerEvents: "none",
            }}
          />
          <AppendHandleButton
            label={`Add next component (${channel})`}
            style={{
              left: layout.handleLeftX + 32,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
              position: "absolute",
            }}
          />
        </>
      ) : null}
    </React.Fragment>
  );
}

export function MultiRightHandle({
  channels,
  hoveredEdge,
  connectingFrom,
  nodeId,
  allEdges,
  isConnectionInteractive,
}: {
  channels: string[];
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
  isConnectionInteractive: boolean;
}) {
  const getChannelHighlight = getChannelHighlightResolver({
    hoveredEdge,
    connectingFrom,
    nodeId,
    allEdges,
  });
  const layout = getMultiRightHandleLayout(channels.length);
  const channelPositions = getChannelPositions({
    channels,
    allEdges,
    nodeId,
    isConnectionInteractive,
    getChannelHighlight,
  });

  return (
    <div className="absolute" style={{ left: "100%", top: 0, bottom: 0, pointerEvents: "none" }}>
      <MultiRightHandleLines layout={layout} channels={channelPositions} />

      {channelPositions.map(({ channel, offsetY, isHighlighted, hasOutgoingForChannel }) => {
        const canAppend = isConnectionInteractive && !hasOutgoingForChannel;

        return (
          <MultiRightChannelControl
            key={channel}
            layout={layout}
            channel={channel}
            offsetY={offsetY}
            isHighlighted={isHighlighted}
            isConnectionInteractive={isConnectionInteractive}
            canAppend={canAppend}
          />
        );
      })}
    </div>
  );
}
