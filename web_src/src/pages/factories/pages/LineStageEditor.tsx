import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Separator } from "@/components/ui/separator";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";
import { Check, GripHorizontal, Lock, Plus, Trash2 } from "lucide-react";
import type { DragEvent, KeyboardEvent } from "react";

import type { LineBoardStage, LineStageChanges } from "../lib/lineBoardStages";

export function LineStageEditToolbar({
  hasChanges,
  saving,
  onAddStage,
  onDiscard,
  onSave,
}: {
  hasChanges: boolean;
  saving: boolean;
  onAddStage: () => void;
  onDiscard: () => void;
  onSave: () => void;
}) {
  return (
    <div
      role="toolbar"
      aria-label="Edit stages"
      data-testid="line-stage-edit-toolbar"
      className="mb-3 flex shrink-0 flex-wrap items-center gap-3"
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="text-[13px] font-medium text-foreground">Editing stages</p>
          {hasChanges ? (
            <span
              className="inline-flex items-center gap-1.5 text-xs text-muted-foreground"
              data-testid="line-stage-edit-unsaved"
            >
              <span className="size-1.5 rounded-full bg-amber-500" aria-hidden />
              Unsaved changes
            </span>
          ) : null}
        </div>
        <p className="text-xs text-muted-foreground">Drag stages to reorder. Backlog and Done stay in place.</p>
      </div>
      <div className="flex items-center gap-2">
        <Button type="button" variant="ghost" size="sm" onClick={onAddStage} disabled={saving}>
          <Plus className="size-3.5" aria-hidden />
          Add stage
        </Button>
        <Separator orientation="vertical" className="data-[orientation=vertical]:h-5" />
        {hasChanges ? (
          <>
            <Button type="button" variant="ghost" size="sm" onClick={onDiscard} disabled={saving}>
              Discard
            </Button>
            <LoadingButton type="button" size="sm" onClick={onSave} loading={saving} loadingText="Saving...">
              <Check className="size-3.5" aria-hidden />
              Save changes
            </LoadingButton>
          </>
        ) : (
          <Button type="button" variant="outline" size="sm" onClick={onDiscard}>
            Done
          </Button>
        )}
      </div>
    </div>
  );
}

export function LineStageEditStrip({
  stage,
  onDragStart,
  onDragEnd,
  onKeyDown,
  onDelete,
}: {
  stage: LineBoardStage;
  onDragStart: (event: DragEvent<HTMLButtonElement>) => void;
  onDragEnd: () => void;
  onKeyDown: (event: KeyboardEvent<HTMLButtonElement>) => void;
  onDelete?: () => void;
}) {
  if (stage.kind === "STAGE_KIND_BACKLOG" || stage.kind === "STAGE_KIND_DONE") {
    return (
      <div className="flex h-7 shrink-0 items-center justify-center gap-1 text-xs text-muted-foreground">
        <Lock className="size-3" aria-hidden />
        Fixed
      </div>
    );
  }
  return (
    <div className="flex h-7 shrink-0 items-center justify-between gap-1 px-1">
      <span className="size-6" aria-hidden />
      <Button
        type="button"
        draggable
        variant="ghost"
        size="icon"
        aria-label={`Reorder ${stage.name}`}
        title="Drag to reorder stage"
        data-testid={`lines-stage-reorder-${stage.key}`}
        className="h-6 w-16 cursor-grab touch-none rounded-sm text-muted-foreground hover:text-foreground active:cursor-grabbing"
        onDragStart={onDragStart}
        onDragEnd={onDragEnd}
        onKeyDown={onKeyDown}
      >
        <GripHorizontal className="size-4" aria-hidden />
      </Button>
      {onDelete ? (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={`Delete ${stage.name}`}
          title="Delete stage"
          data-testid={`lines-stage-delete-${stage.key}`}
          className="size-6 rounded-sm text-muted-foreground hover:text-destructive"
          onClick={onDelete}
        >
          <Trash2 className="size-3.5" aria-hidden />
        </Button>
      ) : (
        <span className="size-6" aria-hidden />
      )}
    </div>
  );
}

export function SaveLineStagesDialog({
  changes,
  saving,
  error,
  onCancel,
  onConfirm,
}: {
  changes: LineStageChanges | null;
  saving: boolean;
  error: string;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <AlertDialog open={Boolean(changes)} onOpenChange={(open) => !open && !saving && onCancel()}>
      <AlertDialogContent data-testid="save-line-stages-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>Save stage changes?</AlertDialogTitle>
          <AlertDialogDescription>
            This changes the board for everyone who uses this line. Deleted stages keep their automations in the
            workspace.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {changes ? (
          <ul className="space-y-1 text-sm text-foreground" data-testid="save-line-stages-summary">
            {changes.added.map((stage) => (
              <li key={stage.id}>Add: {stage.name}</li>
            ))}
            {changes.removed.map((stage) => (
              <li key={stage.id}>Delete: {stage.name}</li>
            ))}
            {changes.renamed.map(({ previous, stage }) => (
              <li key={stage.id}>
                Rename: {previous.name} → {stage.name}
              </li>
            ))}
            {changes.reordered ? <li>Stage order changes</li> : null}
          </ul>
        ) : null}
        {error ? (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        ) : null}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={saving}>Cancel</AlertDialogCancel>
          <AlertDialogAction
            disabled={saving}
            onClick={(event) => {
              event.preventDefault();
              onConfirm();
            }}
          >
            Save changes
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

export function DiscardLineStagesDialog({
  open,
  onKeepEditing,
  onDiscard,
}: {
  open: boolean;
  onKeepEditing: () => void;
  onDiscard: () => void;
}) {
  return (
    <AlertDialog open={open} onOpenChange={(next) => !next && onKeepEditing()}>
      <AlertDialogContent data-testid="discard-line-stages-dialog">
        <AlertDialogHeader>
          <AlertDialogTitle>Discard changes?</AlertDialogTitle>
          <AlertDialogDescription>Your stage changes are not saved.</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Keep editing</AlertDialogCancel>
          <AlertDialogAction onClick={onDiscard}>Discard changes</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
