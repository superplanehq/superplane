import type { CanvasesCanvasDraftBranch } from "@/api-client";
import { List, Pencil, Plus, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { TimeAgo } from "@/components/TimeAgo";
import { Button as UIButton } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { cn } from "@/lib/utils";

export type StartEditingDropdownProps = {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  drafts: CanvasesCanvasDraftBranch[];
  defaultDraft: CanvasesCanvasDraftBranch | null;
  disabled?: boolean;
  disabledTooltip?: string;
  isSubmitting?: boolean;
  onContinueDraft: (branchName: string) => void;
  onCreateDraft: () => void;
};

function shortSha(sha?: string): string {
  if (!sha) {
    return "—";
  }

  return sha.slice(0, 7);
}

function formatUpdatedAt(value?: string): string {
  if (!value) {
    return "Unknown";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Unknown";
  }

  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

export function StartEditingDropdown({
  open,
  onOpenChange,
  drafts,
  defaultDraft,
  disabled,
  isSubmitting,
  onContinueDraft,
  onCreateDraft,
}: StartEditingDropdownProps) {
  const [showList, setShowList] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const filteredDrafts = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) {
      return drafts;
    }

    return drafts.filter((draft) => {
      const displayName = (draft.displayName || draft.branchName || "").toLowerCase();
      const owner = (draft.owner?.name || "").toLowerCase();
      return (
        displayName.includes(query) || owner.includes(query) || (draft.branchName || "").toLowerCase().includes(query)
      );
    });
  }, [drafts, searchQuery]);

  const draftCount = drafts.length;
  const continueDraft = defaultDraft ?? drafts[0] ?? null;
  const continueLabel = continueDraft?.displayName || continueDraft?.branchName || "draft";

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setShowList(false);
      setSearchQuery("");
    }
    onOpenChange?.(nextOpen);
  };

  const closeAndRun = (action: () => void) => {
    action();
    handleOpenChange(false);
  };

  // With no existing drafts there is nothing to choose between, so clicking Edit
  // creates a draft directly instead of opening a menu.
  if (draftCount === 0) {
    return (
      <UIButton
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled || isSubmitting}
        data-testid="canvas-edit-button"
        onClick={onCreateDraft}
      >
        Edit
      </UIButton>
    );
  }

  return (
    <DropdownMenu open={open} onOpenChange={handleOpenChange}>
      <DropdownMenuTrigger asChild>
        <UIButton type="button" variant="outline" size="sm" disabled={disabled} data-testid="canvas-edit-button">
          Edit
        </UIButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-72 px-2" data-testid="start-editing-menu">
        {!showList ? (
          <>
            <div className="px-3 pt-3 pb-2">
              <div className="text-sm font-medium text-slate-900">You have an unpublished draft</div>
              {continueDraft?.updatedAt || continueDraft?.createdAt ? (
                <div className="text-xs text-slate-500">
                  Last edited <TimeAgo date={(continueDraft.updatedAt || continueDraft.createdAt)!} />
                </div>
              ) : null}
            </div>
            <DropdownMenuSeparator className="my-0" />
            <div className="py-1">
              {draftCount >= 1 && continueDraft?.branchName ? (
                <DropdownMenuItem
                  className="cursor-pointer gap-2 px-3 py-2"
                  disabled={isSubmitting}
                  data-testid="start-editing-continue"
                  onClick={() => closeAndRun(() => onContinueDraft(continueDraft.branchName!))}
                >
                  <Pencil className="h-4 w-4" />
                  <span>Continue {continueLabel}</span>
                </DropdownMenuItem>
              ) : null}
              <DropdownMenuItem
                className="cursor-pointer gap-2 px-3 py-2"
                disabled={isSubmitting}
                data-testid="start-editing-create"
                onClick={() => closeAndRun(onCreateDraft)}
              >
                <Plus className="h-4 w-4" />
                <span>Create new draft</span>
              </DropdownMenuItem>
              {draftCount >= 2 ? (
                <DropdownMenuItem
                  className="cursor-pointer gap-2 px-3 py-2"
                  disabled={isSubmitting}
                  data-testid="start-editing-choose-list"
                  onSelect={(event) => event.preventDefault()}
                  onClick={() => setShowList(true)}
                >
                  <List className="h-4 w-4" />
                  <span>Choose from list…</span>
                </DropdownMenuItem>
              ) : null}
            </div>
          </>
        ) : (
          <>
            <div className="px-3 pt-3 pb-2">
              <div className="text-sm font-medium text-slate-900">Choose a draft</div>
              <div className="text-xs text-slate-500">Select a draft branch to continue editing.</div>
            </div>
            <div className="px-3 pb-2">
              <div className="relative">
                <Search className="pointer-events-none absolute left-2.5 top-2.5 h-4 w-4 text-slate-400" aria-hidden />
                <Input
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
                  placeholder="Search drafts…"
                  className="h-8 pl-8"
                  aria-label="Search drafts"
                />
              </div>
            </div>
            <DropdownMenuSeparator className="my-0" />
            <div className="max-h-64 overflow-auto py-1">
              {filteredDrafts.length === 0 ? (
                <p className="px-3 py-2 text-sm text-slate-600">No drafts match your search.</p>
              ) : (
                filteredDrafts.map((draft) => (
                  <DropdownMenuItem
                    key={draft.branchName}
                    className={cn(
                      "cursor-pointer flex-col items-start gap-0.5 px-3 py-2",
                      draft.branchName === continueDraft?.branchName ? "bg-blue-50" : "",
                    )}
                    data-testid="start-editing-draft-row"
                    onClick={() => {
                      if (draft.branchName) {
                        closeAndRun(() => onContinueDraft(draft.branchName!));
                      }
                    }}
                  >
                    <span className="text-sm font-medium text-slate-900">
                      {draft.displayName || draft.branchName || "Draft"}
                    </span>
                    <span className="text-xs text-slate-500">
                      {draft.owner?.name || "Unknown"} · {shortSha(draft.tipSha)} ·{" "}
                      {formatUpdatedAt(draft.updatedAt || draft.createdAt)}
                    </span>
                  </DropdownMenuItem>
                ))
              )}
            </div>
            <DropdownMenuSeparator className="my-0" />
            <div className="py-1">
              <DropdownMenuItem
                className="cursor-pointer px-3 py-2"
                onSelect={(event) => event.preventDefault()}
                onClick={() => setShowList(false)}
              >
                Back
              </DropdownMenuItem>
            </div>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
