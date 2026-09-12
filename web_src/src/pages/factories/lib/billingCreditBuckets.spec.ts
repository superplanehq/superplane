import { describe, expect, it } from "bun:test";

import {
  BILLING_SPEND_ORDER_COPY,
  BILLING_SPEND_ORDER_WITH_GRANT_COPY,
  BILLING_TRIAL_TTL_COPY,
  adminCreditGrantCents,
  billingCreditBucketsView,
  billingCreditRemainingShares,
  billingSpendOrderCopy,
} from "./billingCreditBuckets";

describe("billingCreditBucketsView", () => {
  const now = new Date("2026-09-10T12:00:00.000Z");
  const trialEndsAt = "2026-09-24T12:00:00.000Z";
  const currentPeriodEnd = "2026-10-10T12:00:00.000Z";

  it("lists trial, included usage, and top-up in spend order", () => {
    const buckets = billingCreditBucketsView({
      welcomeRemainingCents: 4124,
      includedRemainingCents: 5000,
      purchasedRemainingCents: 5000,
      purchasedCents: 5000,
      plan: "business",
      trialEndsAt,
      currentPeriodEnd,
      now,
    });

    expect(buckets.map((bucket) => bucket.key)).toEqual(["trial", "included", "topup"]);
    expect(buckets[0]).toMatchObject({
      heading: "Trial credit",
      spendOrderLabel: "Spend first",
      remainingLabel: "$41.24 remaining",
      usedPercent: 18,
      footer: `Expires on ${new Date(trialEndsAt).toLocaleDateString()}. ${BILLING_TRIAL_TTL_COPY}`,
    });
    expect(buckets[1]).toMatchObject({
      heading: "Included usage",
      spendOrderLabel: "Spend next",
      remainingLabel: "$50.00 remaining",
      usedPercent: 0,
      footer: `Resets ${new Date(currentPeriodEnd).toLocaleDateString()}`,
    });
    expect(buckets[2]).toMatchObject({
      heading: "Top-up credit",
      spendOrderLabel: "Spend last",
      remainingLabel: "$50.00 remaining",
      usedPercent: 0,
      footer: null,
    });
  });

  it("says trial credit expired 14 days after registration", () => {
    const expiredAt = "2026-09-10T12:00:00.000Z";
    const [trial] = billingCreditBucketsView({
      welcomeRemainingCents: 0,
      includedRemainingCents: 5000,
      purchasedRemainingCents: 5000,
      purchasedCents: 5000,
      plan: "business",
      trialEndsAt: expiredAt,
      now,
    });

    expect(trial.footer).toBe(`Expired on ${new Date(expiredAt).toLocaleDateString()}. ${BILLING_TRIAL_TTL_COPY}`);
    expect(trial.usedPercent).toBe(100);
  });

  it("keeps included usage at zero until Business is active", () => {
    const [, included] = billingCreditBucketsView({
      welcomeRemainingCents: 4124,
      includedRemainingCents: 0,
      purchasedRemainingCents: 0,
      purchasedCents: 0,
      plan: "trial",
      trialEndsAt,
      now,
    });

    expect(included.remainingLabel).toBe("$0.00 remaining");
    expect(included.footer).toBe("Included with Business.");
  });

  it("shows SuperPlane grant after top-up when remaining grant is greater than zero", () => {
    const buckets = billingCreditBucketsView({
      welcomeRemainingCents: 0,
      includedRemainingCents: 0,
      purchasedRemainingCents: 104278,
      purchasedCents: 104278,
      adminRemainingCents: 11000,
      adminGrantCents: 11000,
    });

    expect(buckets.map((bucket) => bucket.key)).toEqual(["trial", "included", "topup", "grant"]);
    expect(buckets[2]).toMatchObject({
      heading: "Top-up credit",
      spendOrderLabel: "Spend third",
      remainingLabel: "$1042.78 remaining",
    });
    expect(buckets[3]).toMatchObject({
      heading: "SuperPlane grant",
      spendOrderLabel: "Spend last",
      remainingLabel: "$110.00 remaining",
      usedPercent: 0,
      footer: null,
    });
    expect(buckets.reduce((sum, bucket) => sum + bucket.remainingCents, 0)).toBe(115278);
  });

  it("uses SuperPlane grant amounts as the used-percent denominator", () => {
    const [, , , grant] = billingCreditBucketsView({
      welcomeRemainingCents: 0,
      includedRemainingCents: 0,
      purchasedRemainingCents: 0,
      purchasedCents: 0,
      adminRemainingCents: 5500,
      adminGrantCents: 11000,
    });

    expect(grant).toMatchObject({
      key: "grant",
      usedPercent: 50,
      remainingLabel: "$55.00 remaining",
    });
  });

  it("does not show SuperPlane grant when remaining grant is zero", () => {
    const buckets = billingCreditBucketsView({
      welcomeRemainingCents: 4124,
      includedRemainingCents: 0,
      purchasedRemainingCents: 0,
      purchasedCents: 0,
      adminRemainingCents: 0,
      adminGrantCents: 1500,
    });

    expect(buckets.map((bucket) => bucket.key)).toEqual(["trial", "included", "topup"]);
    expect(buckets[2].spendOrderLabel).toBe("Spend last");
    expect(buckets.some((bucket) => bucket.heading === "SuperPlane grant")).toBe(false);
  });
});

