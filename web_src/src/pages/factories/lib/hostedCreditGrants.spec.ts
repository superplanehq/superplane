import { describe, expect, it } from "vitest";

import {
  creditGrantDetails,
  creditGrantSourceLabel,
  formatCreditGrantAmount,
  hostedCreditBalanceWarning,
} from "./hostedCreditGrants";

describe("creditGrantSourceLabel", () => {
  it("maps stored grant kinds to page labels", () => {
    expect(creditGrantSourceLabel("welcome")).toBe("Welcome credit");
    expect(creditGrantSourceLabel("admin")).toBe("SuperPlane grant");
    expect(creditGrantSourceLabel("polar")).toBe("Purchased");
    expect(creditGrantSourceLabel("polar_refund")).toBe("Refund");
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
    expect(creditGrantDetails({ kind: "polar", polarOrderId: "ord_123" })).toBe("Order ord_123");
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
    expect(hostedCreditBalanceWarning(100, true)).toBe("Hosted credit is low.");
    expect(hostedCreditBalanceWarning(100, false)).toBeNull();
  });
});
