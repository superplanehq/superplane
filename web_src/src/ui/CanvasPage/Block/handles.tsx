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

const APPEND_HANDLE_CLASS =
  "after:relative after:-top-[1px] after:content-['+'] after:text-lg after:font-semibold after:text-slate-300 hover:after:text-slate-400";
const APPEND_HANDLE_HOVER_CLASS = "sp-append-source-handle";
const APPEND_PREVIEW_CLASS = "sp-append-source-preview";

const APPEND_HANDLE_STYLE = {
  ...HANDLE_STYLE,
  width: 24,
  height: 24,
  border: "3px solid #C9D5E1",
  background: "white",
  pointerEvents: "auto" as const,
  cursor: "pointer",
  display: "flex",
  alignItems: "center" as const,
  justifyContent: "center" as const,
};

function AppendHandlePreview({
  style,
  connectorTop = "50%",
  containerOffsetY = 0,
}: {
  style: React.CSSProperties;
  connectorTop?: number | string;
  containerOffsetY?: number;
}) {
  const previewStemLength = 44;
  const previewConnectorSize = 12;
  const previewContainerGap = 12;
  const previewContainerLeft = previewStemLength + previewConnectorSize / 2 + previewContainerGap;

  return (
    <div className={APPEND_PREVIEW_CLASS} style={style}>
      <div
        style={{
          position: "absolute",
          left: 0,
          top: connectorTop,
          width: previewStemLength,
          height: 3,
          transform: "translateY(-50%)",
          backgroundColor: "#C9D5E1",
          borderRadius: 999,
        }}
      />
      <div
        style={{
          ...HANDLE_STYLE,
          position: "absolute",
          left: previewStemLength + 6,
          top: connectorTop,
          transform: "translate(-50%, -50%)",
          pointerEvents: "none",
        }}
      />
      <div
        style={{
          marginLeft: previewContainerLeft,
          marginTop: containerOffsetY,
          width: "23rem",
          height: 96,
          borderRadius: 8,
          background: "white",
          outline: "1px solid rgb(15 23 42 / 0.15)",
          opacity: 0.6,
        }}
      />
    </div>
  );
}

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

function getNodeConnectionStats(edges: BlockEdgeState[], nodeId?: string) {
  if (!nodeId) {
    return { hasIncoming: false, hasOutgoing: false };
  }
  const hasIncoming = edges.some((edge) => edge.target === nodeId);
  const hasOutgoing = edges.some((edge) => edge.source === nodeId);
  return { hasIncoming, hasOutgoing };
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
}: {
  channel: string;
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
  isConnectionInteractive: boolean;
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
      position={Position.Right}
      id={channel}
      style={{
        ...HANDLE_STYLE,
        right: -15,
        top: 18,
        pointerEvents: isConnectionInteractive ? "auto" : "none",
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
  isConnectionInteractive,
  onAppendFromNode,
}: {
  channels: string[];
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
  isConnectionInteractive: boolean;
  onAppendFromNode?: (nodeId: string, sourceHandleId?: string | null) => void | Promise<void>;
}) {
  const getChannelHighlight = getChannelHighlightResolver({
    hoveredEdge,
    connectingFrom,
    nodeId,
    allEdges,
  });

  const channelSpacing = 34;
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
    isHighlighted: isConnectionInteractive && getChannelHighlight(channel),
    hasOutgoingForChannel: allEdges.some(
      (edge) => edge.source === nodeId && (edge.sourceHandle ?? "default") === channel,
    ),
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

      {channelPositions.map(({ channel, offsetY, isHighlighted, hasOutgoingForChannel }) => {
        const isAppendHandle = isConnectionInteractive && !!nodeId && !!onAppendFromNode && !hasOutgoingForChannel;

        return (
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
              className={isHighlighted ? "highlighted" : undefined}
              style={{
                ...HANDLE_STYLE,
                left: handleLeftX,
                top: `calc(50% + ${offsetY}px)`,
                transform: "translateY(-50%)",
                pointerEvents: isConnectionInteractive ? "auto" : "none",
              }}
            />
            {isAppendHandle ? (
              <div
                style={{
                  position: "absolute",
                  left: handleLeftX + 12,
                  top: `calc(50% + ${offsetY}px)`,
                  transform: "translateY(-50%)",
                  width: 24,
                  height: 3,
                  backgroundColor: "#C9D5E1",
                  pointerEvents: "none",
                }}
              />
            ) : null}
            {isAppendHandle ? (
              <button
                type="button"
                aria-label={`Add next component (${channel})`}
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  void onAppendFromNode(nodeId, channel);
                }}
                className={`${APPEND_HANDLE_CLASS} ${APPEND_HANDLE_HOVER_CLASS}`}
                style={{
                  ...APPEND_HANDLE_STYLE,
                  left: handleLeftX + 32,
                  top: `calc(50% + ${offsetY}px)`,
                  transform: "translateY(-50%)",
                  position: "absolute",
                }}
              />
            ) : null}
            {isAppendHandle ? (
              <AppendHandlePreview
                style={{
                  position: "absolute",
                  left: handleLeftX + 52,
                  top: `calc(50% + ${offsetY}px)`,
                  transform: "translateY(-50%)",
                }}
                containerOffsetY={54}
              />
            ) : null}
          </React.Fragment>
        );
      })}
    </div>
  );
}

