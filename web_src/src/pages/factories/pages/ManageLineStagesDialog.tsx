import type { FactoriesFactoryLine, FactoryLineStage } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useUpdateFactoryLine } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { reorderListItems } from "@/lib/reorderListItems";
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
import { useListFieldDragReorder } from "@/ui/configurationFieldRenderer/useListFieldDragReorder";
import { ArrowDown, ArrowUp, GripVertical, Lock, Plus, Trash2 } from "lucide-react";
import { useMemo, useRef, useState, type MouseEvent } from "react";

import { isCustomLineBoardStage, orderedLineBoardStages } from "../lib/lineBoardStages";
import {
  LINE_BOARD_COLUMN_COLORS,
  normalizeColumnColors,
  serializeColumnColors,
  type LineBoardColumnColorId,
} from "./lineBoardColumnColors";

type DraftStage = Omit<FactoryLineStage, "id"> & { clientId: string; id?: string; name: string };

export function ManageLineStagesDialog({
  organizationId,
  factoryId,
  line,
  addStageOnOpen,
  onClose,
}: {
  organizationId: string;
  factoryId: string;
  line: FactoriesFactoryLine;
  addStageOnOpen: boolean;
  onClose: () => void;
}) {
  const initial = useMemo(() => initialDraft(line, addStageOnOpen), [addStageOnOpen, line]);
  const [stages, setStages] = useState<DraftStage[]>(initial.stages);
  const [colors, setColors] = useState(initial.colors);
  const [error, setError] = useState("");
  const [confirmClose, setConfirmClose] = useState(false);
  const [openItem, setOpenItem] = useState("");
  const rowRefs = useRef<Array<HTMLDivElement | null>>([]);
  const updateLine = useUpdateFactoryLine(organizationId, factoryId);
  const dirty = snapshot(stages, colors) !== initial.snapshot;
  const validationError = validateStages(stages);
  const { renderedItems, startDrag } = useListFieldDragReorder({
    items: stages,
    allowReorder: true,
    useAccordion: false,
    onChange: (next) => setStages(keepStageBookends(next as DraftStage[])),
    setOpenItem,
    rowRefs,
  });

  const requestClose = () => (dirty ? setConfirmClose(true) : onClose());

  const save = async () => {
    if (validationError || !line.id) {
      return;
    }
    setError("");
    try {
      await updateLine.mutateAsync({
        lineId: line.id,
        stageConfiguration: { stages: stages.map(stageRequest) },
        columnColors: serializeColumnColors(colors),
      });
      onClose();
    } catch (saveError) {
      setError(getApiErrorMessage(saveError, "SuperPlane could not save the stages."));
    }
  };

  const addStage = () => {
    const doneIndex = stages.findIndex((stage) => stage.kind === "STAGE_KIND_DONE");
    const next = newDraftStage();
    const index = doneIndex < 0 ? stages.length : doneIndex;
    setStages([...stages.slice(0, index), next, ...stages.slice(index)]);
  };

  return (
    <>
      <Dialog open onOpenChange={(open) => !open && requestClose()}>
        <DialogContent
          className="max-h-[85vh] overflow-hidden p-0 sm:max-w-2xl"
          data-testid="manage-line-stages-dialog"
        >
          <DialogHeader className="border-b border-border px-6 py-5">
            <DialogTitle>Manage stages</DialogTitle>
            <DialogDescription>Edit stage names, colors, and order for this line.</DialogDescription>
          </DialogHeader>
          <div className="min-h-0 overflow-y-auto px-6">
            <ol className="divide-y divide-border" aria-label="Line stages">
              {(renderedItems as DraftStage[]).map((stage, index) => (
                <li key={stage.clientId}>
                  <div
                    ref={(element) => {
                      rowRefs.current[index] = element;
                    }}
                    className="flex items-center gap-2 py-3"
                    data-testid={`manage-stage-row-${stage.clientId}`}
                  >
                    <StageRow
                      stage={stage}
                      index={index}
                      stageCount={stages.length}
                      color={colors[stage.key ?? stage.clientId] ?? null}
                      autoFocus={addStageOnOpen && stage.clientId.startsWith("new-")}
                      onDragStart={(event) => startDrag(event, index, openItem)}
                      onNameChange={(name) => updateStage(index, { name }, stages, setStages)}
                      onColorChange={(color) =>
                        setColors((current) => ({ ...current, [stage.key ?? stage.clientId]: color }))
                      }
                      onMove={(offset) => setStages(moveStage(stages, index, index + offset))}
                      onRemove={() => setStages(stages.filter((_, position) => position !== index))}
                    />
                  </div>
                </li>
              ))}
            </ol>
            <Button type="button" variant="ghost" className="my-2 gap-2" onClick={addStage}>
              <Plus className="size-4" aria-hidden />
              Add stage
            </Button>
            {validationError || error ? (
              <p role="alert" className="pb-3 text-sm text-destructive">
                {validationError || error}
              </p>
            ) : null}
          </div>
          <ManageStagesFooter
            saveDisabled={Boolean(validationError) || updateLine.isPending}
            onCancel={requestClose}
            onSave={() => void save()}
          />
        </DialogContent>
      </Dialog>
      <DiscardStageChangesDialog open={confirmClose} onOpenChange={setConfirmClose} onDiscard={onClose} />
    </>
  );
}

function ManageStagesFooter({
  saveDisabled,
  onCancel,
  onSave,
}: {
  saveDisabled: boolean;
  onCancel: () => void;
  onSave: () => void;
}) {
  return (
    <DialogFooter className="border-t border-border px-6 py-4">
      <Button type="button" variant="outline" onClick={onCancel}>
        Cancel
      </Button>
      <Button type="button" disabled={saveDisabled} onClick={onSave}>
        Save stages
      </Button>
    </DialogFooter>
  );
}

