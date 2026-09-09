import { describe, expect, it } from "vitest";

import {
  HOSTED_CREDIT_RUNS_STOP_HINT,
  hostedCreditBannerCopy,
  hostedCreditBannerKind,
  hostedCreditBannerTone,
  hostedCreditBillingBalanceCopy,
  hostedCreditEmptyBannerCopy,
  hostedCreditRunsStopHint,
  isHostedCreditTrialOrg,
  isLowHostedCreditRemaining,
  shouldShowHostedCreditEmptyBanner,
  welcomeCreditExpiryLabel,
  welcomeCreditExpirySentence,
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
  it("explains welcome trial credit when Polar has no customer and credit remains", () => {
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
        `This remaining balance is welcome credit. Welcome credit is a free trial grant. ` +
        `Unused credit expires on ${new Date(inFourteenDays).toLocaleDateString()}. ` +
        `Purchase hosted credit to keep SuperPlane-hosted runs after the trial.`,
    });
  });

  it("tells the owner to purchase when Polar has no customer and remaining credit is empty", () => {
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
      description: "Hosted credit is empty. Click Buy more to purchase hosted credit.",
    });
  });

  it("tells the owner the trial ended when welcome credit expires", () => {
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
      description:
        "Welcome credit expired. SuperPlane-hosted runs cannot start. Click Buy more to purchase hosted credit.",
    });
  });

  it("hides trial copy when Polar already has a customer", () => {
    expect(
      hostedCreditBillingBalanceCopy({
        remainingCents: 4124,
        purchasedCents: 0,
        hasBillingCustomer: true,
        billingEnabled: true,
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toEqual({ badge: null, description: null });
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

  it("shows trial-empty when welcome credit is spent before expiry", () => {
    expect(
      hostedCreditBannerKind({
        remainingCreditCents: "0",
        grantTotalCents: "5000",
        superplaneGrantCents: "5000",
        purchasedCreditCents: "0",
        welcomeCreditExpiresAt: inFourteenDays,
        now,
      }),
    ).toBe("trial-empty");
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

describe("welcomeCreditExpirySentence", () => {
  it("names today, one day, and many days", () => {
    expect(welcomeCreditExpirySentence(new Date("2026-09-08T18:00:00.000Z"), now)).toBe("It expires today.");
    expect(welcomeCreditExpirySentence(new Date("2026-09-09T12:00:00.000Z"), now)).toBe("It expires in 1 day.");
    expect(welcomeCreditExpirySentence(new Date(inFourteenDays), now)).toBe("It expires in 14 days.");
  });
});

describe("welcomeCreditExpiryLabel", () => {
  it("names today, one day, and many days as short labels", () => {
    expect(welcomeCreditExpiryLabel(new Date("2026-09-08T18:00:00.000Z"), now)).toBe("Expires today");
    expect(welcomeCreditExpiryLabel(new Date("2026-09-09T12:00:00.000Z"), now)).toBe("1 day remaining");
    expect(welcomeCreditExpiryLabel(new Date(inFourteenDays), now)).toBe("14 days remaining");
  });
});

describe("hostedCreditBannerTone", () => {
  it("keeps a healthy trial informational", () => {
    expect(hostedCreditBannerTone("trial", new Date(inFourteenDays), now)).toBe("info");
  });

  it("warns when the trial expires today", () => {
    expect(hostedCreditBannerTone("trial", new Date("2026-09-08T18:00:00.000Z"), now)).toBe("warning");
  });

  it("warns when hosted runs cannot start", () => {
    expect(hostedCreditBannerTone("trial-empty")).toBe("warning");
    expect(hostedCreditBannerTone("trial-expired")).toBe("warning");
    expect(hostedCreditBannerTone("empty")).toBe("warning");
    expect(hostedCreditBannerTone("low")).toBe("warning");
  });

  it("warns when trial credit is at or below $20", () => {
    expect(hostedCreditBannerTone("trial", new Date(inFourteenDays), now, 1500)).toBe("warning");
  });
});

describe("isLowHostedCreditRemaining", () => {
  it("is true at or below $20 while credit remains", () => {
    expect(isLowHostedCreditRemaining(2000)).toBe(true);
    expect(isLowHostedCreditRemaining(432)).toBe(true);
    expect(isLowHostedCreditRemaining(2001)).toBe(false);
    expect(isLowHostedCreditRemaining(0)).toBe(false);
    expect(isLowHostedCreditRemaining(undefined)).toBe(false);
  });
});

describe("hostedCreditRunsStopHint", () => {
  it("names the stop outcome only while remaining credit is low", () => {
    expect(hostedCreditRunsStopHint(432)).toBe(HOSTED_CREDIT_RUNS_STOP_HINT);
    expect(hostedCreditRunsStopHint(4124)).toBeUndefined();
    expect(hostedCreditRunsStopHint(0)).toBeUndefined();
  });
});

describe("hostedCreditBannerCopy", () => {
  it("names remaining trial credit and the expiry window", () => {
    expect(
      hostedCreditBannerCopy({
        kind: "trial",
        billingEnabled: true,
        remainingCreditCents: 4124,
        welcomeCreditExpiresAt: new Date(inFourteenDays),
        now,
      }),
    ).toEqual({
      title: "Trial",
      description: "You have $41.24 of free hosted credit. It expires in 14 days.",
      remainingLabel: "$41.24 remaining",
      expiryLabel: "14 days remaining",
      actionLabel: "Add credits",
      tone: "info",
    });
  });

  it("tells the user that tasks stop when trial credit is low", () => {
    expect(
      hostedCreditBannerCopy({
        kind: "trial",
        billingEnabled: true,
        remainingCreditCents: 432,
        welcomeCreditExpiresAt: new Date(inFourteenDays),
        now,
      }),
    ).toEqual({
      title: "Trial",
      description: "You have $4.32 of free hosted credit. It expires in 14 days.",
      remainingLabel: "$4.32 remaining",
      expiryLabel: "14 days remaining",
      consequenceHint: HOSTED_CREDIT_RUNS_STOP_HINT,
      actionLabel: "Add credits",
      tone: "warning",
    });
  });

  it("tells the user to buy credit when the trial is empty", () => {
    expect(hostedCreditBannerCopy({ kind: "trial-empty", billingEnabled: true })).toEqual({
      title: "Trial credit is empty",
      description: "SuperPlane-hosted runs cannot start.",
      actionLabel: "Add credits",
      tone: "warning",
    });
  });

  it("tells the user the trial ended", () => {
    expect(hostedCreditBannerCopy({ kind: "trial-expired", billingEnabled: true })).toEqual({
      title: "Trial ended",
      description: "Free hosted credit expired. SuperPlane-hosted runs cannot start.",
      actionLabel: "Add credits",
      tone: "warning",
    });
  });

  it("warns before purchased credit runs out", () => {
    expect(
      hostedCreditBannerCopy({
        kind: "low",
        billingEnabled: true,
        remainingCreditCents: 1500,
      }),
    ).toEqual({
      title: "Hosted credit is low",
      description: "Less than $20.00 remains. Add hosted credit to keep SuperPlane-hosted runs.",
      remainingLabel: "$15.00 remaining",
      consequenceHint: HOSTED_CREDIT_RUNS_STOP_HINT,
      actionLabel: "Add credits",
      tone: "warning",
    });
  });

  it("asks an installation admin to add credit when billing is off and credit is low", () => {
    expect(hostedCreditBannerCopy({ kind: "low", billingEnabled: false, remainingCreditCents: 1500 })).toEqual({
      title: "Hosted credit is low",
      description: "Less than $20.00 remains. Ask an installation admin to add hosted credit.",
      remainingLabel: "$15.00 remaining",
      consequenceHint: HOSTED_CREDIT_RUNS_STOP_HINT,
      actionLabel: "Add credits",
      tone: "warning",
    });
  });
});

describe("hostedCreditEmptyBannerCopy", () => {
  it("tells the user to view billing when billing is on", () => {
    expect(hostedCreditEmptyBannerCopy(true)).toEqual({
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start.",
      actionLabel: "Add credits",
      tone: "warning",
    });
  });

  it("asks the user to wait for an installation admin when billing is off", () => {
    expect(hostedCreditEmptyBannerCopy(false)).toEqual({
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
      actionLabel: "Add credits",
      tone: "warning",
    });
  });

  it("points every role to Billing", () => {
    expect(hostedCreditEmptyBannerCopy(true, false)).toEqual({
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start.",
      actionLabel: "Add credits",
      tone: "warning",
    });
  });
});
