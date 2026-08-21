import React from "react";
import { Handle, Position } from "@xyflow/react";
import { CANVAS_CONNECTOR_COLOR } from "@/lib/canvasEdgeColors";
import { nodeCanvasChannelLabelClassName } from "@/lib/nodeCanvasSections";
import { AppendHandlePreview, AppendSourceHandle, type AppendFromNodeHandler } from "./appendHandle";
import { isAlreadyConnectedToNode } from "./connectionState";
import { resolveHandleStyle } from "./handleStyle";
import type { BlockConnectionState, BlockEdgeState } from "./types";

const MULTI_HANDLE_CHANNEL_SPACING = 96;
const CHANNEL_LABEL_PADDING_Y = 4;
const CHANNEL_STEM_LENGTH = 28;
const CHANNEL_HANDLE_GAP = 4;

type ChannelPosition = {
  channel: string;
  offsetX: number;
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
    offsetX: (index - (channels.length - 1) / 2) * MULTI_HANDLE_CHANNEL_SPACING,
    isHighlighted: isConnectionInteractive && getChannelHighlight(channel),
    hasOutgoingForChannel: allEdges.some(
      (edge) => edge.source === nodeId && (edge.sourceHandle ?? "default") === channel,
    ),
  }));
}

function MultiBottomChannelControl({
  channel,
  offsetX,
  isHighlighted,
  isConnectionInteractive,
  canAppend,
  onAppend,
}: {
  channel: string;
  offsetX: number;
  isHighlighted: boolean | undefined;
  isConnectionInteractive: boolean;
  canAppend: boolean;
  onAppend: () => void | Promise<void>;
}) {
  const handleTop = CHANNEL_STEM_LENGTH + CHANNEL_HANDLE_GAP;

  return (
    <React.Fragment>
      <span
        className={nodeCanvasChannelLabelClassName}
        style={{
          left: `calc(50% + ${offsetX}px)`,
          top: CHANNEL_LABEL_PADDING_Y,
          transform: "translateX(-50%)",
          lineHeight: "14px",
        }}
      >
        {channel}
      </span>

      <div
        aria-hidden
        style={{
          position: "absolute",
          left: `calc(50% + ${offsetX}px)`,
          top: 18,
          width: 3,
          height: CHANNEL_STEM_LENGTH,
          transform: "translateX(-50%)",
          backgroundColor: CANVAS_CONNECTOR_COLOR,
          pointerEvents: "none",
        }}
      />

      {canAppend ? (
        <>
          <AppendSourceHandle
            channel={channel}
            label={`Add next component (${channel})`}
            onAppend={onAppend}
            isHighlighted={isHighlighted}
            orientation="vertical"
            lineWidth={28}
            buttonLeft={-9}
            buttonTop={40}
            style={{
              left: `calc(50% + ${offsetX}px)`,
              top: handleTop,
              transform: "translateX(-50%)",
            }}
          />
          <AppendHandlePreview
            orientation="vertical"
            style={{
              position: "absolute",
              left: `calc(50% + ${offsetX}px)`,
              top: handleTop + 52,
              transform: "translateX(-50%)",
            }}
            containerOffsetY={0}
          />
        </>
      ) : (
        <Handle
          type="source"
          position={Position.Bottom}
          id={channel}
          style={{
            ...resolveHandleStyle(true),
            left: `calc(50% + ${offsetX}px)`,
            top: handleTop,
            transform: "translateX(-50%)",
            pointerEvents: isConnectionInteractive ? "auto" : "none",
          }}
          className={isHighlighted ? "highlighted" : undefined}
        />
      )}
    </React.Fragment>
  );
}

export function MultiBottomHandle({
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
  const channelPositions = getChannelPositions({
    channels,
    allEdges,
    nodeId,
    isConnectionInteractive,
    getChannelHighlight,
  });

  return (
    <div className="absolute left-0 right-0" style={{ top: "100%", height: 120, pointerEvents: "none" }}>
      {channelPositions.map(({ channel, offsetX, isHighlighted, hasOutgoingForChannel }) => {
        const canAppend = isConnectionInteractive && !!nodeId && !!onAppendFromNode && !hasOutgoingForChannel;

        return (
          <MultiBottomChannelControl
            key={channel}
            channel={channel}
            offsetX={offsetX}
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
