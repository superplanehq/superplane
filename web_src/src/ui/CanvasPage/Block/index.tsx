import React from "react";

import { useCanvasNodeOverlay } from "../canvasNodeOverlayContext";
import { BlockContent } from "./content";
import { LeftHandle, RightHandle } from "./handles";
import type { BlockProps } from "./types";

export type { BlockData, BlockProps } from "./types";
export type { CanvasBlockData } from "./types";

export const Block = React.memo(function Block(props: BlockProps) {
  const overlay = useCanvasNodeOverlay();
  const data = props.data;
  const nodeId = props.nodeId ?? "";
  const isHighlighted =
    (overlay && overlay.highlightedNodeIds.has(nodeId)) || data._isHighlighted || false;
  const hasHighlightedNodes = overlay?.hasHighlightedNodes || data._hasHighlightedNodes || false;
  const shouldDim = hasHighlightedNodes && !isHighlighted;
  const isConnectionInteractive = props.canvasMode !== "live";

  return (
    <div className={`relative w-fit ${shouldDim ? "opacity-30" : ""}`} onClick={(e) => props.onClick?.(e)}>
      <LeftHandle data={data} nodeId={props.nodeId} isConnectionInteractive={isConnectionInteractive} />
      <BlockContent {...props} />
      <RightHandle data={data} nodeId={props.nodeId} isConnectionInteractive={isConnectionInteractive} />
    </div>
  );
});
