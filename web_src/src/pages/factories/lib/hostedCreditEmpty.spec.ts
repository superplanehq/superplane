import { describe, expect, it } from "vitest";

import { hostedCreditEmptyBannerCopy, shouldShowHostedCreditEmptyBanner } from "./hostedCreditEmpty";

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

describe("hostedCreditEmptyBannerCopy", () => {
  it("tells the user to add hosted credit when billing is on", () => {
    expect(hostedCreditEmptyBannerCopy(true)).toEqual({
      title: "Hosted credit is empty",
      description: "Add hosted credit to start SuperPlane-hosted runs.",
      actionLabel: "Add hosted credit",
    });
  });

  it("asks the user to wait for an installation admin when billing is off", () => {
    expect(hostedCreditEmptyBannerCopy(false)).toEqual({
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
      actionLabel: "View spending",
    });
  });
});
