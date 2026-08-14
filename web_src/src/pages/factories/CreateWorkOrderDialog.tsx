import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { cn } from "@/lib/utils";
import { ChevronRight, Factory as FactoryIcon, Maximize2, Minimize2, XIcon } from "lucide-react";
import { useState, type ReactNode } from "react";

import { useFactoriesLayout } from "./layout/factoriesLayoutContext";
import { CreateWorkOrderPropertyPills } from "./CreateWorkOrderPropertyPills";
import { WorkOrderDescriptionEditor } from "./WorkOrderDescriptionEditor";
import { useCreateWorkOrderComposer } from "./useCreateWorkOrderComposer";

interface CreateWorkOrderDialogProps {
  open: boolean;
  onClose: () => void;
  onCreated: (orderNumber: string) => void;
}

export function CreateWorkOrderDialog({ open, onClose, onCreated }: CreateWorkOrderDialogProps) {
  if (!open) {
    return null;
  }

  return <CreateWorkOrderDialogSession onClose={onClose} onCreated={onCreated} />;
}

function CreateWorkOrderDialogSession({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (orderNumber: string) => void;
}) {
  const { organizationId, factoryId, factory } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const composer = useCreateWorkOrderComposer({ organizationId, factoryId, onClose, onCreated });
  const lines = factory?.lines ?? [];
  const [isExpanded, setIsExpanded] = useState(false);
  const canDispatch = canAct("work_orders", "update");

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      handleClose();
    }
  };

  const handleClose = () => {
    if (composer.isSaving) {
      return;
    }
    onClose();
  };

  return (
    <Dialog open onOpenChange={handleOpenChange}>
      <DialogContent
        showCloseButton={false}
        size="large"
        className={cn(
          "flex flex-col gap-0 overflow-hidden p-0 sm:rounded-xl",
          isExpanded ? "h-[90vh] w-[90vw] max-w-none" : "h-[min(72vh,560px)] w-[calc(100%-2rem)] max-w-2xl",
        )}
        data-testid="create-work-order-dialog"
      >
        <CreateWorkOrderDialogHeader
          workspaceName={factory?.name ?? "Workspace"}
          canSaveDraft={composer.canSaveDraft}
          isSavingDraft={composer.isSavingDraft}
          isExpanded={isExpanded}
          onSaveDraft={() => void composer.handleSaveDraft()}
          onToggleExpanded={() => setIsExpanded((current) => !current)}
        >
          <DialogTitle className="text-[13px] font-medium text-foreground">New work order</DialogTitle>
          <DialogDescription className="sr-only">Create a work order for this workspace.</DialogDescription>
        </CreateWorkOrderDialogHeader>

        <div className="flex min-h-0 flex-1 flex-col px-5 py-4">
          <Label htmlFor="work-order-title-input" className="sr-only">
            Title
          </Label>
          <Input
            id="work-order-title-input"
            data-testid="work-order-title-input"
            value={composer.title}
            onChange={(event) => composer.updateTitle(event.target.value)}
            placeholder="Work order title"
            maxLength={composer.maxTitleLength}
            autoFocus
            className="h-auto border-0 bg-transparent p-0 text-[22px] font-semibold tracking-[-0.02em] shadow-none placeholder:font-semibold placeholder:text-muted-foreground/70 focus-visible:ring-0"
          />
          {composer.titleError ? <p className="mt-1 text-[12px] text-destructive">{composer.titleError}</p> : null}

          <div className="mt-3 min-h-0 flex-1 overflow-y-auto">
            <Label htmlFor="work-order-description-input" className="sr-only">
              Description
            </Label>
            <WorkOrderDescriptionEditor
              value={composer.description}
              maxLength={composer.maxDescriptionLength}
              disabled={composer.isSaving}
              onChange={composer.updateDescription}
            />
          </div>
        </div>

        <CreateWorkOrderDialogFooter
          organizationId={organizationId}
          assigneeIds={composer.assigneeIds}
          lines={lines}
          selectedLineName={composer.selectedLineName}
          isSaving={composer.isSaving}
          canDispatch={canDispatch}
          canSendToLine={composer.canSendToLine}
          isSendingToLine={composer.isSendingToLine}
          onAssigneeChange={composer.setAssigneeIds}
          onLineSelect={composer.setSelectedLineName}
          onSendToLine={() => void composer.handleSendToLine()}
        />
      </DialogContent>
    </Dialog>
  );
}

