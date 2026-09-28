import { describe, expect, it } from "bun:test";

import {
  isWorkOrderCreditFailureReason,
  workOrderCreditFailureCopy,
  workOrderExecutionCreditFailure,
} from "./workOrderFailureReason";

describe("workOrderCreditFailureCopy", () => {
  it("maps no hosted credit to No credit and Add credits", () => {
    expect(workOrderCreditFailureCopy("no_hosted_credit")).toEqual({
      label: "No credit",
      message: "This step did not start. The organization has no SuperPlane hosted credit.",
      actionLabel: "Add credits",
    });
  });

  it("maps a missing Business plan to No plan and Subscribe", () => {
    expect(workOrderCreditFailureCopy("hosted_subscription_required")).toEqual({
      label: "No plan",
      message: "This step did not start. SuperPlane hosted runs need a Business plan.",
      actionLabel: "Subscribe",
    });
  });

  it("maps an empty workspace budget to No workspace budget", () => {
    expect(workOrderCreditFailureCopy("workspace_budget_empty")).toEqual({
      label: "No workspace budget",
      message: "This step did not start. This workspace has no hosted credit budget left.",
      actionLabel: "Open billing",
    });
  });

  it("returns null for other reasons", () => {
    expect(workOrderCreditFailureCopy(undefined)).toBeNull();
    expect(workOrderCreditFailureCopy("")).toBeNull();
    expect(workOrderCreditFailureCopy("toString")).toBeNull();
    expect(isWorkOrderCreditFailureReason("provider_error")).toBe(false);
  });
});

describe("workOrderExecutionCreditFailure", () => {
  it("returns copy only for a failed step", () => {
    expect(workOrderExecutionCreditFailure({ result: "RESULT_FAILED", failureReason: "no_hosted_credit" })?.label).toBe(
      "No credit",
    );
    expect(workOrderExecutionCreditFailure({ result: "RESULT_PASSED", failureReason: "no_hosted_credit" })).toBeNull();
    expect(workOrderExecutionCreditFailure({ result: "RESULT_FAILED" })).toBeNull();
    expect(workOrderExecutionCreditFailure(null)).toBeNull();
  });
});
