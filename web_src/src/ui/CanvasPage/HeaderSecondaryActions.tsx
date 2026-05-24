import { Button as UIButton } from "@/components/ui/button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Switch } from "@/ui/switch";
import { FileCode, Minus, Pencil, Plus } from "lucide-react";

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
  onShowDiff,
  visualDiffEnabled,
  diffCounts,
  showDeletedNodes,
  onToggleShowDeletedNodes,
  showEdgeDiff,
  onToggleShowEdgeDiff,
  onToggleVisualDiff,
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
        hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
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
          onShowDiff={onShowDiff}
          visualDiffEnabled={visualDiffEnabled}
          diffCounts={diffCounts}
          onToggleVisualDiff={onToggleVisualDiff}
          showDeletedNodes={showDeletedNodes}
          onToggleShowDeletedNodes={onToggleShowDeletedNodes}
          showEdgeDiff={showEdgeDiff}
          onToggleShowEdgeDiff={onToggleShowEdgeDiff}
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
              ? "View the console as YAML"
              : "View, copy, download, or import this console as YAML"}
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
      label={hasUnpublishedDraftChanges ? "Continue Editing" : "Edit"}
      disabled={!!enterEditModeDisabled}
      disabledTooltip={enterEditModeDisabledTooltip}
    />
  );
}

function EditModeVersionActions({
  hasUnpublishedDraftChanges,
  onShowDiff,
  visualDiffEnabled,
  diffCounts,
  showDeletedNodes,
  onToggleShowDeletedNodes,
  showEdgeDiff,
  onToggleShowEdgeDiff,
  onToggleVisualDiff,
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
  | "onShowDiff"
  | "visualDiffEnabled"
  | "diffCounts"
  | "showDeletedNodes"
  | "onToggleShowDeletedNodes"
  | "showEdgeDiff"
  | "onToggleShowEdgeDiff"
  | "onToggleVisualDiff"
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
        <>
          {diffCounts && (diffCounts.added > 0 || diffCounts.updated > 0 || diffCounts.removed > 0) && (
            <HoverCard openDelay={100} closeDelay={200}>
              <HoverCardTrigger asChild>
                <button type="button" className="flex items-center gap-0 rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-xs font-medium hover:bg-slate-100 transition-colors cursor-default">
                  {diffCounts.added > 0 && (
                    <span className="flex items-center gap-0.5 text-emerald-600 px-1">
                      <Plus className="h-3 w-3" />{diffCounts.added}
                    </span>
                  )}
                  {diffCounts.added > 0 && (diffCounts.updated > 0 || diffCounts.removed > 0) && (
                    <span className="text-slate-300">|</span>
                  )}
                  {diffCounts.updated > 0 && (
                    <span className="flex items-center gap-0.5 text-sky-600 px-1">
                      <Pencil className="h-3 w-3" />{diffCounts.updated}
                    </span>
                  )}
                  {diffCounts.updated > 0 && diffCounts.removed > 0 && (
                    <span className="text-slate-300">|</span>
                  )}
                  {diffCounts.removed > 0 && (
                    <span className="flex items-center gap-0.5 text-red-600 px-1">
                      <Minus className="h-3 w-3" />{diffCounts.removed}
                    </span>
                  )}
                </button>
              </HoverCardTrigger>
              <HoverCardContent align="start" className="w-auto p-3">
                <div className="flex flex-col gap-2">
                  {onToggleVisualDiff && (
                    <div className="flex items-center gap-1.5 text-xs font-medium text-slate-600">
                      <Switch
                        id="visual-diff-toggle"
                        checked={!!visualDiffEnabled}
                        onCheckedChange={onToggleVisualDiff}
                        data-testid="canvas-toggle-visual-diff"
                      />
                      <label htmlFor="visual-diff-toggle">Diff X-Ray</label>
                    </div>
                  )}
                  {onToggleShowDeletedNodes && (
                    <label className={`flex items-center gap-1.5 text-xs font-medium cursor-pointer ${visualDiffEnabled ? "text-slate-600" : "text-slate-400 cursor-not-allowed"}`}>
                      <input
                        type="checkbox"
                        checked={!!showDeletedNodes}
                        onChange={onToggleShowDeletedNodes}
                        disabled={!visualDiffEnabled}
                        className="h-3.5 w-3.5 rounded border-slate-300 text-slate-600 focus:ring-slate-500 disabled:opacity-50"
                      />
                      Show deleted nodes
                    </label>
                  )}
                  {onToggleShowEdgeDiff && (
                    <label className={`flex items-center gap-1.5 text-xs font-medium cursor-pointer ${visualDiffEnabled ? "text-slate-600" : "text-slate-400 cursor-not-allowed"}`}>
                      <input
                        type="checkbox"
                        checked={!!showEdgeDiff}
                        onChange={onToggleShowEdgeDiff}
                        disabled={!visualDiffEnabled}
                        className="h-3.5 w-3.5 rounded border-slate-300 text-slate-600 focus:ring-slate-500 disabled:opacity-50"
                      />
                      Show edges
                    </label>
                  )}
                </div>
                {onShowDiff && (
                  <div className="-mx-3 -mb-3 mt-2 border-t border-slate-200 px-3 py-2">
                    <button
                      type="button"
                      onClick={onShowDiff}
                      className="text-xs font-medium text-blue-600 hover:text-blue-700 underline-offset-2 hover:underline"
                      data-testid="canvas-show-diff-button"
                    >
                      View full diff
                    </button>
                  </div>
                )}
              </HoverCardContent>
            </HoverCard>
          )}
          <DiscardDraftButton
            onDiscard={() => onDiscardVersion?.()}
            disabled={discardVersionDisabled || !onDiscardVersion}
            disabledTooltip={discardVersionDisabledTooltip}
          />
        </>
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

function HeaderActionSeparator() {
  return <span aria-hidden="true" className="mx-1 h-5 w-px shrink-0 bg-slate-200" />;
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
