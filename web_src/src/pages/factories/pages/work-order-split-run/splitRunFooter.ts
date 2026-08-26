import { getWorkOrderDisplayStatusMeta, type WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import type { WorkOrderStatusNotePresentation } from "../../lib/workOrderStatusNote";

export type SplitRunFooterKind = "draft" | "running" | "waiting" | "failed" | "done";

/** @deprecated Use SplitRunFooterKind. Kept for fixture field name. */
export type SplitRunFooterTone = SplitRunFooterKind;

export type SplitRunFooterActionKind = "start" | "reject" | "stop" | "reopen";

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
    label: "Stop as Canceled",
    actionLabel: "Stop & Close",
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

export function defaultSplitRunStopChoice(status?: WorkOrderDisplayStatus): SplitRunStopChoice | undefined {
  const available = availableSplitRunStopChoices(status);
  if (available.some((item) => item.id === DEFAULT_SPLIT_RUN_STOP_CHOICE)) {
    return DEFAULT_SPLIT_RUN_STOP_CHOICE;
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
  /** When true, the waiting or failed note renders as a sticky strip above Stop. */
  attentionCard?: boolean;
  run?: { appId: string; runId: string };
  status?: WorkOrderDisplayStatus;
}

const START: SplitRunFooterAction = { id: "start", kind: "start", label: "Start", emphasis: "primary" };
const REJECT: SplitRunFooterAction = { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" };
const STOP: SplitRunFooterAction = { id: "stop", kind: "stop", label: "Stop", emphasis: "quiet" };
const REOPEN: SplitRunFooterAction = { id: "reopen", kind: "reopen", label: "Reopen", emphasis: "primary" };

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
 * Reject and Start. Implement and Verify keep Stop. Closed orders keep
 * Reopen. A visible waiting note sits above Stop as an attention card.
 */
export function buildSplitRunFooter(input: {
  kind: SplitRunFooterKind;
  note?: WorkOrderStatusNotePresentation;
  doneSummary?: string;
  attentionCard?: boolean;
  run?: { appId: string; runId: string };
  status?: WorkOrderDisplayStatus;
}): SplitRunFooter {
  const note = input.note ? toFooterNote(input.note) : undefined;
  const attentionCard = input.attentionCard || undefined;
  const withStatus = (footer: SplitRunFooter): SplitRunFooter =>
    input.status ? { ...footer, status: input.status } : footer;
  if (input.kind === "draft") {
    return withStatus({ kind: "draft", sentence: "This work order is a draft.", note, actions: [REJECT, START] });
  }
  if (isClosedWorkOrderDisplayStatus(input.status) || input.kind === "done") {
    if (input.kind === "failed") {
      return withStatus({
        kind: "failed",
        sentence: "This work order needs attention.",
        note,
        attentionCard,
        run: input.run,
        actions: [REOPEN],
      });
    }
    return withStatus({
      kind: "done",
      sentence: input.doneSummary ?? getWorkOrderDisplayStatusMeta(input.status ?? "completed").summary,
      note,
      actions: [REOPEN],
    });
  }
  if (input.kind === "running") {
    return withStatus({
      kind: "running",
      sentence: "This work order is running.",
      note,
      run: input.run,
      actions: [STOP],
    });
  }
  if (input.kind === "waiting" || input.kind === "failed") {
    return withStatus({
      kind: input.kind,
      sentence: "This work order needs attention.",
      note,
      attentionCard,
      run: input.run,
      actions: [STOP],
    });
  }
  return withStatus({
    kind: "done",
    sentence: input.doneSummary ?? getWorkOrderDisplayStatusMeta("completed").summary,
    note,
    actions: [REOPEN],
  });
}

export function doneFooterForStatus(status: WorkOrderDisplayStatus): SplitRunFooter {
  return buildSplitRunFooter({
    kind: "done",
    doneSummary: getWorkOrderDisplayStatusMeta(status).summary,
    status,
  });
}
