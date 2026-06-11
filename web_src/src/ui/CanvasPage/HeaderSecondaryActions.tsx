import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";
import { Button } from "../button";
import { DiffSummaryHoverCard } from "./components/DiffSummaryHoverCard";
import { EnterEditDraftDropdown } from "./components/EnterEditDraftDropdown";
import { StartEditingDropdown } from "./components/StartEditingDropdown";
import type { HeaderProps } from "./Header";

export function SecondaryHeaderActions({
  mode,
  isEditing = false,
  onSave,
  saveButtonHidden,
  saveDisabled,
  saveDisabledTooltip,
  saveIsPrimary,
  hasUnpublishedDraftChanges,
  hasUnpublishedConsoleDraftChanges,
  onShowDiff,
  onShowConsoleDiff,
  visualDiffEnabled,
  draftVisualDiff,
  draftConsoleDiff,
  onToggleVisualDiff,
  filesHeaderActionsSlotId,
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  onPublishVersion,
  publishVersionLabel,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
}: HeaderProps) {
  const onCanvasTab = mode === "version-live" || mode === "version-edit";
  const onConsoleTab = mode === "console";

  return (
    <div className="relative z-10 ml-auto flex shrink-0 items-center gap-1.5">
      {mode === "default" && onSave && !saveButtonHidden ? (
        <SaveButton
          onSave={onSave}
          saveDisabled={saveDisabled}
          saveDisabledTooltip={saveDisabledTooltip}
          saveIsPrimary={saveIsPrimary}
        />
      ) : null}

      <FilesHeaderActionsSlot isEditing={isEditing} mode={mode} slotId={filesHeaderActionsSlotId} />

      {isEditing ? (
        <>
          {onCanvasTab && hasUnpublishedDraftChanges && draftVisualDiff?.diffCounts ? (
            <DiffSummaryHoverCard
              diffCounts={draftVisualDiff.diffCounts}
              visualDiffEnabled={visualDiffEnabled}
              onToggleVisualDiff={onToggleVisualDiff}
              diffToggles={draftVisualDiff.diffToggles}
              onShowDiff={onShowDiff}
            />
          ) : null}
          {onConsoleTab && hasUnpublishedConsoleDraftChanges && draftConsoleDiff?.diffCounts ? (
            <ConsoleDiffSummaryHoverCard
              draftConsoleDiff={draftConsoleDiff}
              visualDiffEnabled={visualDiffEnabled}
              onToggleVisualDiff={onToggleVisualDiff}
              onShowConsoleDiff={onShowConsoleDiff}
            />
          ) : null}
          <EditModePublishDiscardActions
            hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
            onDiscardVersion={onDiscardVersion}
            discardVersionDisabled={discardVersionDisabled}
            discardVersionDisabledTooltip={discardVersionDisabledTooltip}
            onPublishVersion={onPublishVersion}
            publishVersionLabel={publishVersionLabel}
            publishVersionDisabled={publishVersionDisabled}
            publishVersionDisabledTooltip={publishVersionDisabledTooltip}
          />
        </>
      ) : null}
    </div>
  );
}

function ConsoleDiffSummaryHoverCard({
  draftConsoleDiff,
  visualDiffEnabled,
  onToggleVisualDiff,
  onShowConsoleDiff,
}: {
  draftConsoleDiff: NonNullable<HeaderProps["draftConsoleDiff"]>;
} & Pick<HeaderProps, "visualDiffEnabled" | "onToggleVisualDiff" | "onShowConsoleDiff">) {
  return (
    <DiffSummaryHoverCard
      diffCounts={draftConsoleDiff.diffCounts}
      visualDiffEnabled={visualDiffEnabled}
      onToggleVisualDiff={onToggleVisualDiff}
      onShowDiff={onShowConsoleDiff}
    />
  );
}

function FilesHeaderActionsSlot({
  isEditing,
  mode,
  slotId,
}: {
  isEditing: boolean;
  mode: HeaderProps["mode"];
  slotId?: string;
}) {
  if (!isEditing || mode !== "files" || !slotId) {
    return null;
  }

  return <div id={slotId} className="flex shrink-0 items-center gap-2" />;
}

function EditModePublishDiscardActions({
  hasUnpublishedDraftChanges,
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  onPublishVersion,
  publishVersionLabel,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
}: Pick<
  HeaderProps,
  | "hasUnpublishedDraftChanges"
  | "onDiscardVersion"
  | "discardVersionDisabled"
  | "discardVersionDisabledTooltip"
  | "onPublishVersion"
  | "publishVersionLabel"
  | "publishVersionDisabled"
  | "publishVersionDisabledTooltip"
>) {
  return (
    <div className="flex items-center gap-1.5">
      {hasUnpublishedDraftChanges ? (
        <DiscardDraftButton
          onDiscard={() => onDiscardVersion?.()}
          disabled={discardVersionDisabled || !onDiscardVersion}
          disabledTooltip={discardVersionDisabledTooltip}
        />
      ) : null}
      <PublishVersionButton
        onPublish={() => onPublishVersion?.()}
        label={publishVersionLabel || "Publish"}
        disabled={publishVersionDisabled || !onPublishVersion}
        publishVersionDisabled={!!publishVersionDisabled}
        publishVersionDisabledTooltip={publishVersionDisabledTooltip}
      />
    </div>
  );
}

