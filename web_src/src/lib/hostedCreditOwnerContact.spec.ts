import { describe, expect, it } from "vitest";

import { hostedCreditOwnerContactCopy } from "./hostedCreditOwnerContact";

describe("hostedCreditOwnerContactCopy", () => {
  it("names a single owner by display name", () => {
    expect(
      hostedCreditOwnerContactCopy({
        organizationName: "Acme",
        owners: [{ name: "Jane Doe" }],
      }),
    ).toBe("Contact the Acme owner (Jane Doe) to purchase hosted credit.");
  });

  it("lists several owners with the plural article and comma-joined labels", () => {
    expect(
      hostedCreditOwnerContactCopy({
        organizationName: "Acme",
        owners: [{ name: "Jane Doe" }, { name: "John Smith" }],
      }),
    ).toBe("Contact an Acme owner (Jane Doe, John Smith) to purchase hosted credit.");
  });

  it("falls back to email when an owner has no display name", () => {
    expect(
      hostedCreditOwnerContactCopy({
        organizationName: "Acme",
        owners: [{ email: "jane@acme.com" }],
      }),
    ).toBe("Contact the Acme owner (jane@acme.com) to purchase hosted credit.");
  });

  it("uses generic organization wording when the organization name is missing", () => {
    expect(
      hostedCreditOwnerContactCopy({
        owners: [{ name: "Jane Doe" }],
      }),
    ).toBe("Contact the organization owner (Jane Doe) to purchase hosted credit.");
  });

  it("falls back to a generic sentence when no owner has a usable label", () => {
    expect(
      hostedCreditOwnerContactCopy({
        organizationName: "Acme",
        owners: [],
      }),
    ).toBe("Contact an organization owner to purchase hosted credit.");
  });

  it("skips owners with neither a name nor an email", () => {
    expect(
      hostedCreditOwnerContactCopy({
        organizationName: "Acme",
        owners: [{}, { name: "Jane Doe" }],
      }),
    ).toBe("Contact the Acme owner (Jane Doe) to purchase hosted credit.");
  });
});