function DiscardStageChangesDialog({
  open,
  onOpenChange,
  onDiscard,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDiscard: () => void;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Discard stage changes?</AlertDialogTitle>
          <AlertDialogDescription>Your unsaved stage changes will be lost.</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Keep editing</AlertDialogCancel>
          <AlertDialogAction onClick={onDiscard}>Discard changes</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function StageRow({
  stage,
  index,
  stageCount,
  color,
  autoFocus,
  onDragStart,
  onNameChange,
  onColorChange,
  onMove,
  onRemove,
}: {
  stage: DraftStage;
  index: number;
  stageCount: number;
  color: LineBoardColumnColorId | null;
  autoFocus: boolean;
  onDragStart: (event: MouseEvent) => void;
  onNameChange: (name: string) => void;
  onColorChange: (color: LineBoardColumnColorId | null) => void;
  onMove: (offset: -1 | 1) => void;
  onRemove: () => void;
}) {
  const custom = isCustomLineBoardStage(stage);
  const label = stage.name || "New stage";
  const actionLabel = stage.name || "new stage";
  const inputId = `stage-name-${stage.clientId}`;
  return (
    <>
      {custom ? (
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          className="size-7 cursor-grab text-muted-foreground"
          aria-label={`Drag ${actionLabel}`}
          onMouseDown={onDragStart}
        >
          <GripVertical className="size-4" aria-hidden />
        </Button>
      ) : (
        <span className="p-1 text-muted-foreground" title="System stage">
          <Lock className="size-4" aria-hidden />
        </span>
      )}
      <Label htmlFor={inputId} className="sr-only">
        {label} name
      </Label>
      <Input
        id={inputId}
        value={stage.name}
        maxLength={64}
        autoFocus={autoFocus}
        onChange={(event) => onNameChange(event.target.value)}
      />
      <Select
        value={color ?? "none"}
        onValueChange={(value) => onColorChange(value === "none" ? null : (value as LineBoardColumnColorId))}
      >
        <SelectTrigger className="w-28" aria-label={`${label} color`}>
          <SelectValue placeholder="Color" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="none">Default</SelectItem>
          {LINE_BOARD_COLUMN_COLORS.map((option) => (
            <SelectItem key={option.id} value={option.id}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {custom ? (
        <div className="flex items-center">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`Move ${actionLabel} up`}
            disabled={index <= 1}
            onClick={() => onMove(-1)}
          >
            <ArrowUp className="size-4" aria-hidden />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`Move ${actionLabel} down`}
            disabled={index >= stageCount - 2}
            onClick={() => onMove(1)}
          >
            <ArrowDown className="size-4" aria-hidden />
          </Button>
          <Button type="button" variant="ghost" size="icon" aria-label={`Remove ${actionLabel}`} onClick={onRemove}>
            <Trash2 className="size-4" aria-hidden />
          </Button>
        </div>
      ) : null}
    </>
  );
}

function initialDraft(line: FactoriesFactoryLine, addStage: boolean) {
  const stages: DraftStage[] = orderedLineBoardStages(line).map((stage) => ({
    ...stage,
    clientId: stage.id ?? stage.key ?? crypto.randomUUID(),
    name: stage.name ?? "",
  }));
  if (addStage) {
    const doneIndex = stages.findIndex((stage) => stage.kind === "STAGE_KIND_DONE");
    stages.splice(doneIndex < 0 ? stages.length : doneIndex, 0, newDraftStage());
  }
  const colors = normalizeColumnColors(line.columnColors);
  return { stages, colors, snapshot: snapshot(stages, colors) };
}

function newDraftStage(): DraftStage {
  const clientId = `new-${crypto.randomUUID()}`;
  return { clientId, key: clientId, name: "", kind: "STAGE_KIND_CUSTOM" };
}

function stageRequest({ clientId: _clientId, ...stage }: DraftStage): FactoryLineStage {
  return { ...stage, id: stage.id ?? "", name: stage.name.trim() };
}

function updateStage(
  index: number,
  update: Partial<DraftStage>,
  stages: DraftStage[],
  setStages: (stages: DraftStage[]) => void,
) {
  setStages(stages.map((stage, position) => (position === index ? { ...stage, ...update } : stage)));
}

function moveStage(stages: DraftStage[], from: number, to: number): DraftStage[] {
  return keepStageBookends(reorderListItems(stages, from, to) as DraftStage[]);
}

function keepStageBookends(stages: DraftStage[]): DraftStage[] {
  const backlog = stages.find((stage) => stage.kind === "STAGE_KIND_BACKLOG");
  const done = stages.find((stage) => stage.kind === "STAGE_KIND_DONE");
  const middle = stages.filter((stage) => stage !== backlog && stage !== done);
  return [...(backlog ? [backlog] : []), ...middle, ...(done ? [done] : [])];
}

function validateStages(stages: DraftStage[]): string {
  if (stages.length > 64) {
    return "A line cannot have more than 64 stages.";
  }
  const names = new Set<string>();
  for (const stage of stages) {
    const name = stage.name.trim();
    if (!name) {
      return "Enter a name for each stage.";
    }
    if ([...name].length > 64) {
      return "Stage names cannot have more than 64 characters.";
    }
    const key = name.toLocaleLowerCase();
    if (names.has(key)) {
      return "Stage names must be unique.";
    }
    names.add(key);
  }
  return "";
}

function snapshot(stages: DraftStage[], colors: Record<string, LineBoardColumnColorId | null>): string {
  return JSON.stringify({ stages: stages.map(stageRequest), colors: serializeColumnColors(colors) });
}
