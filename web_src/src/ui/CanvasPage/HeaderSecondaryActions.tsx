import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Plus } from "lucide-react";

import { Button } from "../button";
import { DiffSummaryHoverCard } from "./components/DiffSummaryHoverCard";
import { EnterEditDraftDropdown } from "./components/EnterEditDraftDropdown";
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
  onShowDiff,
  visualDiffEnabled,
  draftVisualDiff,
  onToggleVisualDiff,
  onDashboardAddPanel,
  filesHeaderActionsSlotId,
  onCanvasAddComponent,
}: HeaderProps) {
  const onCanvasTab = mode === "version-live" || mode === "version-edit";

  return (
    <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
      {mode === "default" && onSave && !saveButtonHidden ? (
        <SaveButton
          onSave={onSave}
          saveDisabled={saveDisabled}
          saveDisabledTooltip={saveDisabledTooltip}
          saveIsPrimary={saveIsPrimary}
        />
      ) : null}

      {isEditing && onCanvasTab ? (
        <EditModeCanvasSecondaryActions
          hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
          onShowDiff={onShowDiff}
          visualDiffEnabled={visualDiffEnabled}
          draftVisualDiff={draftVisualDiff}
          onToggleVisualDiff={onToggleVisualDiff}
          onCanvasAddComponent={onCanvasAddComponent}
        />
      ) : null}

      {isEditing && mode === "dashboard" ? (
        <EditModeDashboardSecondaryActions onDashboardAddPanel={onDashboardAddPanel} />
      ) : null}

      {mode === "files" && filesHeaderActionsSlotId ? (
        <div id={filesHeaderActionsSlotId} className="flex shrink-0 items-center gap-2" />
      ) : null}
    </div>
  );
}

function EditModeCanvasSecondaryActions({
  hasUnpublishedDraftChanges,
  onShowDiff,
  visualDiffEnabled,
  draftVisualDiff,
  onToggleVisualDiff,
  onCanvasAddComponent,
}: Pick<
  HeaderProps,
  | "hasUnpublishedDraftChanges"
  | "onShowDiff"
  | "visualDiffEnabled"
  | "draftVisualDiff"
  | "onToggleVisualDiff"
  | "onCanvasAddComponent"
>) {
  return (
    <>
      {hasUnpublishedDraftChanges && draftVisualDiff?.diffCounts ? (
        <DiffSummaryHoverCard
          diffCounts={draftVisualDiff.diffCounts}
          visualDiffEnabled={visualDiffEnabled}
          onToggleVisualDiff={onToggleVisualDiff}
          diffToggles={draftVisualDiff.diffToggles}
          onShowDiff={onShowDiff}
        />
      ) : null}

      {onCanvasAddComponent ? (
        <UIButton
          type="button"
          size="sm"
          variant="default"
          onClick={() => onCanvasAddComponent()}
          data-testid="canvas-add-component-button"
        >
          <Plus className="mr-1 h-3.5 w-3.5" />
          Add component
        </UIButton>
      ) : null}
    </>
  );
}

function EditModeDashboardSecondaryActions({ onDashboardAddPanel }: Pick<HeaderProps, "onDashboardAddPanel">) {
  return (
    <>
      {onDashboardAddPanel ? (
        <UIButton
          type="button"
          size="sm"
          variant="default"
          onClick={() => onDashboardAddPanel()}
          data-testid="dashboard-add-panel"
        >
          <Plus className="mr-1 h-3.5 w-3.5" />
          Add panel
        </UIButton>
      ) : null}
    </>
  );
}

export function LiveModeTopHeaderActions({
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  hasUnpublishedDraftChanges,
  onDiscardDraftAndStartEdit,
  unpublishedDraftUpdatedAt,
}: Pick<
  HeaderProps,
  | "onEnterEditMode"
  | "enterEditModeDisabled"
  | "enterEditModeDisabledTooltip"
  | "hasUnpublishedDraftChanges"
  | "onDiscardDraftAndStartEdit"
  | "unpublishedDraftUpdatedAt"
>) {
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
  hasUnpublishedDraftChanges,
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
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
  | "onExitEditMode"
  | "exitEditModeDisabled"
  | "exitEditModeDisabledTooltip"
  | "onPublishVersion"
  | "publishVersionLabel"
  | "publishVersionDisabled"
  | "publishVersionDisabledTooltip"
>) {
  return (
    <div className="flex items-center gap-2">
      {hasUnpublishedDraftChanges ? (
        <DiscardDraftButton
          onDiscard={() => onDiscardVersion?.()}
          disabled={discardVersionDisabled || !onDiscardVersion}
          disabledTooltip={discardVersionDisabledTooltip}
        />
      ) : null}
      {onExitEditMode ? (
        <ExitEditButton
          onClick={() => onExitEditMode()}
          disabled={!!exitEditModeDisabled}
          disabledTooltip={exitEditModeDisabledTooltip}
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
      variant="outline"
      size="sm"
      onClick={onClick}
      disabled={disabled}
      data-testid="canvas-exit-edit-button"
    >
      Exit edit
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
  if (publishVersionDisabled && publishVersionDisabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">
            <UIButton type="button" variant="default" size="sm" onClick={onPublish} disabled={disabled}>
              {label}
            </UIButton>
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">{publishVersionDisabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <UIButton type="button" variant="default" size="sm" onClick={onPublish} disabled={disabled}>
      {label}
    </UIButton>
  );
}
