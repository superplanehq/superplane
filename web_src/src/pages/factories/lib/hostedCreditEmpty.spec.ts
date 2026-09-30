import { describe, expect, it } from "bun:test";

import {
  hostedCreditBannerKind,
  hostedCreditBillingBalanceCopy,
  hostedCreditHeaderKickerActionLabel,
  hostedCreditHeaderKickerLabel,
  isHostedCreditHeaderKickerKind,
  isHostedCreditTrialOrg,
  organizationPlanLabel,
  shouldShowHostedCreditEmptyBanner,
  welcomeCreditHeaderLabel,
} from "./hostedCreditEmpty";

const now = new Date("2026-09-08T12:00:00.000Z");
const inFourteenDays = "2026-09-22T12:00:00.000Z";
const yesterday = "2026-09-07T12:00:00.000Z";

describe("isHostedCreditTrialOrg", () => {
  it("is true when welcome credit has an expiry and the organization has not purchased credit", () => {
    expect(
      isHostedCreditTrialOrg({
        purchasedCreditCents: "0",
        welcomeCreditExpiresAt: inFourteenDays,
      }),
    ).toBe(true);
  });

  it("is false after the organization buys Polar credit", () => {
    expect(
      isHostedCreditTrialOrg({
        purchasedCreditCents: "10000",
        welcomeCreditExpiresAt: inFourteenDays,
      }),
    ).toBe(false);
  });

  it("is false when welcome credit was never granted", () => {
    expect(isHostedCreditTrialOrg({ purchasedCreditCents: "0" })).toBe(false);
  });
});

describe("hostedCreditBillingBalanceCopy", () => {
  it("explains trial usage when Polar has no customer and credit remains", () => {
    expect(
      hostedCreditBillingBalanceCopy({
        remainingCents: 4124,
        purchasedCents: 0,
        hasBillingCustomer: false,
        billingEnabled: true,
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toEqual({
      badge: "Trial",
      description:
        `This is trial usage for machines and managed models. ` +
        `The trial ends on ${new Date(inFourteenDays).toLocaleDateString()}. ` +
        `Subscribe to Business to keep hosted runs.`,
    });
  });

  it("keeps trial usage copy when trial credit is spent before expiry", () => {
    expect(
      hostedCreditBillingBalanceCopy({
        remainingCents: 0,
        purchasedCents: 0,
        hasBillingCustomer: false,
        billingEnabled: true,
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toEqual({
      badge: "Trial",
      description:
        `This is trial usage for machines and managed models. ` +
        `The trial ends on ${new Date(inFourteenDays).toLocaleDateString()}. ` +
        `Subscribe to Business to keep hosted runs.`,
    });
  });

  it("tells the owner the trial ended", () => {
    expect(
      hostedCreditBillingBalanceCopy({
        remainingCents: 0,
        purchasedCents: 0,
        hasBillingCustomer: false,
        billingEnabled: true,
        welcomeCreditExpiresAt: yesterday,
        now,
      }),
    ).toEqual({
      badge: "Trial",
      description: "The trial has ended. Hosted runs cannot start. Subscribe to Business to continue.",
    });
  });

  it("hides trial copy on an active Business plan", () => {
    expect(
      hostedCreditBillingBalanceCopy({
        remainingCents: 4124,
        purchasedCents: 0,
        hasBillingCustomer: true,
        billingEnabled: true,
        welcomeCreditExpiresAt: inFourteenDays,
        plan: "business",
        creditPurchaseAllowed: true,
        now,
      }),
    ).toEqual({ badge: "Business", description: null });
  });
});

describe("hostedCreditBannerKind", () => {
  it("shows the trial banner while welcome credit remains", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "4124",
        grantTotalCents: "5000",
        purchasedCreditCents: "0",
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toBe("trial");
  });

  it("keeps the trial kind when welcome credit is spent before expiry", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "0",
        grantTotalCents: "5000",
        superplaneGrantCents: "5000",
        purchasedCreditCents: "0",
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toBe("trial");
  });

  it("shows trial-expired when welcome credit expires", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "0",
        grantTotalCents: "5000",
        purchasedCreditCents: "0",
        welcomeCreditExpiresAt: yesterday,
        now,
      }),
    ).toBe("trial-expired");
  });

  it("hides trial copy after the organization buys credit", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "10000",
        grantTotalCents: "15000",
        purchasedCreditCents: "10000",
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toBeNull();
  });

  it("shows the paid empty banner after a purchase when remaining credit is empty", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "0",
        grantTotalCents: "10000",
        purchasedCreditCents: "10000",
        welcomeCreditExpiresAt: yesterday,
        billingEnabled: true,
        now,
      }),
    ).toBe("empty");
  });

  it("shows low when purchased credit remains at or below $20", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "1500",
        grantTotalCents: "15000",
        purchasedCreditCents: "10000",
        billingEnabled: true,
        now,
      }),
    ).toBe("low");
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "2000",
        grantTotalCents: "15000",
        purchasedCreditCents: "10000",
        billingEnabled: true,
        now,
      }),
    ).toBe("low");
  });

  it("hides the paid banner just above the $20 threshold", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "2001",
        grantTotalCents: "15000",
        purchasedCreditCents: "10000",
        billingEnabled: true,
        now,
      }),
    ).toBeNull();
  });

  it("keeps the trial banner when welcome credit is low but not empty", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "1500",
        grantTotalCents: "5000",
        purchasedCreditCents: "0",
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toBe("trial");
  });
});

