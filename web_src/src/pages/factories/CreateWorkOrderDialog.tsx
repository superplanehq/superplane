import type { FactoriesFactoryLine } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { cn } from "@/lib/utils";
import { ChevronRight, Factory as FactoryIcon, Maximize2, Minimize2, Play, XIcon } from "lucide-react";
import { useState, type ReactNode } from "react";

import { useFactoriesLayout } from "./layout/factoriesLayoutContext";
import { CreateWorkOrderPropertyPills } from "./CreateWorkOrderPropertyPills";
import { firstFactoryLineName } from "./lib/factoryPagePaths";
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
          isExpanded={isExpanded}
          onToggleExpanded={() => setIsExpanded((current) => !current)}
        >
          <DialogTitle className="text-[13px] font-medium text-foreground">New task</DialogTitle>
          <DialogDescription className="sr-only">Create a task for this workspace.</DialogDescription>
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
            placeholder="Task title"
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
          isSaving={composer.isSaving}
          canDispatch={canDispatch}
          canSaveDraft={composer.canSaveDraft}
          isSavingDraft={composer.isSavingDraft}
          isSendingToLine={composer.isSendingToLine}
          onAssigneeChange={composer.setAssigneeIds}
          onSaveDraft={() => void composer.handleSaveDraft()}
          onStart={() => {
            const lineName = firstFactoryLineName({ lines });
            if (lineName) {
              void composer.handleSendToLine(lineName);
            }
          }}
        />
      </DialogContent>
    </Dialog>
  );
}

function CreateWorkOrderDialogHeader({
  workspaceName,
  isExpanded,
  onToggleExpanded,
  children,
}: {
  workspaceName: string;
  isExpanded: boolean;
  onToggleExpanded: () => void;
  children: ReactNode;
}) {
  return (
    <div
      className="flex items-center justify-between gap-3 border-b border-border px-4 py-2.5"
      data-testid="work-order-create-header"
    >
      <div className="flex min-w-0 items-center gap-2 text-[13px] text-muted-foreground">
        <span className="flex size-5 shrink-0 items-center justify-center rounded-md bg-muted">
          <FactoryIcon className="size-3" aria-hidden />
        </span>
        <span className="truncate text-foreground">{workspaceName}</span>
        <ChevronRight className="size-3.5 shrink-0" aria-hidden />
        {children}
      </div>
      <div className="flex shrink-0 items-center gap-1">
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
  isSaving,
  canDispatch,
  canSaveDraft,
  isSavingDraft,
  isSendingToLine,
  onAssigneeChange,
  onSaveDraft,
  onStart,
}: {
  organizationId: string;
  assigneeIds: string[];
  lines: FactoriesFactoryLine[];
  isSaving: boolean;
  canDispatch: boolean;
  canSaveDraft: boolean;
  isSavingDraft: boolean;
  isSendingToLine: boolean;
  onAssigneeChange: (ids: string[]) => void;
  onSaveDraft: () => void;
  onStart: () => void;
}) {
  return (
    <div className="relative z-10 flex items-center justify-between gap-3 border-t border-border px-4 py-3">
      <CreateWorkOrderPropertyPills
        organizationId={organizationId}
        assigneeIds={assigneeIds}
        isSaving={isSaving}
        onAssigneeChange={onAssigneeChange}
      />

      <div className="flex shrink-0 items-center gap-2">
        <LoadingButton
          type="button"
          variant="outline"
          disabled={!canSaveDraft}
          loading={isSavingDraft}
          loadingText="Saving..."
          onClick={onSaveDraft}
          className="h-8 rounded-full px-4"
          data-testid="work-order-create-draft-button"
        >
          Save as draft
        </LoadingButton>

        <StartWorkOrderButton
          lines={lines}
          canDispatch={canDispatch}
          canSaveDraft={canSaveDraft}
          isSendingToLine={isSendingToLine}
          onStart={onStart}
        />
      </div>
    </div>
  );
}

function StartWorkOrderButton({
  lines,
  canDispatch,
  canSaveDraft,
  isSendingToLine,
  onStart,
}: {
  lines: FactoriesFactoryLine[];
  canDispatch: boolean;
  canSaveDraft: boolean;
  isSendingToLine: boolean;
  onStart: () => void;
}) {
  const hasLines = Boolean(firstFactoryLineName({ lines }));
  const isDisabled = !canDispatch || !canSaveDraft || !hasLines;
  const tooltipMessage = !canDispatch
    ? "You do not have permission to start tasks."
    : "This workspace has no line to start this task on.";

  return (
    <PermissionTooltip allowed={canDispatch && hasLines} message={tooltipMessage}>
      <LoadingButton
        type="button"
        disabled={isDisabled}
        loading={isSendingToLine}
        loadingText="Starting..."
        onClick={onStart}
        className="h-8 shrink-0 rounded-full px-4"
        data-testid="work-order-create-start"
      >
        <Play className="size-3.5" aria-hidden />
        Start
      </LoadingButton>
    </PermissionTooltip>
  );
}
