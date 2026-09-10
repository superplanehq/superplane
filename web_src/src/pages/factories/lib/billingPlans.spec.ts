import { describe, expect, it } from "vitest";

import { billingBusinessPlanAction, billingPlansUsageView, billingUsagePercentUsed } from "./billingPlans";

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

describe("billingPlansUsageView", () => {
  const now = new Date("2026-09-10T12:00:00.000Z");
  const trialEndsAt = "2026-09-22T12:00:00.000Z";

  it("shows remaining trial credit and the trial end date", () => {
    expect(
      billingPlansUsageView({
        plan: "trial",
        remainingCents: 4124,
        grantTotalCents: 5000,
        includedRemainingCents: 0,
        purchasedCents: 0,
        trialEndsAt,
        now,
      }),
    ).toEqual({
      heading: "Your trial usage",
      remainingLabel: "$41.24 remaining",
      usedPercent: 18,
      footer: `Ends ${new Date(trialEndsAt).toLocaleDateString()}`,
    });
  });

  it("says the trial has ended after the trial date", () => {
    expect(
      billingPlansUsageView({
        plan: "trial",
        remainingCents: 0,
        grantTotalCents: 5000,
        includedRemainingCents: 0,
        purchasedCents: 0,
        trialEndsAt: "2026-08-15T12:00:00.000Z",
        now,
      }).footer,
    ).toBe("The trial has ended.");
  });

  it("shows included Business usage and the reset date", () => {
    const currentPeriodEnd = "2026-10-09T12:00:00.000Z";
    expect(
      billingPlansUsageView({
        plan: "business",
        remainingCents: 14124,
        grantTotalCents: 15000,
        includedRemainingCents: 5000,
        purchasedCents: 10000,
        currentPeriodEnd,
        now,
      }),
    ).toEqual({
      heading: "Your included usage",
      remainingLabel: "$50.00 remaining",
      usedPercent: 0,
      footer: `Resets ${new Date(currentPeriodEnd).toLocaleDateString()}`,
    });
  });
});
