import type { CanvasesCanvasVersion } from "@/api-client";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { draftBranchName, draftDisplayName, draftOwnerName, draftUpdatedAt, draftVersionId } from "@/lib/draftVersion";
import {
  draftBranchRowBackgroundClassName,
  draftBranchStatusBadge,
  type DraftBranchEditStatus,
} from "@/pages/app/lib/draft-branch-edit-status";

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
  editStatus = "ready",
  canUpdateCanvas,
  deletePending,
  onOpen,
  onDelete,
}: {
  draft: CanvasesCanvasVersion;
  isActive: boolean;
  editStatus?: DraftBranchEditStatus;
  canUpdateCanvas: boolean;
  deletePending?: boolean;
  onOpen: (branchName: string) => void;
  onDelete?: (versionId: string) => void;
}) {
  const branchName = draftBranchName(draft);
  const displayName = draftDisplayName(draft);
  const ownerName = draftOwnerName(draft);
  const versionId = draftVersionId(draft);
  const statusBadge = draftBranchStatusBadge(editStatus, isActive);

  return (
    <div
      className={cn(
        "flex items-start gap-2 border-b border-slate-100 px-4 py-3",
        draftBranchRowBackgroundClassName(isActive, editStatus),
      )}
      data-testid="canvas-draft-branch-row"
    >
      <button
        type="button"
        className="min-w-0 flex-1 text-left"
        onClick={() => branchName && onOpen(branchName)}
        disabled={!branchName}
      >
        <div className="flex flex-wrap items-center gap-2">
          <span className="truncate text-sm font-medium text-slate-900">{displayName}</span>
          <span className={statusBadge.className}>{statusBadge.label}</span>
        </div>
        <p className="mt-0.5 truncate text-xs text-slate-500">{branchName}</p>
        <p className="mt-1 text-xs text-slate-500">
          {ownerName} · {formatUpdatedAt(draftUpdatedAt(draft))}
        </p>
      </button>
      {canUpdateCanvas && onDelete && versionId ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-8 w-8 shrink-0 text-slate-500 hover:text-red-600"
          aria-label={`Delete ${displayName}`}
          disabled={deletePending}
          onClick={() => onDelete(versionId)}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      ) : null}
    </div>
  );
}
