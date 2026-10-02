import { Copy, Trash2 } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PermissionTooltip } from "@/components/PermissionGate";

import { CopyLinkButton } from "../../CopyLinkButton";

const HEADER_ICON_BUTTON =
  "flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10";

export function PopupHeaderActions({
  copyUrl,
  onArchive,
  archiveBusy = false,
  onDuplicate,
  canDuplicate = false,
  taskActions,
}: {
  copyUrl?: string;
  onArchive?: () => void | Promise<void>;
  archiveBusy?: boolean;
  onDuplicate?: () => void | Promise<void>;
  canDuplicate?: boolean;
  taskActions?: ReactNode;
}) {
  return (
    <>
      {taskActions}
      {onArchive ? <PopupArchiveButton onArchive={onArchive} busy={archiveBusy} /> : null}
      {onDuplicate ? (
        <PopupDuplicateButton onDuplicate={onDuplicate} canDuplicate={canDuplicate} />
      ) : null}
      <CopyLinkButton
        url={copyUrl}
        className={HEADER_ICON_BUTTON}
        iconClassName="h-4 w-4"
        testId="popup-work-order-copy-link-button"
      />
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

function PopupDuplicateButton({
  onDuplicate,
  canDuplicate,
}: {
  onDuplicate: () => void | Promise<void>;
  canDuplicate: boolean;
}) {
  if (!canDuplicate) {
    return (
      <PermissionTooltip label="Duplicate" permission="work_orders:create">
        <span>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            disabled
            className={HEADER_ICON_BUTTON}
            aria-label="Duplicate"
            data-testid="popup-work-order-duplicate-button"
          >
            <Copy className="h-4 w-4" aria-hidden />
          </Button>
        </span>
      </PermissionTooltip>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={() => void onDuplicate()}
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
