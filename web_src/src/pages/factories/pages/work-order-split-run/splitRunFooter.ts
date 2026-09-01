import type { OrgUserDisplay } from "@/lib/orgUserDisplay";

import { getWorkOrderDisplayStatusMeta, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import type { WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";

export type SplitRunFooterKind = "draft" | "running" | "waiting" | "failed" | "stopped" | "done";

/** @deprecated Use SplitRunFooterKind. Kept for fixture field name. */
export type SplitRunFooterTone = SplitRunFooterKind;

export type SplitRunFooterActionKind = "start" | "reject" | "approve" | "rerun" | "reopen" | "back-to-draft";

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
  description: "Opens this task again",
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
  const preferred = kind === "failed" || kind === "stopped" ? "rerun-step" : DEFAULT_SPLIT_RUN_STOP_CHOICE;
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
  icon?: "undo-2";
}

export interface SplitRunFooterNote {
  headline: string;
  text?: string;
  sourceName?: string;
  sourceAppId?: string;
  updatedAt?: string;
  cta?: { label: string; href?: string; icon?: "bug" };
  actor?: OrgUserDisplay;
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
const BACK_TO_DRAFT: SplitRunFooterAction = {
  id: "back-to-draft",
  kind: "back-to-draft",
  label: "To Backlog",
  emphasis: "quiet",
  icon: "undo-2",
};

export const SPLIT_RUN_WAITING_NOTE: SplitRunFooterNote = {
  headline: "This task needs a decision",
  text: "Every automation finished. This task is ready to complete.",
};

export const SPLIT_RUN_FAILED_NOTE_TEXT = "This automation did not finish. Fix the error, then run this step again.";

export const SPLIT_RUN_STOPPED_NOTE_TEXT = "This automation did not finish. This task still needs a decision.";

export const SPLIT_RUN_STOPPED_HEADLINE = "stopped this automation";
export const SPLIT_RUN_STOPPED_HEADLINE_UNKNOWN = "A person stopped this automation";

export const SPLIT_RUN_COMPLETED_NOTE_TEXT = "The work is done. The result met the goal.";
export const SPLIT_RUN_REJECTED_NOTE_TEXT = "The work is done. The result did not meet the goal.";
export const SPLIT_RUN_COMPLETED_HEADLINE = "This task succeeded";
export const SPLIT_RUN_REJECTED_HEADLINE = "This task did not succeed";
export const SPLIT_RUN_COMPLETED_HEADLINE_ACTOR = "marked this task as successful";
export const SPLIT_RUN_REJECTED_HEADLINE_ACTOR = "marked this task as unsuccessful";

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
  if (footer.kind === "stopped" || footer.status === "rejected" || footer.status === "cancelled") {
    return "rejected";
  }
  return "done";
}

function closerHeadline(verb: string, actor?: OrgUserDisplay, automationName?: string): string {
  if (actor) {
    return verb;
  }
  if (automationName) {
    return `${automationName} ${verb}`;
  }
  return "";
}

function closedDecisionNote(
  status?: WorkOrderDisplayStatus,
  closer?: { actor?: OrgUserDisplay; automationName?: string },
): SplitRunFooterNote {
  if (status === "rejected") {
    const headline =
      closerHeadline(SPLIT_RUN_REJECTED_HEADLINE_ACTOR, closer?.actor, closer?.automationName) ||
      SPLIT_RUN_REJECTED_HEADLINE;
    return {
      headline,
      text: SPLIT_RUN_REJECTED_NOTE_TEXT,
      ...(closer?.actor ? { actor: closer.actor } : {}),
    };
  }
  if (status === "cancelled") {
    return { headline: "This task is canceled", text: "Reopen this task if the work should continue." };
  }
  if (status === "failed") {
    return { headline: "This task is closed as failed", text: "Reopen this task to start the line again." };
  }
  const headline =
    closerHeadline(SPLIT_RUN_COMPLETED_HEADLINE_ACTOR, closer?.actor, closer?.automationName) ||
    SPLIT_RUN_COMPLETED_HEADLINE;
  return {
    headline,
    text: SPLIT_RUN_COMPLETED_NOTE_TEXT,
    ...(closer?.actor ? { actor: closer.actor } : {}),
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
 * waiting and failed keep To Backlog with Reject, Approve, or Rerun.
 * Draft keeps Reject and Start. Closed failed keeps Reopen. Completed
 * and rejected explain the result only.
 */
type FooterInput = {
  kind: SplitRunFooterKind;
  note?: WorkOrderStatusNotePresentation;
  doneSummary?: string;
  /** When false, hide the decision strip. Used while a follow-up run is active. */
  decision?: boolean;
  run?: { appId: string; runId: string };
  status?: WorkOrderDisplayStatus;
  actor?: OrgUserDisplay;
  automationName?: string;
};

function withFooterMeta(input: FooterInput, footer: SplitRunFooter): SplitRunFooter {
  const next = input.status ? { ...footer, status: input.status } : footer;
  return input.run ? { ...next, run: input.run } : next;
}

function hiddenDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  return withFooterMeta(input, {
    kind: input.kind,
    sentence: input.kind === "running" ? "This task is running." : "This task needs attention.",
    note,
    actions: [],
  });
}

function draftDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  return withFooterMeta(input, {
    kind: "draft",
    sentence: "This task is a draft.",
    note,
    attentionCard: true,
    actions: [REJECT, START],
  });
}

function closedFooterNote(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooterNote {
  if (input.status === "failed" || input.kind === "failed") {
    return closedDecisionNote(input.status ?? "failed");
  }
  if (input.status === "completed" || input.status === "rejected") {
    return closedDecisionNote(input.status, {
      actor: input.actor,
      automationName: input.automationName,
    });
  }
  return (
    note ??
    closedDecisionNote(input.status, {
      actor: input.actor,
      automationName: input.automationName,
    })
  );
}

function closedDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  const closedNote = closedFooterNote(input, note);
  if (input.kind === "failed") {
    return withFooterMeta(input, {
      kind: "failed",
      sentence: "This task needs attention.",
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

function stoppedNote(note: SplitRunFooterNote | undefined, actor?: OrgUserDisplay): SplitRunFooterNote {
  return {
    headline: actor ? SPLIT_RUN_STOPPED_HEADLINE : SPLIT_RUN_STOPPED_HEADLINE_UNKNOWN,
    text: note?.text ?? SPLIT_RUN_STOPPED_NOTE_TEXT,
    ...(actor ? { actor } : {}),
  };
}

function stoppedDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  return withFooterMeta(input, {
    kind: "stopped",
    sentence: "This task needs attention.",
    note: stoppedNote(note, input.actor),
    attentionCard: true,
    actions: [BACK_TO_DRAFT, REJECT, RERUN],
  });
}

function openDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  const actions = input.kind === "failed" ? [BACK_TO_DRAFT, REJECT, RERUN] : [BACK_TO_DRAFT, REJECT, APPROVE];
  return withFooterMeta(input, {
    kind: input.kind,
    sentence: "This task needs attention.",
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
  if (input.kind === "stopped") {
    return stoppedDecisionFooter(input, note);
  }
  if (input.kind === "waiting" || input.kind === "failed") {
    return openDecisionFooter(input, note);
  }
  return closedDecisionFooter({ ...input, kind: "done" }, note);
}

export function doneFooterForStatus(
  status: WorkOrderDisplayStatus,
  closer?: { actor?: OrgUserDisplay; automationName?: string },
): SplitRunFooter {
  return buildSplitRunFooter({
    kind: "done",
    doneSummary: getWorkOrderDisplayStatusMeta(status).summary,
    status,
    actor: closer?.actor,
    automationName: closer?.automationName,
  });
}
