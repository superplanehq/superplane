import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { FileCode, GitBranch, Pencil, Plus } from "lucide-react";

import { Button } from "../button";
import { EnterEditDraftDropdown } from "./components/EnterEditDraftDropdown";
import type { HeaderProps } from "./Header";

export function SecondaryHeaderActions({
  mode,
  onSave,
  saveButtonHidden,
  saveDisabled,
  saveDisabledTooltip,
  saveIsPrimary,
  hasUnpublishedDraftChanges,
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  onPublishVersion,
  publishVersionLabel,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  unpublishedDraftUpdatedAt,
  onDiscardDraftAndStartEdit,
  onDashboardAddPanel,
  onDashboardOpenYaml,
  dashboardYamlReadOnly,
}: HeaderProps) {
  const showEditButton = mode === "version-live" && !!onEnterEditMode;
  const showDraftDropdown =
    showEditButton && !!hasUnpublishedDraftChanges && !!onDiscardDraftAndStartEdit && !enterEditModeDisabled;
  const showDashboardAddPanel = mode === "dashboard" && !!onDashboardAddPanel;
  const showDashboardYaml = mode === "dashboard" && !!onDashboardOpenYaml;

  return (
    <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
      <LiveModeEditControls
        showEditButton={showEditButton}
        showDraftDropdown={showDraftDropdown}
        onEnterEditMode={onEnterEditMode}
        enterEditModeDisabled={enterEditModeDisabled}
        enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
        onDiscardDraftAndStartEdit={onDiscardDraftAndStartEdit}
        unpublishedDraftUpdatedAt={unpublishedDraftUpdatedAt}
      />

      {mode === "default" && onSave && !saveButtonHidden ? (
        <SaveButton
          onSave={onSave}
          saveDisabled={saveDisabled}
          saveDisabledTooltip={saveDisabledTooltip}
          saveIsPrimary={saveIsPrimary}
        />
      ) : null}

      {mode === "version-edit" ? (
        <EditModeVersionActions
          hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
          onDiscardVersion={onDiscardVersion}
          discardVersionDisabled={discardVersionDisabled}
          discardVersionDisabledTooltip={discardVersionDisabledTooltip}
          onExitEditMode={onExitEditMode}
          exitEditModeDisabled={exitEditModeDisabled}
          exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
          onPublishVersion={onPublishVersion}
          publishVersionLabel={publishVersionLabel}
          publishVersionDisabled={publishVersionDisabled}
          publishVersionDisabledTooltip={publishVersionDisabledTooltip}
        />
      ) : null}

      {showDashboardYaml ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <UIButton
              type="button"
              size="sm"
              variant="outline"
              onClick={() => onDashboardOpenYaml!()}
              data-testid="dashboard-yaml-button"
              aria-label={dashboardYamlReadOnly ? "View YAML" : "View / Import YAML"}
            >
              <FileCode className="mr-1 h-3.5 w-3.5" />
              YAML
            </UIButton>
          </TooltipTrigger>
          <TooltipContent side="bottom">
            {dashboardYamlReadOnly
              ? "View the dashboard as YAML"
              : "View, copy, download, or import this dashboard as YAML"}
          </TooltipContent>
        </Tooltip>
      ) : null}

      {showDashboardAddPanel ? (
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
    </div>
  );
}

function LiveModeEditControls({
  showEditButton,
  showDraftDropdown,
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onDiscardDraftAndStartEdit,
  unpublishedDraftUpdatedAt,
}: Pick<
  HeaderProps,
  | "onEnterEditMode"
  | "enterEditModeDisabled"
  | "enterEditModeDisabledTooltip"
  | "onDiscardDraftAndStartEdit"
  | "unpublishedDraftUpdatedAt"
> & {
  showEditButton: boolean;
  showDraftDropdown: boolean;
}) {
  if (!showEditButton || !onEnterEditMode) {
    return null;
  }

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
      disabled={!!enterEditModeDisabled}
      disabledTooltip={enterEditModeDisabledTooltip}
    />
  );
}

function EditModeVersionActions({
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
      variant="default"
      size="sm"
      onClick={onClick}
      disabled={disabled}
      data-testid="canvas-edit-button"
    >
      <Pencil className="h-3.5 w-3.5" />
      Edit
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
