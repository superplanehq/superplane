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
