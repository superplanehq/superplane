import { describe, expect, it } from "vitest";

import {
  hostedCreditEmptyBannerCopy,
  hostedCreditWarningLevel,
  shouldShowHostedCreditEmptyBanner,
} from "./hostedCreditEmpty";

describe("hostedCreditWarningLevel", () => {
  it("returns null when remaining hosted credit is comfortably above the low-credit threshold", () => {
    expect(
      hostedCreditWarningLevel({
        remainingCreditCents: "4124",
        grantTotalCents: "5000",
        billingEnabled: true,
      }),
    ).toBeNull();
  });

  it("returns 'empty' when a grant is spent and remaining credit is zero", () => {
    expect(
      hostedCreditWarningLevel({
        remainingCreditCents: "0",
        grantTotalCents: "5000",
        superplaneGrantCents: "5000",
        purchasedCreditCents: "0",
        billingEnabled: true,
      }),
    ).toBe("empty");
  });

  it("returns 'empty' when billing is on and remaining credit is zero", () => {
    expect(
      hostedCreditWarningLevel({
        remainingCreditCents: "0",
        grantTotalCents: "0",
        billingEnabled: true,
      }),
    ).toBe("empty");
  });

  it("returns 'low' when remaining credit is above zero but at or below the threshold", () => {
    expect(
      hostedCreditWarningLevel({
        remainingCreditCents: "1500",
        grantTotalCents: "5000",
        billingEnabled: true,
      }),
    ).toBe("low");
  });

  it("returns 'low' at the threshold boundary of $20", () => {
    expect(
      hostedCreditWarningLevel({
        remainingCreditCents: "2000",
        grantTotalCents: "5000",
        billingEnabled: true,
      }),
    ).toBe("low");
  });

  it("returns null just above the $20 threshold", () => {
    expect(
      hostedCreditWarningLevel({
        remainingCreditCents: "2001",
        grantTotalCents: "5000",
        billingEnabled: true,
      }),
    ).toBeNull();
  });

  it("returns null when the organization has no grant and billing is off", () => {
    expect(
      hostedCreditWarningLevel({
        remainingCreditCents: "0",
        grantTotalCents: "0",
        superplaneGrantCents: "0",
        purchasedCreditCents: "0",
        billingEnabled: false,
      }),
    ).toBeNull();
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

  it("shows the banner when remaining credit is low but not yet empty", () => {
    expect(
      shouldShowHostedCreditEmptyBanner({
        remainingCreditCents: "1500",
        grantTotalCents: "5000",
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
  it("tells the user to view spending when billing is on and credit is empty", () => {
    expect(hostedCreditEmptyBannerCopy("empty", true)).toEqual({
      title: "Hosted credit is empty",
      description: "Add hosted credit to start SuperPlane-hosted runs.",
    });
  });

  it("asks the user to wait for an installation admin when billing is off and credit is empty", () => {
    expect(hostedCreditEmptyBannerCopy("empty", false)).toEqual({
      title: "Hosted credit is empty",
      description: "SuperPlane-hosted runs cannot start until an installation admin adds credit.",
    });
  });

  it("warns before credit runs out when billing is on and credit is low", () => {
    expect(hostedCreditEmptyBannerCopy("low", true)).toEqual({
      title: "Hosted credit is running low",
      description: "Less than $20.00 left. Add hosted credit to avoid interrupting SuperPlane-hosted runs.",
    });
  });

  it("asks the user to wait for an installation admin when billing is off and credit is low", () => {
    expect(hostedCreditEmptyBannerCopy("low", false)).toEqual({
      title: "Hosted credit is running low",
      description: "Less than $20.00 left. Ask an installation admin to add hosted credit soon.",
    });
  });
});
