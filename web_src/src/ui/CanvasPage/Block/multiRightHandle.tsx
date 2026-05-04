import React from "react";
import { Handle, Position } from "@xyflow/react";
import {
  APPEND_CONNECTOR_COLOR,
  AppendHandlePreview,
  AppendSourceHandle,
  type AppendFromNodeHandler,
} from "./appendHandle";
import { isAlreadyConnectedToNode } from "./connectionState";
import { HANDLE_STYLE } from "./handleStyle";
import type { BlockConnectionState, BlockEdgeState } from "./types";

const MULTI_HANDLE_CHANNEL_SPACING = 34;

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
  const trunkLength = 16;
  const branchEndX = trunkLength + 24;
  const lineEndX = branchEndX + 62;
  const handleGap = 4;
  const totalHeight = (channelCount - 1) * MULTI_HANDLE_CHANNEL_SPACING;
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
  return channels.map((channel, index) => ({
    channel,
    offsetY: (index - (channels.length - 1) / 2) * MULTI_HANDLE_CHANNEL_SPACING,
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
  onAppend,
}: {
  layout: MultiRightHandleLayout;
  channel: string;
  offsetY: number;
  isHighlighted: boolean | undefined;
  isConnectionInteractive: boolean;
  canAppend: boolean;
  onAppend: () => void | Promise<void>;
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

      {canAppend ? (
        <>
          <AppendSourceHandle
            channel={channel}
            label={`Add next component (${channel})`}
            onAppend={onAppend}
            isHighlighted={isHighlighted}
            lineWidth={24}
            buttonLeft={32}
            buttonTop={-6}
            style={{
              left: layout.handleLeftX,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
            }}
          />
          <AppendHandlePreview
            style={{
              position: "absolute",
              left: layout.handleLeftX + 52,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
            }}
            containerOffsetY={54}
          />
        </>
      ) : (
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
      )}
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
  onAppendFromNode,
}: {
  channels: string[];
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
  isConnectionInteractive: boolean;
  onAppendFromNode?: AppendFromNodeHandler;
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
        const canAppend = isConnectionInteractive && !!nodeId && !!onAppendFromNode && !hasOutgoingForChannel;

        return (
          <MultiRightChannelControl
            key={channel}
            layout={layout}
            channel={channel}
            offsetY={offsetY}
            isHighlighted={isHighlighted}
            isConnectionInteractive={isConnectionInteractive}
            canAppend={canAppend}
            onAppend={() => (nodeId ? onAppendFromNode?.(nodeId, channel) : undefined)}
          />
        );
      })}
    </div>
  );
}
