import type { OrgUserDisplay } from "@/lib/orgUserDisplay";

import { DRAFT_READINESS_NOTES, draftReadiness, type DraftReadinessTone } from "../../lib/draftReadiness";
import { getWorkOrderDisplayStatusMeta, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import type { WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";
export type SplitRunFooterKind = "draft" | "running" | "waiting" | "failed" | "stopped" | "done";

/** @deprecated Use SplitRunFooterKind. Kept for fixture field name. */
export type SplitRunFooterTone = SplitRunFooterKind;

export type SplitRunFooterActionKind =
  | "start"
  | "archive"
  | "reject"
  | "refine"
  | "approve"
  | "rerun"
  | "reopen"
  | "send-to-backlog";

export type SplitRunStopChoice = "canceled" | "completed" | "rerun-step" | "rerun-start" | "reopen";

export const DEFAULT_SPLIT_RUN_STOP_CHOICE: SplitRunStopChoice = "canceled";

export type SplitRunStopChoiceItem = {
  id: SplitRunStopChoice;
  label: string;
  actionLabel: string;
  description: string;
  status: "rejected" | "completed" | "draft" | "running";
};

export const SPLIT_RUN_STOP_CHOICES: SplitRunStopChoiceItem[] = [
  {
    id: "canceled",
    label: "Stop and Close",
    actionLabel: "Stop and Close",
    description: "Marks this task as Rejected",
    status: "rejected",
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
  return status === "completed" || status === "failed" || status === "rejected";
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
    return status !== "rejected";
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
  tooltip?: string;
  disabled?: boolean;
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
  /** Draft Clarity. 0–5. Missing while analysis has not scored yet. */
  clarityScore?: number;
  /** Draft Confidence. 0–5. Missing while analysis has not scored yet. */
  confidenceScore?: number;
}

export function isTaskResultFooter(footer: SplitRunFooter): boolean {
  return footer.status === "completed" || footer.status === "rejected";
}

const REJECT: SplitRunFooterAction = { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" };
const ARCHIVE: SplitRunFooterAction = { id: "archive", kind: "archive", label: "Archive", emphasis: "quiet" };
const APPROVE: SplitRunFooterAction = { id: "approve", kind: "approve", label: "Approve", emphasis: "primary" };
const RERUN: SplitRunFooterAction = { id: "rerun", kind: "rerun", label: "Rerun", emphasis: "primary" };
const START: SplitRunFooterAction = { id: "start", kind: "start", label: "Start", emphasis: "primary" };
const REOPEN: SplitRunFooterAction = { id: "reopen", kind: "reopen", label: "Reopen", emphasis: "primary" };
const SEND_TO_BACKLOG: SplitRunFooterAction = {
  id: "send-to-backlog",
  kind: "send-to-backlog",
  label: "Send to backlog",
  emphasis: "quiet",
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

export const SPLIT_RUN_DRAFT_NOTE: SplitRunFooterNote = { ...DRAFT_READINESS_NOTES.pending };

export const SPLIT_RUN_CLASSIC_DRAFT_NOTE: SplitRunFooterNote = {
  headline: "This task is ready to start",
  text: "Review the details. Change anything you need. Then click Start to send it to the line.",
};

/** Restore the established draft controls outside live refinement mode. */
export function classicSplitRunFooter(footer: SplitRunFooter): SplitRunFooter {
  if (footer.kind !== "draft") {
    return footer;
  }
  return {
    ...footer,
    sentence: "This task is a draft.",
    note: { ...SPLIT_RUN_CLASSIC_DRAFT_NOTE },
    clarityScore: undefined,
    confidenceScore: undefined,
    actions: [ARCHIVE, START],
  };
}

export const SPLIT_RUN_ANALYZING_NOTE: SplitRunFooterNote = { ...DRAFT_READINESS_NOTES.analyzing };
export const SPLIT_RUN_DRAFT_BLOCKED_NOTE: SplitRunFooterNote = { ...DRAFT_READINESS_NOTES.unclear };
export const SPLIT_RUN_DRAFT_AGENT_FIT_NOTE: SplitRunFooterNote = { ...DRAFT_READINESS_NOTES.agentFit };
export const SPLIT_RUN_DRAFT_CAUTION_NOTE: SplitRunFooterNote = { ...DRAFT_READINESS_NOTES.uncertain };

export type SplitRunDecisionTone =
  | "draft"
  | "draft-blocked"
  | "draft-caution"
  | "draft-ready"
  | "waiting"
  | "failed"
  | "done"
  | "rejected";

/** Both draft scores in the shape `draftReadiness` and `startConfirm` read. */
export function splitRunFooterScores(footer: Pick<SplitRunFooter, "clarityScore" | "confidenceScore">) {
  return { clarity: footer.clarityScore, confidence: footer.confidenceScore };
}

export function splitRunDecisionTone(footer: SplitRunFooter): SplitRunDecisionTone {
  if (footer.kind === "draft") {
    return draftDecisionTone(draftReadiness(splitRunFooterScores(footer)).tone);
  }
  if (footer.kind === "waiting") {
    return "waiting";
  }
  if (footer.kind === "failed" || footer.status === "failed") {
    return "failed";
  }
  if (footer.kind === "stopped" || footer.status === "rejected") {
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
  if (status === "failed" || status === "rejected") {
    return [SEND_TO_BACKLOG, REOPEN];
  }
  return [];
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
 * waiting shows Reject and Approve. Failed and stopped show Reject and Rerun.
 * Failed and rejected keep Send to backlog and Reopen. Completed explains
 * the result only.
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
  /** True while the Backlog automation still scores this draft. */
  isAnalyzing?: boolean;
  clarityScore?: number;
  confidenceScore?: number;
};

function withFooterMeta(input: FooterInput, footer: SplitRunFooter): SplitRunFooter {
  const next = input.status ? { ...footer, status: input.status } : footer;
  const withRun = input.run ? { ...next, run: input.run } : next;
  const withClarity = input.clarityScore == null ? withRun : { ...withRun, clarityScore: input.clarityScore };
  return input.confidenceScore == null ? withClarity : { ...withClarity, confidenceScore: input.confidenceScore };
}

const DRAFT_TONE: Record<DraftReadinessTone, SplitRunDecisionTone> = {
  analyzing: "draft",
  pending: "draft",
  blocked: "draft-blocked",
  caution: "draft-caution",
  ready: "draft-ready",
};

function draftDecisionTone(tone: DraftReadinessTone): SplitRunDecisionTone {
  return DRAFT_TONE[tone];
}

function draftReadinessNote(input: FooterInput): SplitRunFooterNote {
  const readiness = draftReadiness({ ...splitRunFooterScores(input), isAnalyzing: input.isAnalyzing });
  return { headline: readiness.headline, text: readiness.text };
}

function draftDecisionActions(): SplitRunFooterAction[] {
  return [ARCHIVE, START];
}

function hiddenDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  return withFooterMeta(input, {
    kind: input.kind,
    sentence: splitRunKindSentence(input.kind),
    note,
    actions: [],
  });
}

function draftDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  const hasScore = input.clarityScore != null || input.confidenceScore != null;
  const analyzing = Boolean(input.isAnalyzing) && !hasScore;
  return withFooterMeta(input, {
    kind: "draft",
    sentence: analyzing ? "SuperPlane is analyzing this task." : "This task is a draft.",
    note: analyzing || hasScore ? draftReadinessNote(input) : (note ?? { ...SPLIT_RUN_DRAFT_NOTE }),
    attentionCard: true,
    actions: draftDecisionActions(),
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
      sentence: splitRunKindSentence("failed"),
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
    sentence: splitRunKindSentence("stopped"),
    note: stoppedNote(note, input.actor),
    attentionCard: true,
    actions: [REJECT, RERUN],
  });
}

function openDecisionFooter(input: FooterInput, note?: SplitRunFooterNote): SplitRunFooter {
  if (input.kind === "waiting" && !note) {
    return hiddenDecisionFooter(input);
  }

  const actions = input.kind === "failed" ? [REJECT, RERUN] : [REJECT, APPROVE];
  return withFooterMeta(input, {
    kind: input.kind,
    sentence: splitRunKindSentence(input.kind),
    note,
    attentionCard: true,
    actions,
  });
}

function splitRunKindSentence(kind: FooterInput["kind"]): string {
  if (kind === "running") {
    return "This task is running.";
  }
  if (kind === "failed") {
    return "This task failed.";
  }
  if (kind === "stopped") {
    return "This task stopped.";
  }
  return "This task is waiting.";
}

export function buildSplitRunFooter(input: FooterInput): SplitRunFooter {
  return buildSplitRunDecisionFooter(input);
}

function buildSplitRunDecisionFooter(input: FooterInput): SplitRunFooter {
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
