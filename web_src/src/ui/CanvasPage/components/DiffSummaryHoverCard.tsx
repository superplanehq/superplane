import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Checkbox } from "@/ui/checkbox";
import { Switch } from "@/ui/switch";

interface DiffSummaryHoverCardProps {
  diffCounts: { added: number; updated: number; removed: number };
  visualDiffEnabled?: boolean;
  onToggleVisualDiff?: () => void;
  diffToggles?: {
    showDeletedNodes: boolean;
    toggleShowDeletedNodes: () => void;
    showEdgeDiff: boolean;
    toggleShowEdgeDiff: () => void;
  };
  onShowDiff?: () => void;
}

function DiffBadgeSegments({ diffCounts }: Pick<DiffSummaryHoverCardProps, "diffCounts">) {
  const { added, updated, removed } = diffCounts;
  return (
    <>
      {added > 0 && <span className="tabular-nums text-emerald-600">+{added}</span>}
      {updated > 0 && <span className="tabular-nums text-sky-600">±{updated}</span>}
      {removed > 0 && <span className="tabular-nums text-red-600">-{removed}</span>}
    </>
  );
}

export function DiffSummaryHoverCard({
  diffCounts,
  visualDiffEnabled,
  onToggleVisualDiff,
  diffToggles,
  onShowDiff,
}: DiffSummaryHoverCardProps) {
  const hasChanges = diffCounts.added > 0 || diffCounts.updated > 0 || diffCounts.removed > 0;
  if (!hasChanges) return null;

  return (
    <HoverCard openDelay={100} closeDelay={200}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          className="flex h-7 cursor-default items-center gap-0.5 rounded-md border border-slate-950/15 bg-white px-1.5 py-1 text-[13px] font-medium transition-colors hover:bg-slate-50"
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
          {diffToggles && (
            <>
              <div
                className={`flex items-center gap-1.5 text-xs font-medium ${visualDiffEnabled ? "text-slate-600" : "text-slate-400"}`}
              >
                <Checkbox
                  id="show-deleted-nodes"
                  checked={diffToggles.showDeletedNodes}
                  onCheckedChange={diffToggles.toggleShowDeletedNodes}
                  disabled={!visualDiffEnabled}
                />
                <label htmlFor="show-deleted-nodes">Show deleted nodes</label>
              </div>
              <div
                className={`flex items-center gap-1.5 text-xs font-medium ${visualDiffEnabled ? "text-slate-600" : "text-slate-400"}`}
              >
                <Checkbox
                  id="show-edge-diff"
                  checked={diffToggles.showEdgeDiff}
                  onCheckedChange={diffToggles.toggleShowEdgeDiff}
                  disabled={!visualDiffEnabled}
                />
                <label htmlFor="show-edge-diff">Show edges</label>
              </div>
            </>
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
