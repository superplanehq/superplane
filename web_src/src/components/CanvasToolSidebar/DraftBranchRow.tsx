import type { CanvasesCanvasDraftBranch } from "@/api-client";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

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

export function DraftBranchRow({
  draft,
  isActive,
  canUpdateCanvas,
  deletePending,
  onOpen,
  onDelete,
}: {
  draft: CanvasesCanvasDraftBranch;
  isActive: boolean;
  canUpdateCanvas: boolean;
  deletePending?: boolean;
  onOpen: (branchName: string) => void;
  onDelete?: (branchName: string) => void;
}) {
  const branchName = draft.branchName || "";
  const displayName = draft.displayName || branchName || "Draft";
  const ownerName = draft.owner?.name || "Unknown";

  return (
    <div
      className={cn("flex items-start gap-2 border-b border-slate-100 px-4 py-3", isActive ? "bg-blue-50" : "bg-white")}
      data-testid="canvas-draft-branch-row"
    >
      <button
        type="button"
        className="min-w-0 flex-1 text-left"
        onClick={() => branchName && onOpen(branchName)}
        disabled={!branchName}
      >
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-medium text-slate-900">{displayName}</span>
          {isActive ? (
            <span className="rounded bg-blue-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-blue-700">
              Active
            </span>
          ) : null}
        </div>
        <p className="mt-0.5 truncate text-xs text-slate-500">{branchName}</p>
        <p className="mt-1 text-xs text-slate-500">
          {ownerName} · {shortSha(draft.tipSha)} · {formatUpdatedAt(draft.updatedAt || draft.createdAt)}
        </p>
        {draft.materializationStatus && draft.materializationStatus !== "ready" ? (
          <p className="mt-1 text-xs text-amber-700">Materialization: {draft.materializationStatus}</p>
        ) : null}
      </button>
      {canUpdateCanvas && onDelete && branchName ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-8 w-8 shrink-0 text-slate-500 hover:text-red-600"
          aria-label={`Delete ${displayName}`}
          disabled={deletePending}
          onClick={() => onDelete(branchName)}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      ) : null}
    </div>
  );
}
