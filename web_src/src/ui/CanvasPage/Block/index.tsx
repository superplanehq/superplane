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
    className: "border-emerald-200 bg-emerald-500 text-white",
    ringClassName: "ring-2 ring-emerald-400",
    Icon: Plus,
  },
  updated: {
    label: "EDITED",
    className: "border-sky-200 bg-sky-500 text-white",
    ringClassName: "ring-2 ring-sky-400",
    Icon: Diff,
  },
  removed: {
    label: "REMOVED",
    className: "border-rose-200 bg-rose-500 text-white",
    ringClassName: "ring-2 ring-rose-400",
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
    <div className="nodrag absolute -bottom-3 right-2 z-10 flex items-center gap-1">
      {showDiffAction ? (
        <button
          type="button"
          className="flex items-center gap-1 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-[10px] font-medium text-slate-600 opacity-0 shadow-sm transition hover:bg-slate-50 focus:opacity-100 focus:outline-none focus:ring-2 focus:ring-sky-200 group-hover/block:opacity-100"
          onClick={(event) => {
            event.stopPropagation();
            onShowDiff();
          }}
        >
          <Eye className="h-3 w-3" />
          See diff
        </button>
      ) : null}
      <div
        className={cn(
          "flex items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] font-semibold tracking-wide shadow-sm",
          badge.className,
        )}
      >
        <Icon className="h-3 w-3" />
        <span>{badge.label}</span>
      </div>
    </div>
  );
}

export const Block = React.memo(function Block(props: BlockProps) {
  const data = props.data;
  const isHighlighted = data._isHighlighted || false;
  const hasHighlightedNodes = data._hasHighlightedNodes || false;
  const shouldFade = hasHighlightedNodes && !isHighlighted;
  const isRemoved = data._draftDiffStatus === "removed";
  const shouldBlankBody = data._dimBodyBelowHeader || isRemoved;
  const isConnectionInteractive = props.canvasMode !== "live" && !isRemoved;
  const diffRingClassName = data._draftDiffStatus ? DRAFT_DIFF_BADGE[data._draftDiffStatus].ringClassName : undefined;

  return (
    <div
      className={cn(
        "group/block relative w-fit",
        shouldFade && !shouldBlankBody && "opacity-30",
        isRemoved && "pointer-events-none opacity-50",
      )}
      onClick={(e) => props.onClick?.(e)}
    >
      <div className={cn("relative z-[1] w-fit rounded-lg", diffRingClassName)}>
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
