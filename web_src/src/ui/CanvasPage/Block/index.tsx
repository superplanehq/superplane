import React from "react";
import { cn } from "@/lib/utils";
import { Diff, Eye, Minus, Plus } from "lucide-react";
import { BlockContent } from "./content";
import { LeftHandle, RightHandle } from "./handles";
import type { BlockProps } from "./types";

export type { BlockData, BlockProps } from "./types";
export type { CanvasBlockData } from "./types";

const DRAFT_DIFF_BADGE = {
  added: {
    label: "ADDED",
    className: "bg-green-500 text-white",
    Icon: Plus,
  },
  updated: {
    label: "EDITED",
    className: "bg-sky-500 text-white",
    Icon: Diff,
  },
  removed: {
    label: "REMOVED",
    className: "bg-red-500 text-white",
    Icon: Minus,
  },
} as const;

function DraftDiffBadge({
  onShowDiff,
  status,
}: {
  onShowDiff?: () => void;
  status: NonNullable<BlockProps["data"]["_draftDiffStatus"]>;
}) {
  const badge = DRAFT_DIFF_BADGE[status];
  const Icon = badge.Icon;
  const showDiffAction = status === "updated" && onShowDiff;

  return (
    <div className="nodrag absolute top-full right-2 z-10 flex flex-col items-end gap-1">
      <div
        className={cn(
          "flex items-center gap-1 rounded-t-none rounded-b-md px-2 py-0.5 text-[10px] font-semibold tracking-wide shadow-sm",
          badge.className,
        )}
      >
        <Icon className="h-3 w-3" />
        <span>{badge.label}</span>
      </div>
      {showDiffAction ? (
        <button
          type="button"
          className="mt-1 flex items-center gap-1 rounded-full bg-white px-2 py-0.5 text-[10px] font-medium text-slate-600 opacity-0 outline outline-1 outline-slate-950/15 transition hover:bg-slate-50 focus:opacity-100 focus:outline-none focus:ring-2 focus:ring-sky-200 group-hover/block:opacity-100"
          onClick={(event) => {
            event.stopPropagation();
            onShowDiff();
          }}
        >
          <Eye className="h-3 w-3" />
          See diff
        </button>
      ) : null}
    </div>
  );
}

export const Block = React.memo(function Block(props: BlockProps) {
  const data = props.data;
  const isHighlighted = data._isHighlighted || false;
  const hasHighlightedNodes = data._hasHighlightedNodes || false;
  const isAnnotation = data.type === "annotation";
  const shouldFade = !isAnnotation && hasHighlightedNodes && !isHighlighted;
  const isRemoved = data._draftDiffStatus === "removed";
  const shouldBlankBody = !isAnnotation && (data._dimBodyBelowHeader || isRemoved);
  const isConnectionInteractive = props.canvasMode !== "live" && !isRemoved;

  return (
    <div
      className={cn(
        "group/block relative w-fit",
        shouldFade && !shouldBlankBody && "opacity-30",
        isRemoved && "pointer-events-none opacity-50",
      )}
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
      {data._draftDiffStatus ? <DraftDiffBadge status={data._draftDiffStatus} onShowDiff={props.onShowDiff} /> : null}
    </div>
  );
});
