import type { CSSProperties } from "react";
import { Handle, Position } from "@xyflow/react";
import { AppendHandlePreview, AppendSourceHandle, type AppendFromNodeHandler } from "./appendHandle";
import { isAlreadyConnectedToNode } from "./connectionState";
import { getOutputChannels } from "./data";
import { HANDLE_STYLE } from "./handleStyle";
import { MultiRightHandle } from "./multiRightHandle";
import type { BlockConnectionState, BlockEdgeState, BlockOrientation, BlockProps, CanvasBlockData } from "./types";

function getBlockEdges(data: CanvasBlockData): BlockEdgeState[] {
  return data._allEdges || [];
}

function getOrientation(data: CanvasBlockData): BlockOrientation {
  return data._orientation === "vertical" ? "vertical" : "horizontal";
}

const VERTICAL_TARGET_POSITION_STYLE: CSSProperties = {
  top: -15,
  left: "50%",
  transform: "translateX(-50%)",
};

const VERTICAL_SOURCE_POSITION_STYLE: CSSProperties = {
  bottom: -15,
  left: "50%",
  transform: "translateX(-50%)",
};

const HORIZONTAL_TARGET_POSITION_STYLE: CSSProperties = {
  left: -15,
  top: 18,
};

const HORIZONTAL_SOURCE_POSITION_STYLE: CSSProperties = {
  right: -15,
  top: 18,
};

function targetHandlePosition(orientation: BlockOrientation): Position {
  return orientation === "vertical" ? Position.Top : Position.Left;
}

function sourceHandlePosition(orientation: BlockOrientation): Position {
  return orientation === "vertical" ? Position.Bottom : Position.Right;
}

function targetHandlePositionStyle(orientation: BlockOrientation): CSSProperties {
  return orientation === "vertical" ? VERTICAL_TARGET_POSITION_STYLE : HORIZONTAL_TARGET_POSITION_STYLE;
}

function sourceHandlePositionStyle(orientation: BlockOrientation): CSSProperties {
  return orientation === "vertical" ? VERTICAL_SOURCE_POSITION_STYLE : HORIZONTAL_SOURCE_POSITION_STYLE;
}

function getNodeConnectionStats(edges: BlockEdgeState[], nodeId?: string) {
  if (!nodeId) {
    return { hasIncoming: false, hasOutgoing: false };
  }

  return {
    hasIncoming: edges.some((edge) => edge.target === nodeId),
    hasOutgoing: edges.some((edge) => edge.source === nodeId),
  };
}

function isDisconnectedNode(stats: { hasIncoming: boolean; hasOutgoing: boolean }) {
  return !stats.hasIncoming && !stats.hasOutgoing;
}

function shouldHideInactiveLeftHandle(
  isConnectionInteractive: boolean,
  stats: { hasIncoming: boolean; hasOutgoing: boolean },
) {
  return !isConnectionInteractive && isDisconnectedNode(stats);
}

function getLeftHandleHighlight({
  isConnectionInteractive,
  hoveredEdge,
  connectingFrom,
  nodeId,
  isAlreadyConnected,
}: {
  isConnectionInteractive: boolean;
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  isAlreadyConnected: boolean;
}) {
  if (!isConnectionInteractive) {
    return false;
  }

  if (hoveredEdge && hoveredEdge.target === nodeId) {
    return true;
  }

  return connectingFrom?.nodeId !== nodeId && connectingFrom?.handleType === "source" && !isAlreadyConnected;
}

function shouldHideLeftHandle(data: CanvasBlockData) {
  return data.type === "trigger" || data.type === "annotation";
}

function shouldHideRightHandle(data: CanvasBlockData) {
  return data.isTemplate || data.isPendingConnection || data.type === "annotation";
}

function getSingleChannelHighlight(args: {
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  channel: string;
  allEdges: BlockEdgeState[];
}) {
  const { hoveredEdge, connectingFrom, nodeId, channel, allEdges } = args;
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
}

function SingleRightHandle({
  channel,
  hoveredEdge,
  connectingFrom,
  nodeId,
  allEdges,
  isConnectionInteractive,
  orientation,
}: {
  channel: string;
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
  isConnectionInteractive: boolean;
  orientation: BlockOrientation;
}) {
  const isHighlighted =
    isConnectionInteractive &&
    getSingleChannelHighlight({
      hoveredEdge,
      connectingFrom,
      nodeId,
      channel,
      allEdges,
    });

  return (
    <Handle
      type="source"
      position={sourceHandlePosition(orientation)}
      id={channel}
      style={{
        ...HANDLE_STYLE,
        ...sourceHandlePositionStyle(orientation),
        pointerEvents: isConnectionInteractive ? "auto" : "none",
      }}
      className={isHighlighted ? "highlighted" : undefined}
    />
  );
}