function CreateWorkOrderDialogHeader({
  workspaceName,
  canSaveDraft,
  isSavingDraft,
  isExpanded,
  onSaveDraft,
  onToggleExpanded,
  children,
}: {
  workspaceName: string;
  canSaveDraft: boolean;
  isSavingDraft: boolean;
  isExpanded: boolean;
  onSaveDraft: () => void;
  onToggleExpanded: () => void;
  children: ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-2.5">
      <div className="flex min-w-0 items-center gap-2 text-[13px] text-muted-foreground">
        <span className="flex size-5 shrink-0 items-center justify-center rounded-md bg-muted">
          <FactoryIcon className="size-3" aria-hidden />
        </span>
        <span className="truncate text-foreground">{workspaceName}</span>
        <ChevronRight className="size-3.5 shrink-0" aria-hidden />
        {children}
      </div>
      <div className="flex shrink-0 items-center gap-1">
        <LoadingButton
          type="button"
          variant="outline"
          size="sm"
          disabled={!canSaveDraft}
          loading={isSavingDraft}
          loadingText="Saving..."
          onClick={onSaveDraft}
          className="h-7 rounded-full px-3 text-[12px] font-medium"
          data-testid="work-order-create-draft-button"
        >
          Save as draft
        </LoadingButton>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={isExpanded ? "Exit full screen" : "Open full screen"}
          onClick={onToggleExpanded}
          className="size-6 text-muted-foreground"
          data-testid="work-order-create-fullscreen-button"
        >
          {isExpanded ? <Minimize2 className="size-3.5" aria-hidden /> : <Maximize2 className="size-3.5" aria-hidden />}
        </Button>
        <DialogClose
          className="flex size-6 cursor-pointer items-center justify-center rounded-full text-muted-foreground hover:bg-slate-950/5 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 dark:hover:bg-white/10"
          data-testid="work-order-create-close-button"
        >
          <XIcon className="size-4" />
          <span className="sr-only">Close</span>
        </DialogClose>
      </div>
    </div>
  );
}

function CreateWorkOrderDialogFooter({
  organizationId,
  assigneeIds,
  lines,
  selectedLineName,
  isSaving,
  canDispatch,
  canSendToLine,
  isSendingToLine,
  onAssigneeChange,
  onLineSelect,
  onSendToLine,
}: {
  organizationId: string;
  assigneeIds: string[];
  lines: FactoriesFactoryLine[];
  selectedLineName: string;
  isSaving: boolean;
  canDispatch: boolean;
  canSendToLine: boolean;
  isSendingToLine: boolean;
  onAssigneeChange: (ids: string[]) => void;
  onLineSelect: (lineName: string) => void;
  onSendToLine: () => void;
}) {
  return (
    <div className="relative z-10 flex items-center justify-between gap-3 border-t border-border px-4 py-3">
      <CreateWorkOrderPropertyPills
        organizationId={organizationId}
        assigneeIds={assigneeIds}
        lines={lines}
        selectedLineName={selectedLineName}
        isSaving={isSaving}
        canDispatch={canDispatch}
        onAssigneeChange={onAssigneeChange}
        onLineSelect={onLineSelect}
      />

      <PermissionTooltip allowed={canDispatch} message="You don't have permission to dispatch work orders.">
        <LoadingButton
          type="button"
          disabled={!canDispatch || !canSendToLine}
          loading={isSendingToLine}
          loadingText="Sending..."
          onClick={onSendToLine}
          className="h-8 shrink-0 rounded-full px-4"
          data-testid="work-order-create-send-to-line"
        >
          Send to line
        </LoadingButton>
      </PermissionTooltip>
    </div>
  );
}
