import { Copy, Trash2 } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import { CopyLinkButton } from "../../CopyLinkButton";

const HEADER_ICON_BUTTON =
  "flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10";

export function PopupHeaderActions({
  copyUrl,
  onArchive,
  archiveBusy = false,
  taskActions,
  canDuplicate = false,
  onDuplicate,
  duplicateBusy = false,
}: {
  copyUrl?: string;
  onArchive?: () => void | Promise<void>;
  archiveBusy?: boolean;
  taskActions?: ReactNode;
  canDuplicate?: boolean;
  onDuplicate?: () => void;
  duplicateBusy?: boolean;
}) {
  return (
    <>
      {taskActions}
      {canDuplicate && onDuplicate ? <PopupDuplicateButton onDuplicate={onDuplicate} busy={duplicateBusy} /> : null}
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

function PopupDuplicateButton({ onDuplicate, busy }: { onDuplicate: () => void; busy: boolean }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={onDuplicate}
            disabled={busy}
            className={HEADER_ICON_BUTTON}
            aria-label="Duplicate"
            data-testid="popup-work-order-duplicate-button"
          >
            <Copy className="h-4 w-4" aria-hidden />
          </Button>
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom">Duplicate</TooltipContent>
    </Tooltip>
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
