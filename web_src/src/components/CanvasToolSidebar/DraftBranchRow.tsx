import type { CanvasesCanvasVersion } from "@/api-client";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { draftBranchName, draftDisplayName, draftOwnerName, draftUpdatedAt, draftVersionId } from "@/lib/draftVersion";

export type DraftBranchEditStatus = "uncommitted" | "ready" | "no-changes";

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

function rowBackgroundClassName(isActive: boolean, editStatus?: DraftBranchEditStatus): string {
  if (!editStatus) {
    return "bg-white";
  }

  if (!isActive) {
    return "bg-slate-50";
  }

  if (editStatus === "uncommitted") {
    return "bg-orange-50";
  }

  if (editStatus === "ready" || editStatus === "no-changes") {
    return "bg-blue-50";
  }

  return "bg-white";
}

const grayStatusBadgeClassName =
  "rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-600";
const activeBlueStatusBadgeClassName =
  "rounded bg-blue-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-blue-800";

function editStatusBadge(editStatus: DraftBranchEditStatus, isActive: boolean) {
  if (editStatus === "no-changes") {
    return {
      label: "No changes",
      className: isActive ? activeBlueStatusBadgeClassName : grayStatusBadgeClassName,
    };
  }

  if (editStatus === "uncommitted") {
    return {
      label: "Uncommitted changes",
      className: isActive
        ? "rounded bg-orange-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-orange-800"
        : grayStatusBadgeClassName,
    };
  }

  return {
    label: "Ready to publish",
    className: isActive ? activeBlueStatusBadgeClassName : grayStatusBadgeClassName,
  };
}

export function DraftBranchRow({
  draft,
  isActive,
  editStatus,
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
  const statusBadge = editStatus ? editStatusBadge(editStatus, isActive) : null;
  const versionId = draftVersionId(draft);

  return (
    <div
      className={cn(
        "flex items-start gap-2 border-b border-slate-100 px-4 py-3",
        rowBackgroundClassName(isActive, editStatus),
      )}
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
          {statusBadge ? <span className={statusBadge.className}>{statusBadge.label}</span> : null}
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
