import { Ellipsis, Trash2 } from "lucide-react";
import { useState, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

import { CopyLinkButton } from "../../CopyLinkButton";
import { ForkTaskDialog, type ForkTaskTarget } from "../../ForkTaskDialog";
import { FORK_TASK_COPY } from "../../lib/forkTask";

const HEADER_ICON_BUTTON =
  "flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10";

export function PopupHeaderActions({
  copyUrl,
  onArchive,
  archiveBusy = false,
  taskActions,
  fork,
}: {
  copyUrl?: string;
  onArchive?: () => void | Promise<void>;
  archiveBusy?: boolean;
  taskActions?: ReactNode;
  fork?: ForkTaskTarget;
}) {
  return (
    <>
      {taskActions}
      {fork ? <PopupForkMenu fork={fork} /> : null}
      {onArchive ? <PopupArchiveButton onArchive={onArchive} busy={archiveBusy} /> : null}
      <CopyLinkButton
        url={copyUrl}
        className={HEADER_ICON_BUTTON}
        iconClassName="h-4 w-4"
        testId="popup-work-order-copy-link-button"
      />
    </>
  );
}

function PopupForkMenu({ fork }: { fork: ForkTaskTarget }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className={HEADER_ICON_BUTTON}
            aria-label="Task actions"
            data-testid="popup-fork-task-menu"
          >
            <Ellipsis className="h-4 w-4" aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuItem onSelect={() => setOpen(true)} data-testid="fork-task-menu-item">
            {FORK_TASK_COPY.menu}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <ForkTaskDialog open={open} onOpenChange={setOpen} target={fork} />
    </>
  );
}

function PopupArchiveButton({ onArchive, busy }: { onArchive: () => void | Promise<void>; busy: boolean }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={() => void onArchive()}
            disabled={busy}
            className={HEADER_ICON_BUTTON}
            aria-label="Archive"
            data-testid="popup-work-order-archive-button"
          >
            <Trash2 className="h-4 w-4" aria-hidden />
          </Button>
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom">Archive</TooltipContent>
    </Tooltip>
  );
}
