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
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";

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
  const rowTitle = [displayName, branchName, ownerName, formatUpdatedAt(draftUpdatedAt(draft))]
    .filter(Boolean)
    .join(" · ");

  return (
    <div
      className={cn(RUNS_SIDEBAR_ROW_CLASS, draftBranchRowBackgroundClassName(isActive, editStatus))}
      data-testid="canvas-draft-branch-row"
      title={rowTitle}
    >
      <button
        type="button"
        className="flex min-w-0 flex-1 items-center gap-1.5 text-left"
        onClick={() => branchName && onOpen(branchName)}
        disabled={!branchName}
      >
        <span className="min-w-0 flex-1 truncate text-xs font-medium text-slate-900">{displayName}</span>
        <span className={cn(statusBadge.className, "shrink-0")}>{statusBadge.label}</span>
      </button>
      {canUpdateCanvas && onDelete && versionId ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 shrink-0 text-slate-500 hover:text-red-600"
          aria-label={`Delete ${displayName}`}
          disabled={deletePending}
          onClick={() => onDelete(versionId)}
        >
          <Trash2 className="size-4" />
        </Button>
      ) : null}
    </div>
  );
}
