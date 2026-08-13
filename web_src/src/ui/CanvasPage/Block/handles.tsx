import type { CSSProperties } from "react";
import { Handle, Position } from "@xyflow/react";
import { isVerticalCanvasFlow } from "@/lib/canvasFlowDirection";
import { AppendHandlePreview, AppendSourceHandle, type AppendFromNodeHandler } from "./appendHandle";
import { isAlreadyConnectedToNode } from "./connectionState";
import { getOutputChannels } from "./data";
import { FACTORY_SIDE_HANDLE_ID, FACTORY_SPINE_HANDLE_ID } from "@/lib/layout/factoryRunLeafLayout";
import { FACTORY_HANDLE_OUTSET_PX, resolveHandleStyle } from "./handleStyle";
import { MultiBottomHandle } from "./multiBottomHandle";
import { MultiRightHandle } from "./multiRightHandle";
import type { BlockConnectionState, BlockEdgeState, BlockProps, CanvasBlockData } from "./types";

function connectionPointerEvents(isConnectionInteractive: boolean): CSSProperties["pointerEvents"] {
  return isConnectionInteractive ? "auto" : "none";
}

function getBlockEdges(data: CanvasBlockData): BlockEdgeState[] {
  return data._allEdges || [];
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

function shouldHideInactiveTargetHandle(
  isConnectionInteractive: boolean,
  stats: { hasIncoming: boolean; hasOutgoing: boolean },
) {
  return !isConnectionInteractive && isDisconnectedNode(stats);
}

function getTargetHandleHighlight({
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

function shouldHideTargetHandle(data: CanvasBlockData) {
  return data.type === "trigger" || data.type === "annotation";
}

function shouldHideSourceHandle(data: CanvasBlockData) {
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

function SingleSourceHandle({
  channel,
  hoveredEdge,
  connectingFrom,
  nodeId,
  allEdges,
  isConnectionInteractive,
  isVertical,
}: {
  channel: string;
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  nodeId?: string;
  allEdges: BlockEdgeState[];
  isConnectionInteractive: boolean;
  isVertical: boolean;
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
      position={isVertical ? Position.Bottom : Position.Right}
      id={channel}
      style={{
        ...resolveHandleStyle(isVertical),
        ...(isVertical
          ? {
              bottom: -FACTORY_HANDLE_OUTSET_PX,
              left: "50%",
              transform: "translateX(-50%)",
            }
          : {
              right: -15,
              top: 18,
            }),
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
  isVertical,
}: {
  nodeId?: string;
  channel: string;
  onAppendFromNode?: AppendFromNodeHandler;
  hoveredEdge?: BlockEdgeState;
  connectingFrom?: BlockConnectionState;
  allEdges: BlockEdgeState[];
  isConnectionInteractive: boolean;
  isVertical: boolean;
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

  if (isVertical) {
    return (
      <>
        <AppendSourceHandle
          channel={channel}
          label="Add next component"
          onAppend={() => onAppendFromNode(nodeId, channel)}
          isHighlighted={isHighlighted}
          orientation="vertical"
          style={{
            bottom: -FACTORY_HANDLE_OUTSET_PX,
            left: "50%",
            transform: "translateX(-50%)",
          }}
          buttonLeft={-9}
          buttonTop={54}
        />
        <AppendHandlePreview
          orientation="vertical"
          style={{
            position: "absolute",
            left: "50%",
            top: "calc(100% + 85px)",
            transform: "translateX(-50%)",
          }}
        />
      </>
    );
  }

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

/** Target handle: left for horizontal canvases, top for vertical (factory) canvases. */
export function LeftHandle({
  data,
  nodeId,
  isConnectionInteractive = true,
}: Pick<BlockProps, "data" | "nodeId"> & { isConnectionInteractive?: boolean }) {
  const allEdges = getBlockEdges(data);
  const connectionStats = getNodeConnectionStats(allEdges, nodeId);
  const isVertical = isVerticalCanvasFlow(data._flowDirection);
  const useSideTarget = Boolean(data._factorySideTarget);

  if (shouldHideInactiveTargetHandle(isConnectionInteractive, connectionStats)) {
    return null;
  }

  if (shouldHideTargetHandle(data)) return null;

  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;
  const isAlreadyConnected = isAlreadyConnectedToNode(
    allEdges,
    connectingFrom,
    connectingFrom?.nodeId,
    nodeId,
    connectingFrom?.handleId,
  );
  const isHighlighted = getTargetHandleHighlight({
    isConnectionInteractive,
    hoveredEdge,
    connectingFrom,
    nodeId,
    isAlreadyConnected,
  });

  const position = useSideTarget ? Position.Left : isVertical ? Position.Top : Position.Left;
  const style: CSSProperties = useSideTarget
    ? {
        ...resolveHandleStyle(isVertical),
        left: -FACTORY_HANDLE_OUTSET_PX,
        top: "50%",
        transform: "translateY(-50%)",
        pointerEvents: connectionPointerEvents(isConnectionInteractive),
      }
    : {
        ...resolveHandleStyle(isVertical),
        ...(isVertical
          ? {
              top: -FACTORY_HANDLE_OUTSET_PX,
              left: "50%",
              transform: "translateX(-50%)",
            }
          : {
              left: -15,
              top: 18,
            }),
        pointerEvents: connectionPointerEvents(isConnectionInteractive),
      };

  return (
    <Handle type="target" position={position} style={style} className={isHighlighted ? "highlighted" : undefined} />
  );
}

function FactorySideSourceHandle({ isConnectionInteractive }: { isConnectionInteractive: boolean }) {
  return (
    <Handle
      type="source"
      position={Position.Right}
      id={FACTORY_SIDE_HANDLE_ID}
      style={{
        ...resolveHandleStyle(true),
        right: -FACTORY_HANDLE_OUTSET_PX,
        top: "50%",
        transform: "translateY(-50%)",
        pointerEvents: connectionPointerEvents(isConnectionInteractive),
      }}
    />
  );
}

function FactorySpineSourceHandle({ isConnectionInteractive }: { isConnectionInteractive: boolean }) {
  return (
    <Handle
      type="source"
      position={Position.Bottom}
      id={FACTORY_SPINE_HANDLE_ID}
      style={{
        ...resolveHandleStyle(true),
        bottom: -FACTORY_HANDLE_OUTSET_PX,
        left: "50%",
        transform: "translateX(-50%)",
        pointerEvents: connectionPointerEvents(isConnectionInteractive),
      }}
    />
  );
}

/** Source handle: right for horizontal canvases, bottom for vertical (factory) canvases. */
export function RightHandle({
  data,
  nodeId,
  isConnectionInteractive = true,
  onAppendFromNode,
}: Pick<BlockProps, "data" | "nodeId" | "onAppendFromNode"> & { isConnectionInteractive?: boolean }) {
  const allEdges = getBlockEdges(data);
  const { hasIncoming, hasOutgoing } = getNodeConnectionStats(allEdges, nodeId);
  const isDisconnected = isDisconnectedNode({ hasIncoming, hasOutgoing });
  const isVertical = isVerticalCanvasFlow(data._flowDirection);

  if (!isConnectionInteractive && (!hasOutgoing || isDisconnected)) {
    return null;
  }

  if (shouldHideSourceHandle(data)) return null;

  const channels = getOutputChannels(data);
  const hoveredEdge = data._hoveredEdge;
  const connectingFrom = data._connectingFrom;

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
        isVertical={isVertical}
      />
    );
  }

  const sideSource = data._factorySideSource ? (
    <FactorySideSourceHandle isConnectionInteractive={isConnectionInteractive} />
  ) : null;

  // Run-inspection compact fork: centered spine + side — no MultiBottom S-stems.
  if (data._factoryCompactFork) {
    return (
      <>
        {data._factorySpineSource ? (
          <FactorySpineSourceHandle isConnectionInteractive={isConnectionInteractive} />
        ) : null}
        {sideSource}
      </>
    );
  }

  if (channels.length === 1) {
    return (
      <>
        <SingleSourceHandle
          channel={channels[0]}
          hoveredEdge={hoveredEdge}
          connectingFrom={connectingFrom}
          nodeId={nodeId}
          allEdges={allEdges}
          isConnectionInteractive={isConnectionInteractive}
          isVertical={isVertical}
        />
        {sideSource}
      </>
    );
  }

  if (isVertical) {
    return (
      <>
        <MultiBottomHandle
          channels={channels}
          hoveredEdge={hoveredEdge}
          connectingFrom={connectingFrom}
          nodeId={nodeId}
          allEdges={allEdges}
          isConnectionInteractive={isConnectionInteractive}
          onAppendFromNode={onAppendFromNode}
        />
        {sideSource}
      </>
    );
  }

  return (
    <>
      <MultiRightHandle
        channels={channels}
        hoveredEdge={hoveredEdge}
        connectingFrom={connectingFrom}
        nodeId={nodeId}
        allEdges={allEdges}
        isConnectionInteractive={isConnectionInteractive}
        onAppendFromNode={onAppendFromNode}
      />
      {sideSource}
    </>
  );
}
