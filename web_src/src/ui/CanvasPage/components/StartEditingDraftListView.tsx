import type { CanvasesCanvasVersion } from "@/api-client";
import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { DropdownMenuItem, DropdownMenuSeparator } from "@/ui/dropdownMenu";
import { cn } from "@/lib/utils";
import { draftBranchName, draftDisplayName, draftOwnerName, draftUpdatedAt } from "@/lib/draftVersion";

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

type StartEditingDraftListViewProps = {
  filteredDrafts: CanvasesCanvasVersion[];
  continueBranchName: string;
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  onContinueDraft: (branchName: string) => void;
  onBack: () => void;
};

export function StartEditingDraftListView({
  filteredDrafts,
  continueBranchName,
  searchQuery,
  onSearchQueryChange,
  onContinueDraft,
  onBack,
}: StartEditingDraftListViewProps) {
  return (
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
            onChange={(event) => onSearchQueryChange(event.target.value)}
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
          filteredDrafts.map((draft) => {
            const branchName = draftBranchName(draft);
            return (
              <DropdownMenuItem
                key={branchName || draft.metadata?.id}
                className={cn(
                  "cursor-pointer flex-col items-start gap-0.5 px-3 py-2",
                  branchName === continueBranchName ? "bg-blue-50" : "",
                )}
                data-testid="start-editing-draft-row"
                onClick={() => {
                  if (branchName) {
                    onContinueDraft(branchName);
                  }
                }}
              >
                <span className="text-sm font-medium text-slate-900">{draftDisplayName(draft)}</span>
                <span className="text-xs text-slate-500">
                  {draftOwnerName(draft)} · {formatUpdatedAt(draftUpdatedAt(draft))}
                </span>
              </DropdownMenuItem>
            );
          })
        )}
      </div>
      <DropdownMenuSeparator className="my-0" />
      <div className="py-1">
        <DropdownMenuItem
          className="cursor-pointer px-3 py-2"
          onSelect={(event) => event.preventDefault()}
          onClick={onBack}
        >
          Back
        </DropdownMenuItem>
      </div>
    </>
  );
}
