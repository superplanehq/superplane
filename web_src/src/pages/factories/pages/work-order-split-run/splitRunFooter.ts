import { getWorkOrderDisplayStatusMeta, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import type { WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";

export type SplitRunFooterKind = "draft" | "running" | "waiting" | "failed" | "done";

/** @deprecated Use SplitRunFooterKind. Kept for fixture field name. */
export type SplitRunFooterTone = SplitRunFooterKind;

export type SplitRunFooterActionKind = "start" | "reject" | "approve" | "rerun" | "reopen";

export type SplitRunStopChoice = "canceled" | "completed" | "rerun-step" | "rerun-start" | "reopen";

export const DEFAULT_SPLIT_RUN_STOP_CHOICE: SplitRunStopChoice = "canceled";

export type SplitRunStopChoiceItem = {
  id: SplitRunStopChoice;
  label: string;
  actionLabel: string;
  description: string;
  status: "cancelled" | "completed" | "draft" | "running";
};

export const SPLIT_RUN_STOP_CHOICES: SplitRunStopChoiceItem[] = [
  {
    id: "canceled",
    label: "Stop and Close",
    actionLabel: "Stop and Close",
    description: "Marks this task as Canceled",
    status: "cancelled",
  },
  {
    id: "completed",
    label: "Stop and Complete",
    actionLabel: "Stop and Complete",
    description: "Marks this task as Completed",
    status: "completed",
  },
  {
    id: "rerun-step",
    label: "Rerun this step",
    actionLabel: "Rerun step",
    description: "Starts this step again",
    status: "running",
  },
  {
    id: "rerun-start",
    label: "Rerun from the start",
    actionLabel: "Rerun from start",
    description: "Starts this task from the first step",
    status: "running",
  },
];

export const SPLIT_RUN_REOPEN_CHOICE: SplitRunStopChoiceItem = {
  id: "reopen",
  label: "Reopen",
  actionLabel: "Reopen",
  description: "Opens this work order again",
  status: "draft",
};

export function isSplitRunRerunChoice(choice: SplitRunStopChoice): choice is "rerun-step" | "rerun-start" {
  return choice === "rerun-step" || choice === "rerun-start";
}

export function rerunStartStepIndex(choice: SplitRunStopChoice, currentStepIndex = 0): number {
  if (choice === "rerun-start") {
    return 0;
  }
  return Math.max(0, currentStepIndex);
}

export function splitRunCloseNeedsConfirm(kind: SplitRunFooterKind): boolean {
  return kind === "running";
}

export function isClosedWorkOrderDisplayStatus(status?: WorkOrderDisplayStatus): boolean {
  return status === "completed" || status === "failed" || status === "rejected" || status === "cancelled";
}

export function isSplitRunStopChoiceAvailable(choice: SplitRunStopChoice, status?: WorkOrderDisplayStatus): boolean {
  if (choice === "reopen") {
    return isClosedWorkOrderDisplayStatus(status);
  }
  if (isClosedWorkOrderDisplayStatus(status)) {
    return false;
  }
  if (choice === "completed") {
    return status !== "completed";
  }
  if (choice === "canceled") {
    return status !== "cancelled" && status !== "rejected";
  }
  return status !== "draft";
}

export function availableSplitRunStopChoices(status?: WorkOrderDisplayStatus): SplitRunStopChoiceItem[] {
  if (isClosedWorkOrderDisplayStatus(status)) {
    return [SPLIT_RUN_REOPEN_CHOICE];
  }
  return SPLIT_RUN_STOP_CHOICES.filter((item) => isSplitRunStopChoiceAvailable(item.id, status));
}

export function defaultSplitRunStopChoice(
  status?: WorkOrderDisplayStatus,
  kind?: SplitRunFooterKind,
): SplitRunStopChoice | undefined {
  const available = availableSplitRunStopChoices(status);
  const preferred = kind === "failed" ? "rerun-step" : DEFAULT_SPLIT_RUN_STOP_CHOICE;
  if (available.some((item) => item.id === preferred)) {
    return preferred;
  }
  return available[0]?.id;
}

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
  /** When true, the decision note renders as a sticky strip. */
  attentionCard?: boolean;
  run?: { appId: string; runId: string };
  status?: WorkOrderDisplayStatus;
}

const REJECT: SplitRunFooterAction = { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" };
const APPROVE: SplitRunFooterAction = { id: "approve", kind: "approve", label: "Approve", emphasis: "primary" };
const RERUN: SplitRunFooterAction = { id: "rerun", kind: "rerun", label: "Rerun", emphasis: "primary" };
const START: SplitRunFooterAction = { id: "start", kind: "start", label: "Start", emphasis: "primary" };
const REOPEN: SplitRunFooterAction = { id: "reopen", kind: "reopen", label: "Reopen", emphasis: "primary" };

export const SPLIT_RUN_WAITING_NOTE: SplitRunFooterNote = {
  headline: "This task waits on a person",
  text: "No automation is running. Click Approve if the result is good. Click Reject to close this task as rejected.",
};

export const SPLIT_RUN_FAILED_NOTE_TEXT =
  "This automation failed. Open the run to review the error. Fix the automation, then click Rerun. Or close this task.";

export const SPLIT_RUN_DRAFT_NOTE: SplitRunFooterNote = {
  headline: "This task is ready to start",
  text: "Review the details. Change anything you need. Then click Start to send it to the line.",
};

export type SplitRunDecisionTone = "draft" | "waiting" | "failed" | "done" | "rejected";

export function splitRunDecisionTone(footer: SplitRunFooter): SplitRunDecisionTone {
  if (footer.kind === "draft") {
    return "draft";
  }
  if (footer.kind === "waiting") {
    return "waiting";
  }
  if (footer.kind === "failed" || footer.status === "failed") {
    return "failed";
  }
  if (footer.status === "rejected" || footer.status === "cancelled") {
    return "rejected";
  }
  return "done";
}

function closedDecisionNote(status?: WorkOrderDisplayStatus): SplitRunFooterNote {
  if (status === "rejected") {
    return {
      headline: "This task is rejected",
      text: "A person closed this task. The line automations did not finish the work.",
    };
  }
  if (status === "cancelled") {
    return { headline: "This task is canceled", text: "Reopen this task if the work should continue." };
  }
  if (status === "failed") {
    return { headline: "This task is closed as failed", text: "Reopen this task to start the line again." };
  }
  return {
    headline: "This task finished successfully",
    text: "The line automations completed every step.",
  };
}

function closedDecisionActions(status?: WorkOrderDisplayStatus): SplitRunFooterAction[] {
  if (status === "completed" || status === "rejected") {
    return [];
  }
  return [REOPEN];
}

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
 * Decision strip for the work-order popup. Running has no strip. Open
 * states keep Reject, Approve, Rerun, or Start on the note. Closed failed
 * keeps Reopen. Completed and rejected explain the result only.
 */
type FooterInput = {
  kind: SplitRunFooterKind;
  note?: WorkOrderStatusNotePresentation;
  doneSummary?: string;
  /** When false, hide the decision strip. Used while a follow-up run is active. */
  decision?: boolean;
  run?: { appId: string; runId: string };
  status?: WorkOrderDisplayStatus;
};

function withFooterMeta(input: FooterInput, footer: SplitRunFooter): SplitRunFooter {
  const next = input.status ? { ...footer, status: input.status } : footer;
  return input.run ? { ...next, run: input.run } : next;
}

function hiddenDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  return withFooterMeta(input, {
    kind: input.kind,
    sentence: input.kind === "running" ? "This work order is running." : "This work order needs attention.",
    note,
    actions: [],
  });
}

function draftDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  return withFooterMeta(input, {
    kind: "draft",
    sentence: "This work order is a draft.",
    note,
    attentionCard: true,
    actions: [REJECT, START],
  });
}

function closedDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  const closedNote =
    input.status === "failed" || input.kind === "failed"
      ? closedDecisionNote(input.status ?? "failed")
      : (note ?? closedDecisionNote(input.status));
  if (input.kind === "failed") {
    return withFooterMeta(input, {
      kind: "failed",
      sentence: "This work order needs attention.",
      note: closedNote,
      attentionCard: true,
      actions: closedDecisionActions(input.status ?? "failed"),
    });
  }
  return withFooterMeta(input, {
    kind: "done",
    sentence: input.doneSummary ?? getWorkOrderDisplayStatusMeta(input.status ?? "completed").summary,
    note: closedNote,
    attentionCard: true,
    actions: closedDecisionActions(input.status ?? "completed"),
  });
}

function openDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  const actions = input.kind === "failed" ? [REJECT, RERUN] : [REJECT, APPROVE];
  return withFooterMeta(input, {
    kind: input.kind,
    sentence: "This work order needs attention.",
    note: note ?? (input.kind === "waiting" ? { ...SPLIT_RUN_WAITING_NOTE } : undefined),
    attentionCard: true,
    actions,
  });
}

export function buildSplitRunFooter(input: FooterInput): SplitRunFooter {
  const note = input.note ? toFooterNote(input.note) : undefined;
  if (input.kind === "running" || input.decision === false) {
    return hiddenDecisionFooter(input, note);
  }
  if (input.kind === "draft") {
    return draftDecisionFooter(input, note);
  }
  if (isClosedWorkOrderDisplayStatus(input.status) || input.kind === "done") {
    return closedDecisionFooter(input, note);
  }
  if (input.kind === "waiting" || input.kind === "failed") {
    return openDecisionFooter(input, note);
  }
  return closedDecisionFooter({ ...input, kind: "done" }, note);
}

export function doneFooterForStatus(status: WorkOrderDisplayStatus): SplitRunFooter {
  return buildSplitRunFooter({
    kind: "done",
    doneSummary: getWorkOrderDisplayStatusMeta(status).summary,
    status,
  });
}
