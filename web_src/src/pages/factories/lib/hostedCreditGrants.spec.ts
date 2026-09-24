import { describe, expect, it } from "bun:test";

import {
  creditGrantDetails,
  creditGrantSourceLabel,
  formatCreditGrantAmount,
  hostedCreditBalanceWarning,
  welcomeCreditUnusedExpiryNote,
} from "./hostedCreditGrants";

describe("creditGrantSourceLabel", () => {
  it("maps stored grant kinds to page labels", () => {
    expect(creditGrantSourceLabel("welcome")).toBe("Trial");
    expect(creditGrantSourceLabel("admin")).toBe("SuperPlane grant");
    expect(creditGrantSourceLabel("included")).toBe("Included");
    expect(creditGrantSourceLabel("topup")).toBe("Top-up");
    expect(creditGrantSourceLabel("topup_refund")).toBe("Refund");
  });
});

describe("creditGrantDetails", () => {
  it("joins an admin note and actor name", () => {
    expect(
      creditGrantDetails({
        kind: "admin",
        note: "Support grant",
        actorName: "Ada",
      }),
    ).toBe("Support grant · Granted by Ada");
  });

  it("shows a stored purchase order id", () => {
    expect(creditGrantDetails({ kind: "topup", polarOrderId: "ord_123" })).toBe("Order ord_123");
  });

  it("names the welcome credit expiry date", () => {
    const now = new Date("2026-09-08T12:00:00.000Z");
    expect(creditGrantDetails({ kind: "welcome", expiresAt: "2026-09-22T12:00:00.000Z" }, now)).toMatch(/^Expires on /);
    expect(creditGrantDetails({ kind: "welcome", expiresAt: "2026-09-01T12:00:00.000Z" }, now)).toMatch(/^Expired on /);
  });
});

describe("formatCreditGrantAmount", () => {
  it("signs added and refunded amounts", () => {
    expect(formatCreditGrantAmount(2500)).toBe("+$25.00");
    expect(formatCreditGrantAmount(-500)).toBe("-$5.00");
    expect(formatCreditGrantAmount(0)).toBe("$0.00");
  });
});

describe("hostedCreditBalanceWarning", () => {
  it("names the empty and low remaining-credit states", () => {
    expect(hostedCreditBalanceWarning(0, true)).toBe("Hosted credit is empty. SuperPlane-hosted runs cannot start.");
    expect(hostedCreditBalanceWarning(0, true, true)).toBe(
      "Welcome credit expired. SuperPlane-hosted runs cannot start.",
    );
    expect(hostedCreditBalanceWarning(100, true)).toBe("Hosted credit is low.");
    expect(hostedCreditBalanceWarning(100, false)).toBeNull();
  });
});

describe("welcomeCreditUnusedExpiryNote", () => {
  it("explains unused expired welcome credit", () => {
    expect(
      welcomeCreditUnusedExpiryNote({
        remainingCents: 0,
        purchasedCents: 0,
        welcomeCreditExpiresAt: "2026-09-01T12:00:00.000Z",
        now: new Date("2026-09-08T12:00:00.000Z"),
      }),
    ).toBe("Welcome credit expired. Unused free credit no longer pays for SuperPlane-hosted runs.");
  });

  it("hides the note after a purchase", () => {
    expect(
      welcomeCreditUnusedExpiryNote({
        remainingCents: 10000,
        purchasedCents: 10000,
        welcomeCreditExpiresAt: "2026-09-01T12:00:00.000Z",
        now: new Date("2026-09-08T12:00:00.000Z"),
      }),
    ).toBeNull();
  });
});