function EndNodeAppendConnector({
  nodeId,
  channel,
  onAppendFromNode,
  hoveredEdge,
  connectingFrom,
  allEdges,
  isConnectionInteractive,
}: {
  nodeId?: string;
  channel: string;
  onAppendFromNode?: AppendFromNodeHandler;
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  allEdges: BlockEdgeState[];
  isConnectionInteractive: boolean;
}) {
  if (!nodeId || !onAppendFromNode) {
    return null;
  }

  const isHighlighted =
    isConnectionInteractive &&
    getSingleChannelHighlight({
      hoveredEdge,
      connectingFrom,
      nodeId,
      channel,
      allEdges,
    });

  return (
    <>
      <AppendSourceHandle
        channel={channel}
        label="Add next component"
        onAppend={() => onAppendFromNode(nodeId, channel)}
        isHighlighted={isHighlighted}
        style={{
          right: -15,
          top: 18,
        }}
      />
      <AppendHandlePreview
        style={{
          position: "absolute",
          left: "calc(100% + 85px)",
          top: 0,
        }}
        connectorTop={18}
      />
    </>
  );
}

export function LeftHandle({
  data,
  nodeId,
  isConnectionInteractive = true,
}: Pick<BlockProps, "data" | "nodeId"> & { isConnectionInteractive?: boolean }) {
  const allEdges = getBlockEdges(data);
  const connectionStats = getNodeConnectionStats(allEdges, nodeId);

  if (shouldHideInactiveLeftHandle(isConnectionInteractive, connectionStats)) {
    return null;
  }

  if (shouldHideLeftHandle(data)) return null;

  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;
  const isAlreadyConnected = isAlreadyConnectedToNode(
    allEdges,
    connectingFrom,
    connectingFrom?.nodeId,
    nodeId,
    connectingFrom?.handleId,
  );
  const isHighlighted = getLeftHandleHighlight({
    isConnectionInteractive,
    hoveredEdge,
    connectingFrom,
    nodeId,
    isAlreadyConnected,
  });
  const orientation = getOrientation(data);

  return (
    <Handle
      type="target"
      position={targetHandlePosition(orientation)}
      style={{
        ...HANDLE_STYLE,
        ...targetHandlePositionStyle(orientation),
        pointerEvents: isConnectionInteractive ? "auto" : "none",
      }}
      className={isHighlighted ? "highlighted" : undefined}
    />
  );
}

export function RightHandle({
  data,
  nodeId,
  isConnectionInteractive = true,
  onAppendFromNode,
}: Pick<BlockProps, "data" | "nodeId" | "onAppendFromNode"> & { isConnectionInteractive?: boolean }) {
  const allEdges = getBlockEdges(data);
  const { hasIncoming, hasOutgoing } = getNodeConnectionStats(allEdges, nodeId);
  const isDisconnected = isDisconnectedNode({ hasIncoming, hasOutgoing });

  if (!isConnectionInteractive && (!hasOutgoing || isDisconnected)) {
    return null;
  }

  if (shouldHideRightHandle(data)) return null;

  const channels = getOutputChannels(data);
  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;

  if (getOrientation(data) === "vertical") {
    return (
      <VerticalSourceHandles
        channels={channels}
        hoveredEdge={hoveredEdge}
        connectingFrom={connectingFrom}
        nodeId={nodeId}
        allEdges={allEdges}
        isConnectionInteractive={isConnectionInteractive}
      />
    );
  }

  if (isConnectionInteractive && !hasOutgoing && channels.length === 1 && onAppendFromNode) {
    return (
      <EndNodeAppendConnector
        nodeId={nodeId}
        channel={channels[0]}
        onAppendFromNode={onAppendFromNode}
        hoveredEdge={hoveredEdge}
        connectingFrom={connectingFrom}
        allEdges={allEdges}
        isConnectionInteractive={isConnectionInteractive}
      />
    );
  }

  if (channels.length === 1) {
    return (
      <SingleRightHandle
        channel={channels[0]}
        hoveredEdge={hoveredEdge}
        connectingFrom={connectingFrom}
        nodeId={nodeId}
        allEdges={allEdges}
        isConnectionInteractive={isConnectionInteractive}
        orientation="horizontal"
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
      isConnectionInteractive={isConnectionInteractive}
      onAppendFromNode={onAppendFromNode}
    />
  );
}

const VERTICAL_CHANNEL_SPACING = 40;

/**
 * Renders output handles along the bottom edge of a node for the vertical
 * auto-layout view. Multiple channels are distributed horizontally so edges leave
 * the node top-to-bottom. Channel labels are rendered above each handle.
 */
function VerticalSourceHandles({
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
  return (
    <>
      {channels.map((channel, index) => {
        const offsetX = (index - (channels.length - 1) / 2) * VERTICAL_CHANNEL_SPACING;
        const isHighlighted =
          isConnectionInteractive &&
          getSingleChannelHighlight({
            hoveredEdge,
            connectingFrom,
            nodeId,
            channel,
            allEdges,
          });

        return (
          <Handle
            key={channel}
            type="source"
            position={Position.Bottom}
            id={channel}
            style={{
              ...HANDLE_STYLE,
              bottom: -15,
              left: `calc(50% + ${offsetX}px)`,
              transform: "translateX(-50%)",
              pointerEvents: isConnectionInteractive ? "auto" : "none",
            }}
            className={isHighlighted ? "highlighted" : undefined}
          />
        );
      })}
    </>
  );
}