function EndNodeAppendConnector({
  nodeId,
  channel,
  onAppendFromNode,
}: {
  nodeId?: string;
  channel: string;
  onAppendFromNode?: (nodeId: string, sourceHandleId?: string | null) => void | Promise<void>;
}) {
  if (!nodeId || !onAppendFromNode) {
    return null;
  }

  return (
    <>
      <Handle
        type="source"
        position={Position.Right}
        id={channel}
        style={{
          ...HANDLE_STYLE,
          right: -21,
          top: 13,
          pointerEvents: "auto",
          transform: "none",
        }}
      />
      <div
        style={{
          position: "absolute",
          right: -63,
          top: 17,
          width: 42,
          height: 3,
          backgroundColor: "#C9D5E1",
          pointerEvents: "none",
        }}
      />
      <button
        type="button"
        aria-label="Add next component"
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          void onAppendFromNode(nodeId, channel);
        }}
        className={`${APPEND_HANDLE_CLASS} ${APPEND_HANDLE_HOVER_CLASS}`}
        style={{
          ...APPEND_HANDLE_STYLE,
          right: -87,
          top: 6,
          position: "absolute",
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
  const { hasIncoming, hasOutgoing } = getNodeConnectionStats(allEdges, nodeId);
  const isDisconnected = !hasIncoming && !hasOutgoing;

  // In live mode, hide all handles for disconnected components.
  if (!isConnectionInteractive && isDisconnected) {
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
  const isHighlighted =
    isConnectionInteractive &&
    ((hoveredEdge && hoveredEdge.target === nodeId) ||
      (connectingFrom &&
        connectingFrom.nodeId !== nodeId &&
        connectingFrom.handleType === "source" &&
        !isAlreadyConnected));

  return (
    <Handle
      type="target"
      position={Position.Left}
      style={{
        ...HANDLE_STYLE,
        left: -15,
        top: 18,
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
}: Pick<BlockProps, "data" | "nodeId"> & { isConnectionInteractive?: boolean }) {
  const allEdges = getBlockEdges(data);
  const { hasIncoming, hasOutgoing } = getNodeConnectionStats(allEdges, nodeId);
  const isDisconnected = !hasIncoming && !hasOutgoing;

  // In live mode, hide end connectors and all connectors for disconnected nodes.
  if (!isConnectionInteractive && (!hasOutgoing || isDisconnected)) {
    return null;
  }

  if (shouldHideRightHandle(data)) return null;

  const channels = getOutputChannels(data);
  const onAppendFromNode = data._callbacksRef?.current?.onAppendFromNode;
  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;

  if (isConnectionInteractive && !hasOutgoing && channels.length === 1 && nodeId && onAppendFromNode) {
    return <EndNodeAppendConnector nodeId={nodeId} channel={channels[0]} onAppendFromNode={onAppendFromNode} />;
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