describe("isHostedCreditHeaderKickerKind", () => {
  it("uses the title chip for every credit state", () => {
    expect(isHostedCreditHeaderKickerKind("trial")).toBe(true);
    expect(isHostedCreditHeaderKickerKind("trial-expired")).toBe(true);
    expect(isHostedCreditHeaderKickerKind("lapsed")).toBe(true);
    expect(isHostedCreditHeaderKickerKind("low")).toBe(true);
    expect(isHostedCreditHeaderKickerKind("empty")).toBe(true);
  });
});

describe("hostedCreditHeaderKickerLabel", () => {
  it("names the chip for each header kind", () => {
    expect(hostedCreditHeaderKickerLabel("trial")).toBe("Trial");
    expect(hostedCreditHeaderKickerLabel("trial-expired")).toBe("Trial ended");
    expect(hostedCreditHeaderKickerLabel("lapsed")).toBe("No plan");
    expect(hostedCreditHeaderKickerLabel("low")).toBe("Credit low");
    expect(hostedCreditHeaderKickerLabel("empty")).toBe("No credit");
  });
});

describe("hostedCreditHeaderKickerActionLabel", () => {
  it("opens billing for low and empty credit", () => {
    expect(hostedCreditHeaderKickerActionLabel("low")).toBe("Add credits");
    expect(hostedCreditHeaderKickerActionLabel("empty")).toBe("Add credits");
  });

  it("keeps Subscribe for trial and no-plan states", () => {
    expect(hostedCreditHeaderKickerActionLabel("trial")).toBe("Subscribe");
    expect(hostedCreditHeaderKickerActionLabel("trial-expired")).toBe("Subscribe");
    expect(hostedCreditHeaderKickerActionLabel("lapsed")).toBe("Subscribe");
  });
});

describe("organizationPlanLabel", () => {
  it("names Business, Trial, Trial ended, and No plan", () => {
    expect(organizationPlanLabel({ plan: "business" })).toBe("Business");
    expect(
      organizationPlanLabel({
        plan: "trial",
        trialEndsAt: inFourteenDays,
        now,
      }),
    ).toBe("Trial");
    expect(
      organizationPlanLabel({
        plan: "trial",
        trialEndsAt: yesterday,
        now,
      }),
    ).toBe("Trial ended");
    expect(organizationPlanLabel({ plan: "none" })).toBe("No plan");
  });

  it("names inferred trial from welcome credit when plan is unset", () => {
    expect(
      organizationPlanLabel({
        purchasedCreditCents: "0",
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toBe("Trial");
  });

  it("returns undefined when the plan is unknown", () => {
    expect(organizationPlanLabel({})).toBeUndefined();
    expect(organizationPlanLabel({ plan: "enterprise" })).toBeUndefined();
  });
});

describe("shouldShowHostedCreditEmptyBanner", () => {
  it("hides the banner when remaining hosted credit is greater than zero", () => {
    expect(
      shouldShowHostedCreditEmptyBanner({
        remainingCreditCents: "4124",
        grantTotalCents: "5000",
        billingEnabled: true,
      }),
    ).toBe(false);
  });

  it("shows the banner when a grant is spent and remaining credit is empty", () => {
    expect(
      shouldShowHostedCreditEmptyBanner({
        remainingCreditCents: "0",
        grantTotalCents: "5000",
        superplaneGrantCents: "5000",
        purchasedCreditCents: "0",
        billingEnabled: true,
      }),
    ).toBe(true);
  });

  it("shows the banner when billing is on and remaining credit is empty", () => {
    expect(
      shouldShowHostedCreditEmptyBanner({
        remainingCreditCents: "0",
        grantTotalCents: "0",
        billingEnabled: true,
      }),
    ).toBe(true);
  });

  it("hides the banner when the organization has no grant and billing is off", () => {
    expect(
      shouldShowHostedCreditEmptyBanner({
        remainingCreditCents: "0",
        grantTotalCents: "0",
        superplaneGrantCents: "0",
        purchasedCreditCents: "0",
        billingEnabled: false,
      }),
    ).toBe(false);
  });

  it("shows the banner when remaining credit is low but not yet empty", () => {
    expect(
      shouldShowHostedCreditEmptyBanner({
        remainingCreditCents: "1500",
        grantTotalCents: "15000",
        purchasedCreditCents: "10000",
        billingEnabled: true,
      }),
    ).toBe(true);
  });
});

describe("welcomeCreditHeaderLabel", () => {
  it("names today, one day, and many days above the page title", () => {
    expect(welcomeCreditHeaderLabel(new Date("2026-09-08T18:00:00.000Z"), now)).toBe("Ends today");
    expect(welcomeCreditHeaderLabel(new Date("2026-09-09T12:00:00.000Z"), now)).toBe("1 day");
    expect(welcomeCreditHeaderLabel(new Date(inFourteenDays), now)).toBe("14 days");
  });

  it("names an ended trial", () => {
    expect(welcomeCreditHeaderLabel(new Date(yesterday), now)).toBe("Ended");
  });
});
