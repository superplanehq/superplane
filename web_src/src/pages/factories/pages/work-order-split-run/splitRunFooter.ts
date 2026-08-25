import { getWorkOrderDisplayStatusMeta, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import type { WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";

export type SplitRunFooterKind = "draft" | "running" | "waiting" | "failed" | "done";

/** @deprecated Use SplitRunFooterKind. Kept for fixture field name. */
export type SplitRunFooterTone = SplitRunFooterKind;

export type SplitRunFooterActionKind = "start" | "reject" | "cancel" | "stop-to-draft" | "note-cta";

export interface SplitRunFooterAction {
  id: string;
  kind: SplitRunFooterActionKind;
  label: string;
  emphasis: "primary" | "quiet";
  href?: string;
}

export interface SplitRunFooterNote {
  headline: string;
  text?: string;
  sourceName?: string;
}

export interface SplitRunFooter {
  kind: SplitRunFooterKind;
  sentence: string;
  actions: SplitRunFooterAction[];
  note?: SplitRunFooterNote;
}

const START: SplitRunFooterAction = { id: "start", kind: "start", label: "Start", emphasis: "primary" };
const REJECT: SplitRunFooterAction = { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" };
const CANCEL: SplitRunFooterAction = { id: "cancel", kind: "cancel", label: "Cancel", emphasis: "quiet" };
const STOP_TO_DRAFT: SplitRunFooterAction = {
  id: "stop-to-draft",
  kind: "stop-to-draft",
  label: "Stop and send to Draft",
  emphasis: "quiet",
};

export function toFooterNote(note: WorkOrderStatusNotePresentation): SplitRunFooterNote {
  return {
    headline: note.headline,
    text: note.text || undefined,
    sourceName: note.source?.name,
  };
}

function noteCta(note?: WorkOrderStatusNotePresentation): SplitRunFooterAction | undefined {
  if (!note?.cta) {
    return undefined;
  }
  return {
    id: "note-cta",
    kind: "note-cta",
    label: note.cta.label,
    emphasis: "primary",
    href: note.cta.href,
  };
}

/**
 * The footer is the human-readable centerpiece under the Log: the note row
 * tells the reader where the order came from, what SuperPlane did, and what
 * to do next; the state bar below it holds the always-on actions. Callers
 * synthesize a note for every state — the log carries the raw detail.
 */
export function buildSplitRunFooter(input: {
  kind: SplitRunFooterKind;
  note?: WorkOrderStatusNotePresentation;
  doneSummary?: string;
}): SplitRunFooter {
  const note = input.note ? toFooterNote(input.note) : undefined;
  if (input.kind === "draft") {
    return { kind: "draft", sentence: "This work order is a draft.", note, actions: [START, REJECT] };
  }
  if (input.kind === "running") {
    return { kind: "running", sentence: "This work order is running.", note, actions: [STOP_TO_DRAFT, CANCEL] };
  }
  if (input.kind === "waiting" || input.kind === "failed") {
    const cta = noteCta(input.note);
    return {
      kind: input.kind,
      sentence: "This work order needs attention.",
      note,
      actions: cta ? [CANCEL, cta] : [CANCEL],
    };
  }
  return {
    kind: "done",
    sentence: input.doneSummary ?? getWorkOrderDisplayStatusMeta("completed").summary,
    note,
    actions: [],
  };
}

export function doneFooterForStatus(status: WorkOrderDisplayStatus): SplitRunFooter {
  return buildSplitRunFooter({
    kind: "done",
    doneSummary: getWorkOrderDisplayStatusMeta(status).summary,
  });
}
