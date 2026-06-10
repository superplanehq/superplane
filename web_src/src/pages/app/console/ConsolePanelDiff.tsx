import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import type { DraftConsoleDiffItem } from "../draftConsoleDiff";
import type { DraftDiffStatus } from "../draftNodeDiff";
import { UnifiedDiffView } from "../UnifiedDiffView";
import { PANEL_DIFF_BADGE } from "./consolePanelDiffPresentation";

export function ConsolePanelDiffBadge({
  status,
  panelTitle,
  onShowDiff,
}: {
  status: DraftDiffStatus;
  panelTitle: string;
  onShowDiff: () => void;
}) {
  const badge = PANEL_DIFF_BADGE[status];
  const Icon = badge.Icon;

  return (
    <button
      type="button"
      aria-label={`See ${panelTitle} diff`}
      title={`See ${panelTitle} diff`}
      className={cn(
        "console-grid-no-drag absolute bottom-2 left-2 z-10 inline-flex w-fit max-w-max items-center gap-1 whitespace-nowrap rounded-full px-2 py-0.5 text-[10px] font-semibold tracking-wide shadow-sm outline-none hover:brightness-95 focus-visible:ring-2",
        badge.className,
      )}
      onClick={(event) => {
        event.stopPropagation();
        onShowDiff();
      }}
    >
      <Icon className="h-3 w-3" />
      <span>{badge.label}</span>
    </button>
  );
}

export function ConsolePanelDiffDialog({
  item,
  onOpenChange,
}: {
  item: DraftConsoleDiffItem | null;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={!!item} onOpenChange={onOpenChange}>
      <DialogContent className="min-w-[60vw] max-w-5xl max-h-[92vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-base">{item?.title || "Panel diff"}</DialogTitle>
          <DialogDescription>Console panel diff for {item?.changeType || "changed"} panel.</DialogDescription>
        </DialogHeader>
        {item ? (
          <UnifiedDiffView diffId={item.id} emptyMessage="No diff available for this panel." lines={item.lines} />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
