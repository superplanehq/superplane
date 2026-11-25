import { describe, expect, it } from "bun:test";

import {
  insertStageBeforeDone,
  lineStageChanges,
  hasLineStageChanges,
  removeLineBoardStage,
  reorderLineBoardStages,
  stagesForSave,
  stageAutoAdvances,
  type LineBoardStage,
} from "./lineBoardStages";

const SAVED: LineBoardStage[] = [
  { id: "backlog", key: "backlog", name: "Backlog", kind: "STAGE_KIND_BACKLOG" },
  { id: "phase-0", key: "phase-0", name: "Implement", kind: "STAGE_KIND_PHASE", stepIndex: 0 },
  { id: "qa", key: "custom-qa", name: "QA", kind: "STAGE_KIND_CUSTOM" },
  { id: "verify", key: "verify", name: "Verify", kind: "STAGE_KIND_VERIFY" },
  { id: "done", key: "done", name: "Done", kind: "STAGE_KIND_DONE" },
];

describe("line stage drafts", () => {
  it("adds a new stage before Done and sends it without an id", () => {
    const draft = insertStageBeforeDone(SAVED, " Deploy ");

    expect(draft.map((stage) => stage.name)).toEqual(["Backlog", "Implement", "QA", "Verify", "Deploy", "Done"]);
    const saved = stagesForSave(draft);
    expect(saved[4]).toMatchObject({ id: "", name: "Deploy", kind: "STAGE_KIND_CUSTOM" });
    expect(saved[4]?.autoAdvance).toBe(true);
    expect(saved[4]?.key).toMatch(/^new-/);
    expect(saved[2]).toMatchObject({ id: "qa", key: "custom-qa" });
  });

  it("summarizes added, removed, and reordered stages", () => {
    const reordered = reorderLineBoardStages(SAVED, "qa", "phase-0");
    const draft = insertStageBeforeDone(removeLineBoardStage(reordered, "verify"), "Deploy");

    const changes = lineStageChanges(SAVED, draft);

    expect(changes.added.map((stage) => stage.name)).toEqual(["Deploy"]);
    expect(changes.removed.map((stage) => stage.name)).toEqual(["Verify"]);
    expect(changes.reordered).toBe(true);
  });

  it("does not report a reorder when only a stage is removed", () => {
    const changes = lineStageChanges(SAVED, removeLineBoardStage(SAVED, "qa"));

    expect(changes.reordered).toBe(false);
    expect(changes.removed.map((stage) => stage.id)).toEqual(["qa"]);
  });

  it("keeps Backlog and Done", () => {
    expect(removeLineBoardStage(SAVED, "done")).toEqual(SAVED);
    expect(removeLineBoardStage(SAVED, "backlog")).toEqual(SAVED);
  });

  it("defaults regular stages on and Verify off", () => {
    expect(stageAutoAdvances(SAVED[1]!)).toBe(true);
    expect(stageAutoAdvances(SAVED[2]!)).toBe(true);
    expect(stageAutoAdvances(SAVED[3]!)).toBe(false);
  });

  it("tracks an auto-advance change", () => {
    const draft = SAVED.map((stage) => (stage.id === "qa" ? { ...stage, autoAdvance: false } : stage));
    const changes = lineStageChanges(SAVED, draft);

    expect(changes.autoAdvanceChanged.map(({ stage }) => stage.id)).toEqual(["qa"]);
    expect(hasLineStageChanges(changes)).toBe(true);
  });
});
