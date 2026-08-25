import { getWorkOrderDisplayStatusMeta, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import type { WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";

export type SplitRunFooterKind = "draft" | "running" | "waiting" | "failed" | "done";

/** @deprecated Use SplitRunFooterKind. Kept for fixture field name. */
export type SplitRunFooterTone = SplitRunFooterKind;

export type SplitRunFooterActionKind = "start" | "reject" | "stop";

export type SplitRunStopChoice = "canceled" | "completed" | "draft";

export const DEFAULT_SPLIT_RUN_STOP_CHOICE: SplitRunStopChoice = "canceled";

export const SPLIT_RUN_STOP_CHOICES: {
  id: SplitRunStopChoice;
  label: string;
  actionLabel: string;
  description: string;
  status: "cancelled" | "completed" | "draft";
}[] = [
  {
    id: "canceled",
    label: "Stop as Canceled",
    actionLabel: "Stop & Cancel",
    description: "Marks this task as Canceled",
    status: "cancelled",
  },
  {
    id: "completed",
    label: "Stop as Completed",
    actionLabel: "Mark as Complete",
    description: "Marks this task as Completed",
    status: "completed",
  },
  {
    id: "draft",
    label: "Stop and return to Draft",
    actionLabel: "Return to Draft",
    description: "Returns this task to Backlog",
    status: "draft",
  },
];

export interface SplitRunFooterAction {
  id: string;
  kind: SplitRunFooterActionKind;
  label: string;
  emphasis: "primary" | "quiet";
}

export interface SplitRunFooterNote {
  headline: string;
  text?: string;
  sourceName?: string;
  sourceAppId?: string;
  updatedAt?: string;
  cta?: { label: string; href?: string };
}

export interface SplitRunFooter {
  kind: SplitRunFooterKind;
  sentence: string;
  actions: SplitRunFooterAction[];
  note?: SplitRunFooterNote;
  /** When true, the waiting or failed note renders as a sticky strip above Stop. */
  attentionCard?: boolean;
  run?: { appId: string; runId: string };
}

const START: SplitRunFooterAction = { id: "start", kind: "start", label: "Start", emphasis: "primary" };
const REJECT: SplitRunFooterAction = { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" };
const STOP: SplitRunFooterAction = { id: "stop", kind: "stop", label: "Stop", emphasis: "quiet" };

export function toFooterNote(note: WorkOrderStatusNotePresentation): SplitRunFooterNote {
  return {
    headline: note.headline,
    ...(note.text ? { text: note.text } : {}),
    ...(note.source?.name ? { sourceName: note.source.name } : {}),
    ...(note.source?.appId ? { sourceAppId: note.source.appId } : {}),
    ...(note.updatedAt ? { updatedAt: note.updatedAt } : {}),
    ...(note.cta ? { cta: note.cta } : {}),
  };
}

/**
 * The footer is the action bar under Description and Log. Draft keeps
 * Reject and Start. Implement and Verify keep Stop. A visible waiting
 * note sits above Stop as an attention card.
 */
export function buildSplitRunFooter(input: {
  kind: SplitRunFooterKind;
  note?: WorkOrderStatusNotePresentation;
  doneSummary?: string;
  attentionCard?: boolean;
  run?: { appId: string; runId: string };
}): SplitRunFooter {
  const note = input.note ? toFooterNote(input.note) : undefined;
  const attentionCard = input.attentionCard || undefined;
  if (input.kind === "draft") {
    return { kind: "draft", sentence: "This work order is a draft.", note, actions: [REJECT, START] };
  }
  if (input.kind === "running") {
    return { kind: "running", sentence: "This work order is running.", note, actions: [STOP] };
  }
  if (input.kind === "waiting" || input.kind === "failed") {
    return {
      kind: input.kind,
      sentence: "This work order needs attention.",
      note,
      attentionCard,
      run: input.run,
      actions: [STOP],
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
