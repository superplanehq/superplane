import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Checkbox } from "@/ui/checkbox";
import { Switch } from "@/ui/switch";
import { Minus, Pencil, Plus } from "lucide-react";

interface DiffSummaryHoverCardProps {
  diffCounts: { added: number; updated: number; removed: number };
  visualDiffEnabled?: boolean;
  onToggleVisualDiff?: () => void;
  showDeletedNodes?: boolean;
  onToggleShowDeletedNodes?: () => void;
  showEdgeDiff?: boolean;
  onToggleShowEdgeDiff?: () => void;
  onShowDiff?: () => void;
}

function DiffBadgeSegments({ diffCounts }: Pick<DiffSummaryHoverCardProps, "diffCounts">) {
  const { added, updated, removed } = diffCounts;
  return (
    <>
      {added > 0 && (
        <span className="flex items-center gap-0.5 px-1 text-emerald-600">
          <Plus className="h-3 w-3" />
          {added}
        </span>
      )}
      {added > 0 && (updated > 0 || removed > 0) && <span className="text-slate-300">|</span>}
      {updated > 0 && (
        <span className="flex items-center gap-0.5 px-1 text-sky-600">
          <Pencil className="h-3 w-3" />
          {updated}
        </span>
      )}
      {updated > 0 && removed > 0 && <span className="text-slate-300">|</span>}
      {removed > 0 && (
        <span className="flex items-center gap-0.5 px-1 text-red-600">
          <Minus className="h-3 w-3" />
          {removed}
        </span>
      )}
    </>
  );
}

export function DiffSummaryHoverCard({
  diffCounts,
  visualDiffEnabled,
  onToggleVisualDiff,
  showDeletedNodes,
  onToggleShowDeletedNodes,
  showEdgeDiff,
  onToggleShowEdgeDiff,
  onShowDiff,
}: DiffSummaryHoverCardProps) {
  const hasChanges = diffCounts.added > 0 || diffCounts.updated > 0 || diffCounts.removed > 0;
  if (!hasChanges) return null;

  return (
    <HoverCard openDelay={100} closeDelay={200}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          className="flex cursor-default items-center gap-0 rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-xs font-medium transition-colors hover:bg-slate-100"
        >
          <DiffBadgeSegments diffCounts={diffCounts} />
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-auto p-3">
        <div className="flex flex-col gap-2">
          {onToggleVisualDiff && (
            <div className="flex items-center gap-1.5 text-xs font-medium text-slate-600">
              <Switch
                id="visual-diff-toggle"
                checked={!!visualDiffEnabled}
                onCheckedChange={onToggleVisualDiff}
                data-testid="canvas-toggle-visual-diff"
              />
              <label htmlFor="visual-diff-toggle">Diff X-Ray</label>
            </div>
          )}
          {onToggleShowDeletedNodes && (
            <div
              className={`flex items-center gap-1.5 text-xs font-medium ${visualDiffEnabled ? "text-slate-600" : "text-slate-400"}`}
            >
              <Checkbox
                id="show-deleted-nodes"
                checked={!!showDeletedNodes}
                onCheckedChange={onToggleShowDeletedNodes}
                disabled={!visualDiffEnabled}
              />
              <label htmlFor="show-deleted-nodes">Show deleted nodes</label>
            </div>
          )}
          {onToggleShowEdgeDiff && (
            <div
              className={`flex items-center gap-1.5 text-xs font-medium ${visualDiffEnabled ? "text-slate-600" : "text-slate-400"}`}
            >
              <Checkbox
                id="show-edge-diff"
                checked={!!showEdgeDiff}
                onCheckedChange={onToggleShowEdgeDiff}
                disabled={!visualDiffEnabled}
              />
              <label htmlFor="show-edge-diff">Show edges</label>
            </div>
          )}
        </div>
        {onShowDiff && (
          <div className="-mx-3 -mb-3 mt-2 border-t border-slate-200 px-3 py-2">
            <button
              type="button"
              onClick={onShowDiff}
              className="text-xs font-medium text-blue-600 underline-offset-2 hover:text-blue-700 hover:underline"
              data-testid="canvas-show-diff-button"
            >
              View full diff
            </button>
          </div>
        )}
      </HoverCardContent>
    </HoverCard>
  );
}
