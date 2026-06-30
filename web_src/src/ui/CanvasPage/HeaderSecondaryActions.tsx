import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { X } from "lucide-react";
import { DiffSummaryHoverCard } from "./components/DiffSummaryHoverCard";
import type { HeaderProps } from "./Header";
import { isCanvasTabHeaderMode } from "./canvasTabHeaderMode";

export function SecondaryHeaderActions({
  mode,
  isEditing = false,
  hasUnpublishedDraftChanges,
  hasUnpublishedConsoleDraftChanges,
  hasUncommittedCanvasDraftChanges,
  hasUncommittedConsoleDraftChanges,
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
  hasStagingChanges,
  onCommitStaging,
  commitStagingPending,
  resetStagingPending,
  onResetStaging,
}: HeaderProps) {
  const onCanvasTab = isCanvasTabHeaderMode(mode);
  const onConsoleTab = mode === "console";

  return (
    <div className="relative z-10 ml-auto flex shrink-0 items-center gap-1.5">
      <FilesHeaderActionsSlot isEditing={isEditing} mode={mode} slotId={filesHeaderActionsSlotId} />

      {isEditing ? (
        <>
          {onCanvasTab &&
          (hasUnpublishedDraftChanges || hasUncommittedCanvasDraftChanges) &&
          draftVisualDiff?.diffCounts ? (
            <DiffSummaryHoverCard
              diffCounts={draftVisualDiff.diffCounts}
              visualDiffEnabled={visualDiffEnabled}
              onToggleVisualDiff={onToggleVisualDiff}
              diffToggles={draftVisualDiff.diffToggles}
              onShowDiff={onShowDiff}
            />
          ) : null}
          {onConsoleTab &&
          (hasUnpublishedConsoleDraftChanges || hasUncommittedConsoleDraftChanges) &&
          draftConsoleDiff?.diffCounts ? (
            <ConsoleDiffSummaryHoverCard
              draftConsoleDiff={draftConsoleDiff}
              visualDiffEnabled={visualDiffEnabled}
              onToggleVisualDiff={onToggleVisualDiff}
              onShowConsoleDiff={onShowConsoleDiff}
            />
          ) : null}
          <EditModePublishDiscardActions
            onDiscardVersion={onDiscardVersion}
            discardVersionDisabled={discardVersionDisabled}
            discardVersionDisabledTooltip={discardVersionDisabledTooltip}
            onPublishVersion={onPublishVersion}
            publishVersionLabel={publishVersionLabel}
            publishVersionDisabled={publishVersionDisabled}
            publishVersionDisabledTooltip={publishVersionDisabledTooltip}
            hasStagingChanges={hasStagingChanges}
            onCommitStaging={onCommitStaging}
            commitStagingPending={commitStagingPending}
            resetStagingPending={resetStagingPending}
            onResetStaging={onResetStaging}
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
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  onPublishVersion,
  publishVersionLabel,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  hasStagingChanges,
  onCommitStaging,
  commitStagingPending,
  resetStagingPending,
  onResetStaging,
}: Pick<
  HeaderProps,
  | "onDiscardVersion"
  | "discardVersionDisabled"
  | "discardVersionDisabledTooltip"
  | "onPublishVersion"
  | "publishVersionLabel"
  | "publishVersionDisabled"
  | "publishVersionDisabledTooltip"
  | "hasStagingChanges"
  | "onCommitStaging"
  | "commitStagingPending"
  | "resetStagingPending"
  | "onResetStaging"
>) {
  const stagingActionPending = !!commitStagingPending || !!resetStagingPending;

  // Keep showing the staging controls while a staging action is in flight even
  // after `hasStagingChanges` optimistically flips false, so the header never
  // flashes enabled Reset/Commit controls or premature Discard/Publish actions.
  const showStagingActions = !!onCommitStaging && (!!hasStagingChanges || stagingActionPending);

  // Staging and committed states are mutually exclusive: while there are staged
  // edits the user can only Reset/Commit them; once everything is committed they
  // can Discard the draft or Publish it.
  if (showStagingActions) {
    return (
      <div className="flex items-center gap-1.5">
        {onResetStaging ? (
          <ResetStagingButton onReset={() => onResetStaging()} disabled={stagingActionPending} />
        ) : null}
        <CommitStagingButton onCommit={() => onCommitStaging?.()} disabled={stagingActionPending} />
      </div>
    );
  }

  return (
    <div className="flex items-center gap-1.5">
      {onDiscardVersion ? (
        <DiscardDraftButton
          onDiscard={() => onDiscardVersion()}
          disabled={!!discardVersionDisabled}
          disabledTooltip={discardVersionDisabledTooltip}
        />
      ) : null}
      {onPublishVersion ? (
        <PublishVersionButton
          onPublish={() => onPublishVersion()}
          label={publishVersionLabel || "Publish"}
          disabled={!!publishVersionDisabled}
          publishVersionDisabled={!!publishVersionDisabled}
          publishVersionDisabledTooltip={publishVersionDisabledTooltip}
        />
      ) : null}
    </div>
  );
}

function ResetStagingButton({ onReset, disabled }: { onReset: () => void; disabled: boolean }) {
  return (
    <Tooltip delayDuration={2000}>
      <TooltipTrigger asChild>
        <UIButton
          type="button"
          variant="outline"
          size="sm"
          onClick={onReset}
          disabled={disabled}
          data-testid="canvas-reset-staging-button"
        >
          Reset
        </UIButton>
      </TooltipTrigger>
      <TooltipContent side="top">Reset to last commit</TooltipContent>
    </Tooltip>
  );
}

// The label stays fixed (no "Committing…") so the button keeps a constant
// width while a commit is in flight; the button is disabled instead to signal
// the in-flight commit.
function CommitStagingButton({ onCommit, disabled }: { onCommit: () => void; disabled: boolean }) {
  return (
    <UIButton
      type="button"
      variant="default"
      size="sm"
      className={cn("bg-orange-500 text-white hover:bg-orange-600 hover:opacity-95 focus-visible:ring-orange-500/40")}
      onClick={onCommit}
      disabled={disabled}
      data-testid="canvas-commit-staging-button"
    >
      Commit
    </UIButton>
  );
}

export function LiveModeTopHeaderActions({
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
}: Pick<HeaderProps, "onEnterEditMode" | "enterEditModeDisabled" | "enterEditModeDisabledTooltip">) {
  if (!onEnterEditMode) {
    return null;
  }

  return (
    <EnterEditButton
      onClick={onEnterEditMode}
      label="Edit"
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
