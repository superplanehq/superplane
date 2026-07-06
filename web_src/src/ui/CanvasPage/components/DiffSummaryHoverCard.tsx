import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Checkbox } from "@/ui/checkbox";
import { Switch } from "@/ui/switch";
import { Eye } from "lucide-react";

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
      {added > 0 && <span className="tabular-nums text-emerald-600 dark:text-emerald-400">+{added}</span>}
      {updated > 0 && <span className="tabular-nums text-sky-600 dark:text-sky-400">±{updated}</span>}
      {removed > 0 && <span className="tabular-nums text-red-600 dark:text-red-400">-{removed}</span>}
    </>
  );
}

const diffMenuLabelClassName = "text-[13px] font-normal text-gray-800 dark:text-gray-100";

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
          className="flex h-7 cursor-default items-center gap-0.5 rounded-full border border-slate-950/15 bg-white px-2.5 py-1 text-[13px] font-medium transition-colors hover:bg-slate-50 dark:border-gray-600/70 dark:bg-gray-800 dark:hover:bg-gray-700"
        >
          <DiffBadgeSegments diffCounts={diffCounts} />
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-auto p-0">
        <div className="flex flex-col gap-2 p-3">
          {onToggleVisualDiff && (
            <div className="flex items-center gap-1.5">
              <Switch
                id="visual-diff-toggle"
                checked={!!visualDiffEnabled}
                onCheckedChange={onToggleVisualDiff}
                data-testid="canvas-toggle-visual-diff"
              />
              <label htmlFor="visual-diff-toggle" className={diffMenuLabelClassName}>
                Diff X-Ray
              </label>
            </div>
          )}
          {diffToggles && visualDiffEnabled && (
            <>
              <div className="flex items-center gap-1.5">
                <Checkbox
                  id="show-deleted-nodes"
                  checked={diffToggles.showDeletedNodes}
                  onCheckedChange={diffToggles.toggleShowDeletedNodes}
                />
                <label htmlFor="show-deleted-nodes" className={diffMenuLabelClassName}>
                  Show deleted nodes
                </label>
              </div>
              <div className="flex items-center gap-1.5">
                <Checkbox
                  id="show-edge-diff"
                  checked={diffToggles.showEdgeDiff}
                  onCheckedChange={diffToggles.toggleShowEdgeDiff}
                />
                <label htmlFor="show-edge-diff" className={diffMenuLabelClassName}>
                  Show edges
                </label>
              </div>
            </>
          )}
        </div>
        {onShowDiff && (
          <div className="border-t border-slate-950/15 p-2 dark:border-gray-800/70">
            <button
              type="button"
              onClick={onShowDiff}
              className="flex w-full items-center gap-1.5 rounded-md p-1 text-left transition-colors hover:bg-slate-50 dark:hover:bg-gray-800"
              data-testid="canvas-show-diff-button"
            >
              <Eye className="h-4 w-4 shrink-0 text-gray-800 dark:text-gray-100" />
              <span className={diffMenuLabelClassName}>See Full Diff</span>
            </button>
          </div>
        )}
      </HoverCardContent>
    </HoverCard>
  );
}
