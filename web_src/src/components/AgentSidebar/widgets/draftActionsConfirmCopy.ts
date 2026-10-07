export type DraftActionsConfirmKind = "commit" | "save";

export function draftActionsConfirmCopy(kind: DraftActionsConfirmKind): { idle: string; busy: string } {
  if (kind === "save") {
    return { idle: "Save", busy: "Saving..." };
  }
  return { idle: "Commit", busy: "Committing..." };
}
