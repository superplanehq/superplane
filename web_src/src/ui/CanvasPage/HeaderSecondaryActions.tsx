import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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
  hasStagingChanges,
  stagingStale,
  onCommitStaging,
  commitStagingPending,
  resetStagingPending,
  onResetStaging,
  onDiscardStaleStaging,
  discardStaleStagingPending,
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
          <EditModeStagingActions
            stagingStale={stagingStale}
            hasStagingChanges={hasStagingChanges}
            onCommitStaging={onCommitStaging}
            commitStagingPending={commitStagingPending}
            resetStagingPending={resetStagingPending}
            onResetStaging={onResetStaging}
            onDiscardStaleStaging={onDiscardStaleStaging}
            discardStaleStagingPending={discardStaleStagingPending}
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

function EditModeStagingActions({
  stagingStale,
  hasStagingChanges,
  onCommitStaging,
  commitStagingPending,
  resetStagingPending,
  onResetStaging,
  onDiscardStaleStaging,
  discardStaleStagingPending,
}: Pick<
  HeaderProps,
  | "stagingStale"
  | "hasStagingChanges"
  | "onCommitStaging"
  | "commitStagingPending"
  | "resetStagingPending"
  | "onResetStaging"
  | "onDiscardStaleStaging"
  | "discardStaleStagingPending"
>) {
  const stagingActionPending = !!commitStagingPending || !!resetStagingPending || !!discardStaleStagingPending;

  if (stagingStale) {
    return (
      <div className="flex max-w-md items-center gap-2">
        <p className="text-xs text-amber-800">
          Main branch has been updated since you last edited. Discard your changes and start again.
        </p>
        <DiscardStaleStagingButton
          onDiscard={() => onDiscardStaleStaging?.()}
          disabled={!!discardStaleStagingPending}
        />
      </div>
    );
  }

  const showStagingActions =
    (!!onCommitStaging || !!onResetStaging) && (!!onCommitStaging || !!hasStagingChanges || stagingActionPending);
  if (!showStagingActions) {
    return null;
  }

  const stagingActionsDisabled = stagingActionPending || !hasStagingChanges;

  return (
    <div className="flex items-center gap-1.5">
      {onResetStaging ? (
        <ResetStagingButton onReset={() => onResetStaging()} disabled={stagingActionsDisabled} />
      ) : null}
      {onCommitStaging ? (
        <CommitStagingButton onCommit={() => onCommitStaging()} disabled={stagingActionsDisabled} />
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

function CommitStagingButton({ onCommit, disabled }: { onCommit: () => void; disabled: boolean }) {
  return (
    <UIButton
      type="button"
      variant="default"
      size="sm"
      onClick={onCommit}
      disabled={disabled}
      data-testid="canvas-commit-staging-button"
    >
      Commit
    </UIButton>
  );
}

function DiscardStaleStagingButton({ onDiscard, disabled }: { onDiscard: () => void; disabled: boolean }) {
  return (
    <UIButton type="button" variant="outline" size="sm" onClick={onDiscard} disabled={disabled}>
      Discard
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
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      data-testid="canvas-exit-edit-button"
      aria-label="Finish editing"
      className="group flex items-center gap-2 disabled:cursor-not-allowed disabled:opacity-50"
    >
      <span className="text-[13px] font-medium text-slate-600 dark:text-gray-400">Finish Editing</span>
      <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-slate-950 transition-colors group-hover:bg-slate-950/5 dark:text-gray-100 dark:group-hover:bg-gray-800/50">
        <X className="h-4 w-4" aria-hidden />
      </span>
    </button>
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
