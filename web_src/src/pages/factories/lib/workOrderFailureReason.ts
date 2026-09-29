import type { FactoriesWorkOrderExecution } from "@/api-client";

import { isHostedCreditTrialOrg, isWelcomeCreditExpired } from "./hostedCreditEmpty";

/** Failure reason codes the API sets when SuperPlane hosted credit blocks a step. */
export type WorkOrderCreditFailureReason =
  | "no_hosted_credit"
  | "hosted_subscription_required"
  | "workspace_budget_empty";

export interface WorkOrderCreditFailureCopy {
  /** Short text for the task card chip. */
  label: string;
  /** Sentence that tells the user why the agent run is blocked. */
  message: string;
  /** Text for the billing link. */
  actionLabel: string;
}

const CREDIT_FAILURE_COPY: Record<WorkOrderCreditFailureReason, WorkOrderCreditFailureCopy> = {
  no_hosted_credit: {
    label: "No credit",
    message: "This agent run is blocked. The organization has no SuperPlane hosted credit.",
    actionLabel: "Add credits",
  },
  hosted_subscription_required: {
    label: "No plan",
    message: "This agent run is blocked. SuperPlane hosted runs need a Business plan.",
    actionLabel: "Subscribe",
  },
  workspace_budget_empty: {
    label: "No workspace budget",
    message: "This agent run is blocked. This workspace has no hosted credit budget left.",
    actionLabel: "Open billing",
  },
};

export function isWorkOrderCreditFailureReason(value: string | undefined): value is WorkOrderCreditFailureReason {
  return value != null && Object.prototype.hasOwnProperty.call(CREDIT_FAILURE_COPY, value);
}

/** Copy for a SuperPlane hosted credit failure. Other reasons return null. */
export function workOrderCreditFailureCopy(reason: string | undefined): WorkOrderCreditFailureCopy | null {
  return isWorkOrderCreditFailureReason(reason) ? CREDIT_FAILURE_COPY[reason] : null;
}

/** Credit failure copy for a failed step. Steps that did not fail return null. */
export function workOrderExecutionCreditFailure(
  execution: Pick<FactoriesWorkOrderExecution, "result" | "failureReason"> | null | undefined,
): WorkOrderCreditFailureCopy | null {
  if (execution?.result !== "RESULT_FAILED") {
    return null;
  }
  return workOrderCreditFailureCopy(execution.failureReason);
}

/** Organization credit the task popup needs to explain a blocked run. */
export interface HostedCreditRunContext {
  plan?: string;
  remainingCreditCents?: number;
  trialEndsAt?: string;
  welcomeCreditExpiresAt?: string;
  billingHref?: string;
}

/** Tooltip on Start and Rerun when the organization has no hosted credit left. */
export const OUT_OF_CREDIT_ACTION_TOOLTIP = "This organization is out of credit.";

/**
 * True when remaining hosted credit is known and at or below zero.
 * Undefined remaining credit means the balance is still loading — do not block.
 * This does not mean the next run needs hosted credit. A BYOK line does not.
 */
export function isOutOfHostedCredit(credit?: HostedCreditRunContext): boolean {
  return credit?.remainingCreditCents != null && credit.remainingCreditCents <= 0;
}

/**
 * True when Start or Rerun would spend hosted credit and none remains.
 * A BYOK-only line does not spend hosted credit. A mixed line still needs
 * credit when any step uses a hosted runner, even if the draft model is BYOK.
 * Unknown runner funding keeps the credit block so a hosted line does not start early.
 */
export function hostedCreditBlocksDispatch(
  credit: HostedCreditRunContext | undefined,
  usesHostedRunner: boolean | undefined,
): boolean {
  if (usesHostedRunner === false) {
    return false;
  }
  return isOutOfHostedCredit(credit);
}

/** Warning or failure copy for a draft or a failed step that cannot run. */
export interface HostedCreditBlockNotice extends FailedStepCreditNote {
  href?: string;
}

/**
 * Copy when the organization cannot start or rerun a task.
 * An open trial with $0 warns and links to billing. Any other empty balance
 * asks the user to add credit. A balance that is still loading returns null.
 */
export function hostedCreditBlockNote(
  credit?: HostedCreditRunContext,
  now: Date = new Date(),
): HostedCreditBlockNotice | null {
  if (!isOutOfHostedCredit(credit)) {
    return null;
  }
  const note = isActiveTrialWithoutCredit(credit, now) ? TRIAL_EMPTY_NOTE : NO_CREDIT_NOTE;
  return credit?.billingHref ? { ...note, href: credit.billingHref } : note;
}

/** Copy and tone for a failed step that hosted credit blocked. */
export interface FailedStepCreditNote {
  headline: string;
  text: string;
  actionLabel: string;
  /** Warning strip. An open trial with no credit uses this instead of a failure. */
  warning: boolean;
}

const TRIAL_EMPTY_NOTE: FailedStepCreditNote = {
  headline: "No credit",
  text: "This organization is out of credit.",
  actionLabel: "Open billing",
  warning: true,
};

const NO_CREDIT_NOTE: FailedStepCreditNote = {
  headline: "No credit",
  text: "This organization has no hosted credit. Add credit, then run this step again.",
  actionLabel: "Add credits",
  warning: false,
};

/**
 * Note for a failed step when hosted credit or the plan blocked the run.
 * An open trial with $0 warns that the organization is out of credit and
 * links to billing. Other credit failures stay on the failure strip.
 * A missing reason is not a credit failure. Other errors return null.
 */
export function failedStepCreditNote(
  failureReason: string | undefined,
  credit?: HostedCreditRunContext,
  now: Date = new Date(),
): FailedStepCreditNote | null {
  if (isActiveTrialWithoutCredit(credit, now) && isTrialCreditBlock(failureReason)) {
    return TRIAL_EMPTY_NOTE;
  }

  if (failureReason === "no_hosted_credit") {
    return NO_CREDIT_NOTE;
  }
  if (failureReason === "hosted_subscription_required") {
    return {
      headline: "No plan",
      text: "You cannot run tasks until you subscribe.",
      actionLabel: "Subscribe",
      warning: true,
    };
  }
  if (failureReason === "workspace_budget_empty") {
    return {
      headline: "No workspace budget",
      text: "This agent run is blocked. This workspace has no hosted credit budget left.",
      actionLabel: "Open billing",
      warning: false,
    };
  }
  return null;
}

function isTrialCreditBlock(failureReason: string | undefined): boolean {
  return failureReason === "no_hosted_credit" || failureReason === "hosted_subscription_required";
}

function isActiveTrialWithoutCredit(credit: HostedCreditRunContext | undefined, now: Date): boolean {
  if (!credit || credit.remainingCreditCents == null || credit.remainingCreditCents > 0) {
    return false;
  }
  if (
    !isHostedCreditTrialOrg({
      plan: credit.plan,
      trialEndsAt: credit.trialEndsAt,
      welcomeCreditExpiresAt: credit.welcomeCreditExpiresAt,
    })
  ) {
    return false;
  }
  return !isWelcomeCreditExpired(credit.trialEndsAt ?? credit.welcomeCreditExpiresAt, now);
}
