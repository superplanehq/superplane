import React from "react";
import { cn } from "@/lib/utils";
import { BlockContent } from "./content";
import { LeftHandle, RightHandle } from "./handles";
import type { BlockProps } from "./types";

export type { BlockData, BlockProps } from "./types";
export type { CanvasBlockData } from "./types";

const DRAFT_DIFF_RING: Record<string, string> = {
  added: "ring-2 ring-green-500",
  updated: "ring-2 ring-orange-400",
  removed: "ring-2 ring-red-500 pointer-events-none",
};

export const Block = React.memo(function Block(props: BlockProps) {
  const data = props.data;
  const isHighlighted = data._isHighlighted || false;
  const hasHighlightedNodes = data._hasHighlightedNodes || false;
  const shouldFade = hasHighlightedNodes && !isHighlighted;
  const isDeleted = data._draftDiffStatus === "removed";
  const shouldBlankBody = data._dimBodyBelowHeader || isDeleted;
  const isConnectionInteractive = props.canvasMode !== "live";
  const diffRing = data._draftDiffStatus ? DRAFT_DIFF_RING[data._draftDiffStatus] || "" : "";

  return (
    <div
      className={cn("relative w-fit rounded-md", diffRing, shouldFade && !shouldBlankBody && "opacity-30")}
      onClick={(e) => props.onClick?.(e)}
    >
      <div className="relative z-[1] w-fit">
        <LeftHandle data={data} nodeId={props.nodeId} isConnectionInteractive={isConnectionInteractive} />
        <BlockContent {...props} dimBodyBelowHeader={shouldBlankBody} />
        <RightHandle
          data={data}
          nodeId={props.nodeId}
          isConnectionInteractive={isConnectionInteractive}
          onAppendFromNode={props.onAppendFromNode}
        />
      </div>
    </div>
  );
});
