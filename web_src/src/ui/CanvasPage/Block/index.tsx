import React from "react";
import { cn } from "@/lib/utils";
import { Plus, Minus, Diff } from "lucide-react";
import { BlockContent } from "./content";
import { LeftHandle, RightHandle } from "./handles";
import type { BlockProps } from "./types";

export type { BlockData, BlockProps } from "./types";
export type { CanvasBlockData } from "./types";

const DIFF_BADGE: Record<string, { label: string; bg: string; Icon: React.FC<{ className?: string }> }> = {
  added: { label: "ADDED", bg: "bg-green-500", Icon: Plus },
  updated: { label: "EDITED", bg: "bg-blue-500", Icon: Diff },
  removed: { label: "REMOVED", bg: "bg-red-500", Icon: Minus },
};

function DraftDiffBadge({ status }: { status: string }) {
  const badge = DIFF_BADGE[status];
  if (!badge) return null;
  const { Icon } = badge;
  return (
    <div
      className={cn(
        "absolute -bottom-3 right-2 z-10 flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold tracking-wide text-white shadow-sm",
        badge.bg,
      )}
    >
      <Icon className="h-3 w-3" />
      <span>{badge.label}</span>
    </div>
  );
}

export const Block = React.memo(function Block(props: BlockProps) {
  const data = props.data;
  const isHighlighted = data._isHighlighted || false;
  const hasHighlightedNodes = data._hasHighlightedNodes || false;
  const shouldFade = hasHighlightedNodes && !isHighlighted;
  const isDeleted = data._draftDiffStatus === "removed";
  const shouldBlankBody = data._dimBodyBelowHeader || isDeleted;
  const isConnectionInteractive = props.canvasMode !== "live";

  return (
    <div
      className={cn("relative w-fit", shouldFade && !shouldBlankBody && "opacity-30")}
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
      {data._draftDiffStatus && <DraftDiffBadge status={data._draftDiffStatus} />}
    </div>
  );
});
