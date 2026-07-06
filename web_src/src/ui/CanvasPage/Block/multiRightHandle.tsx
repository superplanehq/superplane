import React from "react";
import { Handle, Position } from "@xyflow/react";
import { CANVAS_CONNECTOR_COLOR } from "@/lib/canvasEdgeColors";
import { nodeCanvasChannelLabelClassName } from "@/lib/nodeCanvasSections";
import { AppendHandlePreview, AppendSourceHandle, type AppendFromNodeHandler } from "./appendHandle";
import { isAlreadyConnectedToNode } from "./connectionState";
import { HANDLE_STYLE } from "./handleStyle";
import type { BlockConnectionState, BlockEdgeState } from "./types";

const MULTI_HANDLE_CHANNEL_SPACING = 34;
const MULTI_APPEND_SOURCE_HANDLE_LEFT_NUDGE = -4;
const CHANNEL_LABEL_FONT = "500 12px ui-sans-serif, system-ui, sans-serif";
const CHANNEL_LABEL_PADDING_X = 4;
const CHANNEL_LINE_TRAILING_GAP = 6;
const CHANNEL_LINE_MIN_LENGTH = 30;
const CHANNEL_HANDLE_GAP = 4;
const CHANNEL_TRUNK_LENGTH = 16;
const CHANNEL_BRANCH_END_X = CHANNEL_TRUNK_LENGTH + 24;

type MultiRightHandleLayout = {
  handleSize: number;
  labelStartX: number;
  trunkLength: number;
  branchEndX: number;
  svgHeight: number;
  svgCenterY: number;
  svgWidth: number;
};

type ChannelPosition = {
  channel: string;
  offsetY: number;
  isHighlighted: boolean | undefined;
  hasOutgoingForChannel: boolean;
  lineEndX: number;
  handleLeftX: number;
};

let cachedMeasureCanvas: HTMLCanvasElement | null = null;

function measureChannelLabel(text: string): number {
  if (typeof document === "undefined") {
    return text.length * 7;
  }
  if (!cachedMeasureCanvas) {
    cachedMeasureCanvas = document.createElement("canvas");
  }
  const ctx = cachedMeasureCanvas.getContext("2d");
  if (!ctx) {
    return text.length * 7;
  }
  ctx.font = CHANNEL_LABEL_FONT;
  return Math.ceil(ctx.measureText(text).width);
}

function getChannelLineEndX(channel: string): number {
  const labelWidth = measureChannelLabel(channel) + CHANNEL_LABEL_PADDING_X * 2;
  const lineLength = Math.max(CHANNEL_LINE_MIN_LENGTH, labelWidth + CHANNEL_LINE_TRAILING_GAP);
  return CHANNEL_BRANCH_END_X + lineLength;
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

function getMultiRightHandleLayout(channels: string[]): MultiRightHandleLayout {
  const totalHeight = (channels.length - 1) * MULTI_HANDLE_CHANNEL_SPACING;
  const svgHeight = totalHeight + 40;
  const maxLineEndX = channels.reduce((max, channel) => Math.max(max, getChannelLineEndX(channel)), 0);

  return {
    handleSize: 12,
    labelStartX: CHANNEL_BRANCH_END_X + CHANNEL_LABEL_PADDING_X,
    trunkLength: CHANNEL_TRUNK_LENGTH,
    branchEndX: CHANNEL_BRANCH_END_X,
    svgHeight,
    svgCenterY: svgHeight / 2,
    svgWidth: maxLineEndX,
  };
}

function getChannelPositions({
  channels,
  allEdges,
  nodeId,
  isConnectionInteractive,
  getChannelHighlight,
  sharedLineEndX,
}: {
  channels: string[];
  allEdges: BlockEdgeState[];
  nodeId?: string;
  isConnectionInteractive: boolean;
  getChannelHighlight: (channel: string) => boolean | undefined;
  sharedLineEndX: number;
}): ChannelPosition[] {
  return channels.map((channel, index) => ({
    channel,
    offsetY: (index - (channels.length - 1) / 2) * MULTI_HANDLE_CHANNEL_SPACING,
    isHighlighted: isConnectionInteractive && getChannelHighlight(channel),
    hasOutgoingForChannel: allEdges.some(
      (edge) => edge.source === nodeId && (edge.sourceHandle ?? "default") === channel,
    ),
    lineEndX: sharedLineEndX,
    handleLeftX: sharedLineEndX + CHANNEL_HANDLE_GAP,
  }));
}

function MultiRightHandleLines({ layout, channels }: { layout: MultiRightHandleLayout; channels: ChannelPosition[] }) {
  return (
    <svg
      width={layout.svgWidth}
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
        stroke={CANVAS_CONNECTOR_COLOR}
        strokeWidth={3}
      />

      {channels.map(({ channel, offsetY, lineEndX }) => {
        const y = layout.svgCenterY + offsetY;
        return (
          <g key={channel}>
            <line
              x1={layout.trunkLength}
              y1={layout.svgCenterY}
              x2={layout.branchEndX}
              y2={y}
              stroke={CANVAS_CONNECTOR_COLOR}
              strokeWidth={3}
            />
            <line x1={layout.branchEndX} y1={y} x2={lineEndX} y2={y} stroke={CANVAS_CONNECTOR_COLOR} strokeWidth={3} />
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
  handleLeftX,
}: {
  layout: MultiRightHandleLayout;
  channel: string;
  offsetY: number;
  isHighlighted: boolean | undefined;
  isConnectionInteractive: boolean;
  canAppend: boolean;
  onAppend: () => void | Promise<void>;
  handleLeftX: number;
}) {
  return (
    <React.Fragment>
      <span
        className={nodeCanvasChannelLabelClassName}
        style={{
          left: layout.labelStartX,
          top: `calc(50% + ${offsetY}px)`,
          transform: "translateY(-50%)",
          lineHeight: `${layout.handleSize}px`,
          paddingLeft: CHANNEL_LABEL_PADDING_X,
          paddingRight: CHANNEL_LABEL_PADDING_X,
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
            lineWidth={28}
            buttonLeft={40}
            buttonTop={-9}
            style={{
              left: handleLeftX + MULTI_APPEND_SOURCE_HANDLE_LEFT_NUDGE,
              top: `calc(50% + ${offsetY}px)`,
              transform: "translateY(-50%)",
            }}
          />
          <AppendHandlePreview
            style={{
              position: "absolute",
              left: handleLeftX + 52,
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
            left: handleLeftX,
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
  const layout = getMultiRightHandleLayout(channels);
  const channelPositions = getChannelPositions({
    channels,
    allEdges,
    nodeId,
    isConnectionInteractive,
    getChannelHighlight,
    sharedLineEndX: layout.svgWidth,
  });

  return (
    <div className="absolute" style={{ left: "100%", top: 0, bottom: 0, pointerEvents: "none" }}>
      <MultiRightHandleLines layout={layout} channels={channelPositions} />

      {channelPositions.map(({ channel, offsetY, isHighlighted, hasOutgoingForChannel, handleLeftX }) => {
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
            handleLeftX={handleLeftX}
          />
        );
      })}
    </div>
  );
}
