import type { FactoriesFactoryLine, FactoryLineStage } from "@/api-client";

export type LineBoardStage = FactoryLineStage & {
  id: string;
  key: string;
  name: string;
  kind: NonNullable<FactoryLineStage["kind"]>;
};

export function orderedLineBoardStages(
  line: FactoriesFactoryLine,
  phaseNames: ReadonlyMap<number, string> = new Map(),
): LineBoardStage[] {
  const saved = (line.stages ?? []).flatMap((stage) => {
    if (!stage.id || !stage.key || !stage.name || !stage.kind) {
      return [];
    }
    return [{ ...stage, id: stage.id, key: stage.key, name: stage.name, kind: stage.kind }];
  });
  if (saved.length > 0) {
    return saved;
  }

  const phases = (line.steps ?? []).map(
    (_, stepIndex): LineBoardStage => ({
      id: `phase-${stepIndex}`,
      key: `phase-${stepIndex}`,
      name: phaseNames.get(stepIndex) ?? "Step",
      kind: "STAGE_KIND_PHASE",
      stepIndex,
    }),
  );
  return [
    { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" },
    ...phases,
    { id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" },
    { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" },
  ];
}

export function isCustomLineBoardStage(stage: FactoryLineStage): boolean {
  return stage.kind === "STAGE_KIND_CUSTOM";
}

export function reorderLineBoardStages(stages: LineBoardStage[], stageId: string, targetId: string): LineBoardStage[] {
  const fromIndex = stages.findIndex((stage) => stage.id === stageId);
  const toIndex = stages.findIndex((stage) => stage.id === targetId);
  const stage = stages[fromIndex];
  const target = stages[toIndex];
  if (
    fromIndex === toIndex ||
    fromIndex < 1 ||
    toIndex < 1 ||
    fromIndex >= stages.length - 1 ||
    toIndex >= stages.length - 1 ||
    stage?.kind === "STAGE_KIND_BACKLOG" ||
    stage?.kind === "STAGE_KIND_DONE" ||
    target?.kind === "STAGE_KIND_BACKLOG" ||
    target?.kind === "STAGE_KIND_DONE"
  ) {
    return stages;
  }

  const next = [...stages];
  const [moved] = next.splice(fromIndex, 1);
  next.splice(toIndex, 0, moved);
  return next;
}

const NEW_STAGE_KEY_PREFIX = "new-";

/** A stage added in edit mode. The server creates it on save. */
export function isDraftLineBoardStage(stage: Pick<LineBoardStage, "key">): boolean {
  return stage.key.startsWith(NEW_STAGE_KEY_PREFIX);
}

export function insertStageBeforeDone(stages: LineBoardStage[], name: string): LineBoardStage[] {
  const key = `${NEW_STAGE_KEY_PREFIX}${crypto.randomUUID()}`;
  const stage: LineBoardStage = { id: key, key, name: name.trim(), kind: "STAGE_KIND_CUSTOM" };
  const doneIndex = stages.findIndex((entry) => entry.kind === "STAGE_KIND_DONE");
  const index = doneIndex < 0 ? stages.length : doneIndex;
  return [...stages.slice(0, index), stage, ...stages.slice(index)];
}

export function removeLineBoardStage(stages: LineBoardStage[], stageId: string): LineBoardStage[] {
  return stages.filter(
    (stage) => stage.id !== stageId || stage.kind === "STAGE_KIND_BACKLOG" || stage.kind === "STAGE_KIND_DONE",
  );
}

export type LineStageChanges = {
  added: LineBoardStage[];
  removed: LineBoardStage[];
  renamed: { previous: LineBoardStage; stage: LineBoardStage }[];
  reordered: boolean;
};

export function lineStageChanges(saved: LineBoardStage[], draft: LineBoardStage[]): LineStageChanges {
  const draftIds = new Set(draft.map((stage) => stage.id));
  const savedIds = new Set(saved.map((stage) => stage.id));
  const keptSavedOrder = saved.filter((stage) => draftIds.has(stage.id)).map((stage) => stage.id);
  const keptDraftOrder = draft.filter((stage) => savedIds.has(stage.id)).map((stage) => stage.id);
  return {
    added: draft.filter(isDraftLineBoardStage),
    removed: saved.filter((stage) => !draftIds.has(stage.id)),
    renamed: draft.flatMap((stage) => {
      const previous = saved.find((entry) => entry.id === stage.id);
      return previous && previous.name !== stage.name ? [{ previous, stage }] : [];
    }),
    reordered: keptSavedOrder.join() !== keptDraftOrder.join(),
  };
}

export function hasLineStageChanges(changes: LineStageChanges): boolean {
  return changes.added.length > 0 || changes.removed.length > 0 || changes.renamed.length > 0 || changes.reordered;
}

export function lineStageNameError(name: string, stages: LineBoardStage[], stageId?: string): string {
  const trimmedName = name.trim();
  if (!trimmedName) return "Enter a stage name.";
  if ([...trimmedName].length > 64) return "Stage names cannot exceed 64 characters.";
  if (stages.some((stage) => stage.id !== stageId && stage.name.toLowerCase() === trimmedName.toLowerCase())) {
    return "Stage names must be unique.";
  }
  return "";
}

/** New stages go out without an id, so the server assigns their id and key. */
export function stagesForSave(draft: LineBoardStage[]): FactoryLineStage[] {
  return draft.map((stage) => (isDraftLineBoardStage(stage) ? { ...stage, id: "" } : stage));
}
