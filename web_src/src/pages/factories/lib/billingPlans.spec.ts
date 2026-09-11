import { describe, expect, it } from "vitest";

import {
  billingBusinessPlanAction,
  billingCanCancelBusiness,
  billingCanKeepBusiness,
  billingCancelBusinessConfirmCopy,
  billingSubscriptionEndsCopy,
  billingUsagePercentUsed,
} from "./billingPlans";

describe("billingBusinessPlanAction", () => {
  it("shows the upgrade action when the organization cannot buy credit", () => {
    expect(
      billingBusinessPlanAction({
        canManageBilling: true,
        creditPurchaseAllowed: false,
      }),
    ).toBe("subscribe");
  });

  it("shows Current plan when credit purchase is allowed", () => {
    expect(
      billingBusinessPlanAction({
        canManageBilling: true,
        creditPurchaseAllowed: true,
      }),
    ).toBe("current");
  });

  it("shows ending when Business is set to cancel at period end", () => {
    expect(
      billingBusinessPlanAction({
        canManageBilling: true,
        creditPurchaseAllowed: true,
        cancelAtPeriodEnd: true,
      }),
    ).toBe("ending");
  });

  it("hides the upgrade action for members who cannot manage billing", () => {
    expect(
      billingBusinessPlanAction({
        canManageBilling: false,
        creditPurchaseAllowed: false,
      }),
    ).toBe("hidden");
  });

  it("keeps Current plan for members when the organization is on Business", () => {
    expect(
      billingBusinessPlanAction({
        canManageBilling: false,
        creditPurchaseAllowed: true,
      }),
    ).toBe("current");
  });

  it("shows ending for members when Business is set to cancel at period end", () => {
    expect(
      billingBusinessPlanAction({
        canManageBilling: false,
        creditPurchaseAllowed: true,
        cancelAtPeriodEnd: true,
      }),
    ).toBe("ending");
  });
});

describe("billingCanCancelBusiness", () => {
  it("shows cancel only for Polar Business that is not already ending", () => {
    expect(
      billingCanCancelBusiness({
        canManageBilling: true,
        creditPurchaseAllowed: true,
        planSource: "polar",
      }),
    ).toBe(true);
  });

  it("hides cancel for admin Business", () => {
    expect(
      billingCanCancelBusiness({
        canManageBilling: true,
        creditPurchaseAllowed: true,
        planSource: "admin",
      }),
    ).toBe(false);
  });

  it("hides cancel for members", () => {
    expect(
      billingCanCancelBusiness({
        canManageBilling: false,
        creditPurchaseAllowed: true,
        planSource: "polar",
      }),
    ).toBe(false);
  });

  it("hides cancel when Business is already ending", () => {
    expect(
      billingCanCancelBusiness({
        canManageBilling: true,
        creditPurchaseAllowed: true,
        planSource: "polar",
        cancelAtPeriodEnd: true,
      }),
    ).toBe(false);
  });
});

describe("billingCanKeepBusiness", () => {
  it("shows Keep Business when Polar Business is ending", () => {
    expect(
      billingCanKeepBusiness({
        canManageBilling: true,
        creditPurchaseAllowed: true,
        planSource: "polar",
        cancelAtPeriodEnd: true,
      }),
    ).toBe(true);
  });

  it("hides Keep Business when Business is not ending", () => {
    expect(
      billingCanKeepBusiness({
        canManageBilling: true,
        creditPurchaseAllowed: true,
        planSource: "polar",
      }),
    ).toBe(false);
  });
});

describe("billingSubscriptionEndsCopy", () => {
  it("names the period end date", () => {
    const periodEnd = "2026-10-09T12:00:00.000Z";
    expect(billingSubscriptionEndsCopy(periodEnd)).toBe(
      `Business ends on ${new Date(periodEnd).toLocaleDateString()}. Hosted runs continue until that date.`,
    );
  });
});

describe("billingCancelBusinessConfirmCopy", () => {
  it("names the period end date in the confirm text", () => {
    const periodEnd = "2026-10-09T12:00:00.000Z";
    expect(billingCancelBusinessConfirmCopy(periodEnd)).toBe(
      `Business stays active until ${new Date(periodEnd).toLocaleDateString()}. SuperPlane will not renew after that date.`,
    );
  });
});

describe("billingUsagePercentUsed", () => {
  it("rounds used credit against the allowance", () => {
    expect(billingUsagePercentUsed(4124, 5000)).toBe(18);
  });

  it("returns 0 when nothing is used", () => {
    expect(billingUsagePercentUsed(5000, 5000)).toBe(0);
  });

  it("returns 100 when remaining credit is empty", () => {
    expect(billingUsagePercentUsed(0, 5000)).toBe(100);
  });
});
