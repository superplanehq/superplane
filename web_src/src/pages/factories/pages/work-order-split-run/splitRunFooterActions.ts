import type { SplitRunFooterKind } from "./splitRunFooter";

export type SplitRunFooterActionKind =
  | "start"
  | "archive"
  | "reject"
  | "refine"
  | "approve"
  | "continue-fixes"
  | "rerun"
  | "reopen"
  | "send-to-backlog";

export interface SplitRunFooterAction {
  id: string;
  kind: SplitRunFooterActionKind;
  label: string;
  emphasis: "primary" | "quiet";
  tooltip?: string;
  disabled?: boolean;
}

/** The paused automatic-fixes activity that Continue fixes resumes. */
export interface SplitRunContinueFixesTarget {
  pullRequestId: string;
  runId: string;
}

export const REJECT: SplitRunFooterAction = { id: "reject", kind: "reject", label: "Reject", emphasis: "quiet" };
export const ARCHIVE: SplitRunFooterAction = { id: "archive", kind: "archive", label: "Archive", emphasis: "quiet" };
const APPROVE: SplitRunFooterAction = { id: "approve", kind: "approve", label: "Approve", emphasis: "primary" };
const APPROVE_SECONDARY: SplitRunFooterAction = { ...APPROVE, emphasis: "quiet" };
const CONTINUE_FIXES: SplitRunFooterAction = {
  id: "continue-fixes",
  kind: "continue-fixes",
  label: "Continue fixes",
  emphasis: "primary",
};
export const RERUN: SplitRunFooterAction = { id: "rerun", kind: "rerun", label: "Rerun", emphasis: "primary" };
export const START: SplitRunFooterAction = { id: "start", kind: "start", label: "Start", emphasis: "primary" };
export const REOPEN: SplitRunFooterAction = { id: "reopen", kind: "reopen", label: "Reopen", emphasis: "primary" };
export const SEND_TO_BACKLOG: SplitRunFooterAction = {
  id: "send-to-backlog",
  kind: "send-to-backlog",
  label: "Send to backlog",
  emphasis: "quiet",
};

/** Actions on the open decision strip. Failed tasks rerun; paused fixes offer Continue fixes first. */
export function openDecisionActions(input: {
  kind: SplitRunFooterKind;
  continueFixes?: SplitRunContinueFixesTarget;
}): SplitRunFooterAction[] {
  if (input.kind === "failed") {
    return [REJECT, RERUN];
  }
  if (input.continueFixes) {
    return [CONTINUE_FIXES, REJECT, APPROVE_SECONDARY];
  }
  return [REJECT, APPROVE];
}