export function LiveModeTopHeaderActions({
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  hasUnpublishedDraftChanges,
  onDiscardDraftAndStartEdit,
  unpublishedDraftUpdatedAt,
  startEditingDrafts,
  startEditingDefaultDraft,
  startEditingMenuOpen,
  onStartEditingMenuOpenChange,
  onContinueDraftBranch,
  onCreateDraftBranch,
  createDraftBranchPending,
}: Pick<
  HeaderProps,
  | "onEnterEditMode"
  | "enterEditModeDisabled"
  | "enterEditModeDisabledTooltip"
  | "hasUnpublishedDraftChanges"
  | "onDiscardDraftAndStartEdit"
  | "unpublishedDraftUpdatedAt"
  | "startEditingDrafts"
  | "startEditingDefaultDraft"
  | "startEditingMenuOpen"
  | "onStartEditingMenuOpenChange"
  | "onContinueDraftBranch"
  | "onCreateDraftBranch"
  | "createDraftBranchPending"
>) {
  if (startEditingDrafts !== undefined && onContinueDraftBranch && onCreateDraftBranch) {
    return (
      <StartEditingDropdown
        open={startEditingMenuOpen}
        onOpenChange={onStartEditingMenuOpenChange}
        drafts={startEditingDrafts}
        defaultDraft={startEditingDefaultDraft ?? null}
        disabled={!!enterEditModeDisabled}
        isSubmitting={createDraftBranchPending}
        onContinueDraft={onContinueDraftBranch}
        onCreateDraft={onCreateDraftBranch}
      />
    );
  }

  if (!onEnterEditMode) {
    return null;
  }

  const showDraftDropdown = !!hasUnpublishedDraftChanges && !!onDiscardDraftAndStartEdit && !enterEditModeDisabled;

  if (showDraftDropdown && onDiscardDraftAndStartEdit) {
    return (
      <EnterEditDraftDropdown
        onContinueEditing={onEnterEditMode}
        onDiscardAndStartEdit={onDiscardDraftAndStartEdit}
        updatedAt={unpublishedDraftUpdatedAt}
      />
    );
  }

  return (
    <EnterEditButton
      onClick={onEnterEditMode}
      label={hasUnpublishedDraftChanges ? "Continue Editing" : "Edit"}
      disabled={!!enterEditModeDisabled}
      disabledTooltip={enterEditModeDisabledTooltip}
    />
  );
}

export function EditModeTopHeaderActions({
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
}: Pick<HeaderProps, "onExitEditMode" | "exitEditModeDisabled" | "exitEditModeDisabledTooltip">) {
  if (!onExitEditMode) {
    return null;
  }

  return (
    <ExitEditButton
      onClick={() => onExitEditMode()}
      disabled={!!exitEditModeDisabled}
      disabledTooltip={exitEditModeDisabledTooltip}
    />
  );
}

function EnterEditButton({
  onClick,
  label,
  disabled,
  disabledTooltip,
}: {
  onClick: () => void;
  label: string;
  disabled: boolean;
  disabledTooltip?: string;
}) {
  const button = (
    <UIButton
      type="button"
      variant="outline"
      size="sm"
      onClick={onClick}
      disabled={disabled}
      data-testid="canvas-edit-button"
    >
      {label}
    </UIButton>
  );

  if (disabled && disabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{button}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{disabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return button;
}

function ExitEditButton({
  onClick,
  disabled,
  disabledTooltip,
}: {
  onClick: () => void;
  disabled: boolean;
  disabledTooltip?: string;
}) {
  const button = (
    <UIButton
      type="button"
      variant="ghost"
      size="icon"
      onClick={onClick}
      disabled={disabled}
      data-testid="canvas-exit-edit-button"
      aria-label="Exit edit"
      className="-mr-0.5 size-8 shrink-0 p-0 text-slate-950 hover:bg-transparent hover:text-slate-900"
    >
      <X className="size-5 stroke-[2] text-slate-950 opacity-65" aria-hidden />
    </UIButton>
  );

  if (disabled && disabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{button}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{disabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return button;
}

function SaveButton({
  onSave,
  saveDisabled,
  saveDisabledTooltip,
  saveIsPrimary,
}: {
  onSave: () => void;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  saveIsPrimary?: boolean;
}) {
  if (saveDisabled && saveDisabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">
            <Button
              onClick={onSave}
              size="sm"
              variant={saveIsPrimary ? "default" : "outline"}
              data-testid="save-canvas-button"
              disabled={saveDisabled}
            >
              Save
            </Button>
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">{saveDisabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <Button
      onClick={onSave}
      size="sm"
      variant={saveIsPrimary ? "default" : "outline"}
      data-testid="save-canvas-button"
      disabled={saveDisabled}
    >
      Save
    </Button>
  );
}

function DiscardDraftButton({
  onDiscard,
  disabled,
  disabledTooltip,
}: {
  onDiscard: () => void;
  disabled: boolean;
  disabledTooltip?: string;
}) {
  if (disabled && disabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">
            <UIButton type="button" variant="outline" size="sm" onClick={onDiscard} disabled={disabled}>
              Discard
            </UIButton>
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">{disabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <UIButton type="button" variant="outline" size="sm" onClick={onDiscard} disabled={disabled}>
      Discard
    </UIButton>
  );
}

function publishVersionButtonClassName(): string {
  return "bg-blue-500 text-white hover:bg-blue-600 hover:opacity-95 focus-visible:ring-blue-500/40";
}

function PublishVersionButton({
  onPublish,
  label,
  disabled,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
}: {
  onPublish: () => void;
  label: string;
  disabled: boolean;
  publishVersionDisabled: boolean;
  publishVersionDisabledTooltip?: string;
}) {
  const button = (
    <UIButton
      type="button"
      variant="default"
      size="sm"
      className={cn(publishVersionButtonClassName())}
      onClick={onPublish}
      disabled={disabled}
      data-testid="canvas-publish-version-button"
    >
      {label}
    </UIButton>
  );

  if (publishVersionDisabled && publishVersionDisabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{button}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{publishVersionDisabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return button;
}
