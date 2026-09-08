import { describe, expect, it } from "vitest";

import {
  hostedCreditBannerCopy,
  hostedCreditBannerKind,
  hostedCreditEmptyBannerCopy,
  shouldShowHostedCreditEmptyBanner,
  welcomeCreditExpirySentence,
} from "./hostedCreditEmpty";

const now = new Date("2026-09-08T12:00:00.000Z");
const inFourteenDays = "2026-09-22T12:00:00.000Z";
const yesterday = "2026-09-07T12:00:00.000Z";

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
});

describe("welcomeCreditExpirySentence", () => {
  it("names today, one day, and many days", () => {
    expect(welcomeCreditExpirySentence(new Date("2026-09-08T18:00:00.000Z"), now)).toBe("It expires today.");
    expect(welcomeCreditExpirySentence(new Date("2026-09-09T12:00:00.000Z"), now)).toBe("It expires in 1 day.");
    expect(welcomeCreditExpirySentence(new Date(inFourteenDays), now)).toBe("It expires in 14 days.");
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
      description: "You have $41.24 of free hosted credit. It expires in 14 days. Open Billing to buy more credit.",
      actionLabel: "Open billing",
    });
  });

  it("tells the user to buy credit when the trial is empty", () => {
    expect(hostedCreditBannerCopy({ kind: "trial-empty", billingEnabled: true })).toEqual({
      title: "Trial credit is empty",
      description: "SuperPlane-hosted runs cannot start. Open Billing to buy hosted credit.",
      actionLabel: "Open billing",
    });
  });

  it("tells the user the trial ended", () => {
    expect(hostedCreditBannerCopy({ kind: "trial-expired", billingEnabled: true })).toEqual({
      title: "Trial ended",
      description:
        "Free hosted credit expired. SuperPlane-hosted runs cannot start. Open Billing to buy hosted credit.",
      actionLabel: "Open billing",
    });
  });
});

describe("hostedCreditEmptyBannerCopy", () => {
  it("tells the user to view billing when billing is on", () => {
    expect(hostedCreditEmptyBannerCopy(true)).toEqual({
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start. Open Billing to review remaining credit.",
      actionLabel: "View billing",
    });
  });

  it("asks the user to wait for an installation admin when billing is off", () => {
    expect(hostedCreditEmptyBannerCopy(false)).toEqual({
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
      actionLabel: "View billing",
    });
  });

  it("points every role to Billing", () => {
    expect(hostedCreditEmptyBannerCopy(true, false)).toEqual({
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start. Open Billing to review remaining credit.",
      actionLabel: "View billing",
    });
  });
});
