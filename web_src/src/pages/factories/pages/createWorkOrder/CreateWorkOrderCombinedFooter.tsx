import { PermissionTooltip } from "@/components/PermissionGate";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { LoadingButton } from "@/components/ui/loading-button";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { Popover, PopoverTrigger } from "@/ui/popover";
import { User } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";

import {
  AssigneePickerPanel,
  AssigneePillBody,
  LinePickerPanel,
  PropertyPill,
} from "../../CreateWorkOrderPropertyPills";
import type { CreateWorkOrderActionSlotProps } from "../../createWorkOrderActionSlot";

/**
 * Storybook-only redesign of the New work order footer (issue #6791).
 * Owner sits on the left. Save as draft and Send to line sit together on
 * the right, with line choice happening at send time instead of a separate
 * Line control. Wired in via `CreateWorkOrderActionSlotContext` so the live
 * dialog keeps today's header/footer split by default.
 */
export function CreateWorkOrderCombinedFooter({
  organizationId,
  assigneeIds,
  lines,
  isSaving,
  canDispatch,
  canSaveDraft,
  isSavingDraft,
  isSendingToLine,
  onAssigneeChange,
  onSaveDraft,
  onSendToLine,
}: CreateWorkOrderActionSlotProps) {
  const { resolveUser } = useOrgUserLookup(organizationId);
  const [portalRoot, setPortalRoot] = useState<HTMLDivElement | null>(null);
  const [isOwnerOpen, setIsOwnerOpen] = useState(false);
  const [draftAssigneeIds, setDraftAssigneeIds] = useState(assigneeIds);
  const [isLinePickerOpen, setIsLinePickerOpen] = useState(false);
  const hasLines = lines.length > 0;
  const canSendToLine = canDispatch && canSaveDraft && hasLines;

  useEffect(() => {
    if (isOwnerOpen) {
      setDraftAssigneeIds(assigneeIds);
    }
  }, [assigneeIds, isOwnerOpen]);

  const handleOwnerOpenChange = (nextOpen: boolean) => {
    if (isSaving) {
      return;
    }
    setIsOwnerOpen(nextOpen);
  };

  const handleSaveOwner = () => {
    onAssigneeChange(draftAssigneeIds);
    setIsOwnerOpen(false);
  };

  const handleLinePickerOpenChange = (nextOpen: boolean) => {
    if (!canSendToLine) {
      return;
    }
    setIsLinePickerOpen(nextOpen);
  };

  const handlePickLine = (lineName: string) => {
    setIsLinePickerOpen(false);
    onSendToLine(lineName);
  };

  return (
    <div
      ref={setPortalRoot}
      className="relative z-10 flex items-center justify-between gap-3 border-t border-border px-4 py-3"
    >
      <Popover modal={false} open={isOwnerOpen} onOpenChange={handleOwnerOpenChange}>
        <PopoverTrigger asChild>
          <PropertyPill disabled={isSaving} testId="work-order-assignees-button">
            {assigneeIds.length === 0 ? (
              <>
                <User className="size-3.5" aria-hidden />
                Owner
              </>
            ) : (
              <AssigneePillBody assigneeIds={assigneeIds} resolveUser={resolveUser} />
            )}
          </PropertyPill>
        </PopoverTrigger>
        <AssigneePickerPanel
          organizationId={organizationId}
          selectedIds={draftAssigneeIds}
          isSaving={isSaving}
          portalRoot={portalRoot}
          onChange={setDraftAssigneeIds}
          onSave={handleSaveOwner}
        />
      </Popover>

      <div className="ml-auto flex shrink-0 items-center gap-2">
        <LoadingButton
          type="button"
          variant="outline"
          disabled={!canSaveDraft}
          loading={isSavingDraft}
          loadingText="Saving..."
          onClick={onSaveDraft}
          className="h-8 rounded-full px-4 text-[12px] font-medium"
          data-testid="work-order-create-draft-button"
        >
          Save as draft
        </LoadingButton>

        <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
          <NoLinesHint hasLines={hasLines}>
            <Popover modal={false} open={isLinePickerOpen} onOpenChange={handleLinePickerOpenChange}>
              <PopoverTrigger asChild>
                <LoadingButton
                  type="button"
                  disabled={!canSendToLine}
                  loading={isSendingToLine}
                  loadingText="Sending..."
                  className="h-8 shrink-0 rounded-full px-4"
                  data-testid="work-order-create-send-to-line"
                >
                  Send to line
                </LoadingButton>
              </PopoverTrigger>
              <LinePickerPanel
                lines={lines}
                selectedLineName=""
                isSaving={isSaving}
                portalRoot={portalRoot}
                align="end"
                onSelect={handlePickLine}
              />
            </Popover>
          </NoLinesHint>
        </PermissionTooltip>
      </div>
    </div>
  );
}

function NoLinesHint({ hasLines, children }: { hasLines: boolean; children: ReactNode }) {
  if (hasLines) {
    return <>{children}</>;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="inline-flex">{children}</div>
      </TooltipTrigger>
      <TooltipContent side="top">This workspace has no lines to send this work order to.</TooltipContent>
    </Tooltip>
  );
}
