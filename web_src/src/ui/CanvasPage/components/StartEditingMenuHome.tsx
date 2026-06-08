import type { CanvasesCanvasVersion } from "@/api-client";
import { List, Pencil, Plus } from "lucide-react";
import { TimeAgo } from "@/components/TimeAgo";
import { DropdownMenuItem, DropdownMenuSeparator } from "@/ui/dropdownMenu";
import { draftDisplayName, draftUpdatedAt } from "@/lib/draftVersion";

type StartEditingMenuHomeProps = {
  drafts: CanvasesCanvasVersion[];
  continueDraft: CanvasesCanvasVersion | null;
  continueBranchName: string;
  isSubmitting?: boolean;
  onContinueDraft: (branchName: string) => void;
  onCreateDraft: () => void;
  onShowList: () => void;
};

export function StartEditingMenuHome({
  drafts,
  continueDraft,
  continueBranchName,
  isSubmitting,
  onContinueDraft,
  onCreateDraft,
  onShowList,
}: StartEditingMenuHomeProps) {
  const continueLabel = continueDraft ? draftDisplayName(continueDraft) : "draft";

  return (
    <>
      <div className="px-3 pt-3 pb-2">
        <div className="text-sm font-medium text-slate-900">You have an unpublished draft</div>
        {continueDraft && draftUpdatedAt(continueDraft) ? (
          <div className="text-xs text-slate-500">
            Last edited <TimeAgo date={draftUpdatedAt(continueDraft)!} />
          </div>
        ) : null}
      </div>
      <DropdownMenuSeparator className="my-0" />
      <div className="py-1">
        {drafts.length >= 1 && continueBranchName ? (
          <DropdownMenuItem
            className="cursor-pointer gap-2 px-3 py-2"
            data-testid="start-editing-continue"
            onClick={() => onContinueDraft(continueBranchName)}
          >
            <Pencil className="h-4 w-4" />
            <span>Continue {continueLabel}</span>
          </DropdownMenuItem>
        ) : null}
        <DropdownMenuItem
          className="cursor-pointer gap-2 px-3 py-2"
          disabled={isSubmitting}
          data-testid="start-editing-create"
          onClick={onCreateDraft}
        >
          <Plus className="h-4 w-4" />
          <span>Create new draft</span>
        </DropdownMenuItem>
        {drafts.length >= 2 ? (
          <DropdownMenuItem
            className="cursor-pointer gap-2 px-3 py-2"
            data-testid="start-editing-choose-list"
            onSelect={(event) => event.preventDefault()}
            onClick={onShowList}
          >
            <List className="h-4 w-4" />
            <span>Choose from list…</span>
          </DropdownMenuItem>
        ) : null}
      </div>
    </>
  );
}