describe("billingCreditRemainingShares", () => {
  it("sizes the remaining bar by spend-order remaining amounts", () => {
    const buckets = billingCreditBucketsView({
      welcomeRemainingCents: 2500,
      includedRemainingCents: 2500,
      purchasedRemainingCents: 5000,
      purchasedCents: 5000,
    });

    expect(billingCreditRemainingShares(buckets)).toEqual([
      { key: "trial", percent: 25 },
      { key: "included", percent: 25 },
      { key: "topup", percent: 50 },
    ]);
  });

  it("returns empty shares when no credit remains", () => {
    const buckets = billingCreditBucketsView({
      welcomeRemainingCents: 0,
      includedRemainingCents: 0,
      purchasedRemainingCents: 0,
      purchasedCents: 0,
    });

    expect(billingCreditRemainingShares(buckets)).toEqual([
      { key: "trial", percent: 0 },
      { key: "included", percent: 0 },
      { key: "topup", percent: 0 },
    ]);
  });

  it("includes SuperPlane grant in remaining bar shares", () => {
    const buckets = billingCreditBucketsView({
      welcomeRemainingCents: 0,
      includedRemainingCents: 0,
      purchasedRemainingCents: 104278,
      purchasedCents: 104278,
      adminRemainingCents: 11000,
      adminGrantCents: 11000,
    });

    expect(billingCreditRemainingShares(buckets)).toEqual([
      { key: "trial", percent: 0 },
      { key: "included", percent: 0 },
      { key: "topup", percent: 90 },
      { key: "grant", percent: 10 },
    ]);
  });
});

describe("BILLING_SPEND_ORDER_COPY", () => {
  it("names the spend order in one sentence", () => {
    expect(BILLING_SPEND_ORDER_COPY).toBe(
      "Hosted runs spend trial credit first, then included usage, then top-up credit.",
    );
  });

  it("names SuperPlane grant only when that row is visible", () => {
    expect(billingSpendOrderCopy(false)).toBe(BILLING_SPEND_ORDER_COPY);
    expect(billingSpendOrderCopy(true)).toBe(BILLING_SPEND_ORDER_WITH_GRANT_COPY);
  });
});

describe("adminCreditGrantCents", () => {
  it("sums kind admin grant amounts", () => {
    expect(
      adminCreditGrantCents([
        { kind: "admin", amountCents: "11000" },
        { kind: "welcome", amountCents: "5000" },
        { kind: "admin", amountCents: "2500" },
      ]),
    ).toBe(13500);
  });
});
