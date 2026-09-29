import { describe, expect, it } from "bun:test";

import {
  failedStepCreditNote,
  hostedCreditBlockNote,
  hostedCreditBlocksDispatch,
  isOutOfHostedCredit,
  isWorkOrderCreditFailureReason,
  workOrderCreditFailureCopy,
  workOrderExecutionCreditFailure,
} from "./workOrderFailureReason";

describe("isOutOfHostedCredit", () => {
  it("is true when remaining credit is zero or negative", () => {
    expect(isOutOfHostedCredit({ remainingCreditCents: 0 })).toBe(true);
    expect(isOutOfHostedCredit({ remainingCreditCents: -1 })).toBe(true);
  });

  it("is false when credit remains or the balance is still loading", () => {
    expect(isOutOfHostedCredit({ remainingCreditCents: 1 })).toBe(false);
    expect(isOutOfHostedCredit({})).toBe(false);
    expect(isOutOfHostedCredit(undefined)).toBe(false);
  });
});

describe("hostedCreditBlocksDispatch", () => {
  const empty = { remainingCreditCents: 0 };

  it("blocks a hosted line and an unknown line when credit is empty", () => {
    expect(hostedCreditBlocksDispatch(empty, true)).toBe(true);
    expect(hostedCreditBlocksDispatch(empty, undefined)).toBe(true);
  });

  it("does not block a BYOK-only line when credit is empty", () => {
    expect(hostedCreditBlocksDispatch(empty, false)).toBe(false);
  });

  it("still blocks a hosted or mixed line when credit is empty", () => {
    expect(hostedCreditBlocksDispatch(empty, true)).toBe(true);
    expect(hostedCreditBlocksDispatch(empty, undefined)).toBe(true);
  });
});

describe("workOrderCreditFailureCopy", () => {
  it("maps no hosted credit to No credit and Add credits", () => {
    expect(workOrderCreditFailureCopy("no_hosted_credit")).toEqual({
      label: "No credit",
      message: "This agent run is blocked. The organization has no SuperPlane hosted credit.",
      actionLabel: "Add credits",
    });
  });

  it("maps a missing Business plan to No plan and Subscribe", () => {
    expect(workOrderCreditFailureCopy("hosted_subscription_required")).toEqual({
      label: "No plan",
      message: "This agent run is blocked. SuperPlane hosted runs need a Business plan.",
      actionLabel: "Subscribe",
    });
  });

  it("maps an empty workspace budget to No workspace budget", () => {
    expect(workOrderCreditFailureCopy("workspace_budget_empty")).toEqual({
      label: "No workspace budget",
      message: "This agent run is blocked. This workspace has no hosted credit budget left.",
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

describe("failedStepCreditNote", () => {
  const now = new Date("2026-09-28T12:00:00.000Z");
  const openTrial = {
    plan: "trial",
    remainingCreditCents: 0,
    trialEndsAt: "2026-10-12T12:00:00.000Z",
    billingHref: "/demo/organization/billing",
  };

  it("warns that an open trial with no credit cannot run tasks", () => {
    expect(failedStepCreditNote("no_hosted_credit", openTrial, now)).toEqual({
      headline: "No credit",
      text: "This organization is out of credit.",
      actionLabel: "Open billing",
      warning: true,
    });
  });

  it("does not label a missing or unrelated failure as a credit failure", () => {
    expect(failedStepCreditNote(undefined, openTrial, now)).toBeNull();
    expect(failedStepCreditNote("", openTrial, now)).toBeNull();
    expect(failedStepCreditNote("provider_error", openTrial, now)).toBeNull();
  });

  it("names missing credit when the organization is not on a trial", () => {
    expect(failedStepCreditNote("no_hosted_credit", { plan: "business", remainingCreditCents: 0 }, now)).toEqual({
      headline: "No credit",
      text: "This organization has no hosted credit. Add credit, then run this step again.",
      actionLabel: "Add credits",
      warning: false,
    });
  });

  it("warns on a draft when an open trial has no credit", () => {
    expect(hostedCreditBlockNote(openTrial, now)).toEqual({
      headline: "No credit",
      text: "This organization is out of credit.",
      actionLabel: "Open billing",
      href: "/demo/organization/billing",
      warning: true,
    });
  });

  it("asks for credit on a draft when the organization is not on a trial", () => {
    expect(hostedCreditBlockNote({ plan: "business", remainingCreditCents: 0, billingHref: "/billing" }, now)).toEqual({
      headline: "No credit",
      text: "This organization has no hosted credit. Add credit, then run this step again.",
      actionLabel: "Add credits",
      href: "/billing",
      warning: false,
    });
  });

  it("does not block a draft while credit is still loading", () => {
    expect(hostedCreditBlockNote({ plan: "trial" }, now)).toBeNull();
  });

  it("does not treat an expired trial as an open trial", () => {
    expect(
      failedStepCreditNote(
        "no_hosted_credit",
        { plan: "trial", remainingCreditCents: 0, trialEndsAt: "2026-09-01T12:00:00.000Z" },
        now,
      )?.warning,
    ).toBe(false);
  });
});
