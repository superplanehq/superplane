import { describe, expect, it } from "vitest";

import { billingBusinessPlanAction, billingUsagePercentUsed } from "./billingPlans";

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
