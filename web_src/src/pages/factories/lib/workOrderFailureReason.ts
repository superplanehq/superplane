import type { FactoriesWorkOrderExecution } from "@/api-client";

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

/** Result messages the runner stores when hosted credit blocks a node. */
const CREDIT_FAILURE_MESSAGE: Record<string, WorkOrderCreditFailureReason> = {
  "This organization has no hosted credit.": "no_hosted_credit",
  "Subscribe to Business to keep SuperPlane-hosted runs.": "hosted_subscription_required",
  "This workspace has no remaining hosted credit.": "workspace_budget_empty",
};

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

/** Copy for a node result message. Other messages return null. */
export function workOrderCreditFailureFromMessage(message: string | undefined): WorkOrderCreditFailureCopy | null {
  const reason = message ? CREDIT_FAILURE_MESSAGE[message.trim()] : undefined;
  return reason ? CREDIT_FAILURE_COPY[reason] : null;
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
