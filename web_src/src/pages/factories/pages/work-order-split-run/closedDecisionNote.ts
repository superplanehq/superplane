import type { OrgUserDisplay } from "@/lib/orgUserDisplay";

import type { WorkOrderDisplayStatus } from "../../lib/workOrderProgress";

export const SPLIT_RUN_COMPLETED_NOTE_TEXT = "The work is done. The result met the goal.";
export const SPLIT_RUN_REJECTED_NOTE_TEXT = "The work is done. The result did not meet the goal.";
export const SPLIT_RUN_COMPLETED_HEADLINE = "This task succeeded";
export const SPLIT_RUN_REJECTED_HEADLINE = "This task did not succeed";
export const SPLIT_RUN_COMPLETED_HEADLINE_ACTOR = "marked this task as successful";
export const SPLIT_RUN_REJECTED_HEADLINE_ACTOR = "marked this task as unsuccessful";

const SPLIT_RUN_ARCHIVED_HEADLINE = "This task is archived";
const SPLIT_RUN_ARCHIVED_HEADLINE_ACTOR = "archived this task";
const SPLIT_RUN_ARCHIVED_NOTE_TEXT = "Send this task to Backlog to work on it again.";

type ClosedDecisionCloser = { actor?: OrgUserDisplay; automationName?: string };

export type ClosedDecisionNote = {
  headline: string;
  text: string;
  actor?: OrgUserDisplay;
};

function closerHeadline(verb: string, actor?: OrgUserDisplay, automationName?: string): string {
  if (actor) {
    return verb;
  }
  if (automationName) {
    return `${automationName} ${verb}`;
  }
  return "";
}

function closerDecisionNote(
  verb: string,
  fallback: string,
  text: string,
  closer?: ClosedDecisionCloser,
): ClosedDecisionNote {
  const headline = closerHeadline(verb, closer?.actor, closer?.automationName) || fallback;
  return {
    headline,
    text,
    ...(closer?.actor ? { actor: closer.actor } : {}),
  };
}

export function closedDecisionNote(status?: WorkOrderDisplayStatus, closer?: ClosedDecisionCloser): ClosedDecisionNote {
  if (status === "rejected") {
    return closerDecisionNote(
      SPLIT_RUN_REJECTED_HEADLINE_ACTOR,
      SPLIT_RUN_REJECTED_HEADLINE,
      SPLIT_RUN_REJECTED_NOTE_TEXT,
      closer,
    );
  }
  if (status === "archived") {
    return closerDecisionNote(
      SPLIT_RUN_ARCHIVED_HEADLINE_ACTOR,
      SPLIT_RUN_ARCHIVED_HEADLINE,
      SPLIT_RUN_ARCHIVED_NOTE_TEXT,
      closer,
    );
  }
  if (status === "failed") {
    return { headline: "This task is closed as failed", text: "Reopen this task to start the line again." };
  }
  return closerDecisionNote(
    SPLIT_RUN_COMPLETED_HEADLINE_ACTOR,
    SPLIT_RUN_COMPLETED_HEADLINE,
    SPLIT_RUN_COMPLETED_NOTE_TEXT,
    closer,
  );
}
