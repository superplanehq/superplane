import { Copy, Trash2 } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PermissionTooltip } from "@/components/PermissionGate";
import { usePermissions } from "@/contexts/usePermissions";

import { CopyLinkButton } from "../../CopyLinkButton";

const HEADER_ICON_BUTTON =
  "flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10";

export function PopupHeaderActions({
  copyUrl,
  onArchive,
  archiveBusy = false,
  onDuplicate,
  duplicateBusy = false,
  taskActions,
}: {
  copyUrl?: string;
  onArchive?: () => void | Promise<void>;
  archiveBusy?: boolean;
  onDuplicate?: () => void | Promise<void>;
  duplicateBusy?: boolean;
  taskActions?: ReactNode;
}) {
  const { canAct } = usePermissions();
  const canCreateWorkOrder = canAct("work_orders", "create");

  return (
    <>
      {taskActions}
      {onArchive ? <PopupArchiveButton onArchive={onArchive} busy={archiveBusy} /> : null}
      {onDuplicate ? (
        <PopupDuplicateButton onDuplicate={onDuplicate} busy={duplicateBusy} canCreate={canCreateWorkOrder} />
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
  busy,
  canCreate,
}: {
  onDuplicate: () => void | Promise<void>;
  busy: boolean;
  canCreate: boolean;
}) {
  const permissionDeniedMessage = "You do not have permission to create tasks.";

  const button = (
    <Tooltip>
      <TooltipTrigger asChild>
        <span>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={() => void onDuplicate()}
            disabled={busy || !canCreate}
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

  return (
    <PermissionTooltip allowed={canCreate} message={permissionDeniedMessage}>
      {button}
    </PermissionTooltip>
  );
}
