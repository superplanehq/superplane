import { ChevronDown, ExternalLink, Hourglass } from "lucide-react";
import { Link } from "react-router";

import type { FactoriesWorkOrderResult, FactoriesWorkOrderState } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { appPath } from "@/lib/appPaths";
import { formatRelative } from "@/lib/datetime";
import { MarkdownContent } from "@/pages/app/Markdown";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import type { WorkOrderStatusNotePresentation } from "./lib/workOrderStatusNote";

interface WorkOrderStatusNoteProps {
  note: WorkOrderStatusNotePresentation;
  organizationId: string;
  canClose: boolean;
  canManage: boolean;
  isBusy: boolean;
  showManualUpdate: boolean;
  onClose: (result: FactoriesWorkOrderResult) => void;
  onStatusChange: (state: FactoriesWorkOrderState, result?: FactoriesWorkOrderResult) => Promise<void>;
}

export function WorkOrderStatusNote({
  note,
  organizationId,
  canClose,
  canManage,
  isBusy,
  showManualUpdate,
  onClose,
  onStatusChange,
}: WorkOrderStatusNoteProps) {
  return (
    <aside className="rounded-lg border bg-card px-4 py-4" aria-label="Next step">
      <div className="flex items-start gap-3">
        <span
          className="flex size-8 shrink-0 items-center justify-center rounded-full bg-[color:var(--status-waiting-dot)]/15"
          aria-hidden
        >
          <Hourglass className="size-4 text-[color:var(--status-waiting-fg)]" />
        </span>

        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
            <h3 className="text-[13px] font-semibold tracking-[-0.01em] text-foreground">{note.headline}</h3>
            <NoteAttribution note={note} organizationId={organizationId} />
          </div>

          <div className="mt-1 text-[13px]">
            <MarkdownContent content={note.text} variant="workspace" />
          </div>

          {note.cta || showManualUpdate ? (
            <div className="mt-3 flex flex-wrap items-center gap-2">
              {note.cta ? (
                <Button asChild size="sm">
                  <a href={note.cta.href} target="_blank" rel="noreferrer">
                    {note.cta.label}
                    <ExternalLink className="size-3.5" aria-hidden />
                  </a>
                </Button>
              ) : null}
              {showManualUpdate ? (
                <ManualUpdateMenu
                  canClose={canClose}
                  canManage={canManage}
                  isBusy={isBusy}
                  onClose={onClose}
                  onStatusChange={onStatusChange}
                />
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </aside>
  );
}

/** "PR Closure · 25 minutes ago", with the automation name linked. */
function NoteAttribution({ note, organizationId }: { note: WorkOrderStatusNotePresentation; organizationId: string }) {
  const time = note.updatedAt ? formatRelative(note.updatedAt) : null;
  if (!note.source && !time) {
    return null;
  }

  return (
    <span className="flex shrink-0 items-baseline gap-1 text-[11px] text-muted-foreground">
      {note.source?.appId ? (
        <Link
          to={appPath(organizationId, note.source.appId)}
          className="text-muted-foreground underline underline-offset-2 hover:text-foreground hover:no-underline"
        >
          {note.source.name}
        </Link>
      ) : (
        (note.source?.name ?? null)
      )}
      {note.source && time ? <span aria-hidden>·</span> : null}
      {time}
    </span>
  );
}

function ManualUpdateMenu({
  canClose,
  canManage,
  isBusy,
  onClose,
  onStatusChange,
}: Pick<WorkOrderStatusNoteProps, "canClose" | "canManage" | "isBusy" | "onClose" | "onStatusChange">) {
  return (
    <DropdownMenu>
      <PermissionTooltip allowed={canClose || canManage} message="You don't have permission to manage this work order.">
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={isBusy || (!canClose && !canManage)}
            className="text-muted-foreground"
          >
            Update manually
            <ChevronDown className="size-3.5" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
      </PermissionTooltip>
      <DropdownMenuContent align="start" className="w-44">
        <DropdownMenuItem disabled={!canClose} onSelect={() => onClose("RESULT_COMPLETED")}>
          Complete
        </DropdownMenuItem>
        <DropdownMenuItem disabled={!canClose} onSelect={() => onClose("RESULT_REJECTED")}>
          Reject
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem disabled={!canManage} onSelect={() => void onStatusChange("STATE_DRAFT")}>
          Back to draft
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
