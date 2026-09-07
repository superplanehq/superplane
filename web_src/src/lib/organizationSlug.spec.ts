import { describe, expect, it } from "vitest";

import { isValidOrganizationSlugFormat, slugifyOrganizationName, validateOrganizationSlug } from "./organizationSlug";

describe("slugifyOrganizationName", () => {
  it("lowercases and dashes a simple name", () => {
    expect(slugifyOrganizationName("Acme Corp")).toBe("acme-corp");
  });

  it("collapses punctuation and whitespace runs", () => {
    expect(slugifyOrganizationName("  Acme,   Corp!! ")).toBe("acme-corp");
  });

  it("trims leading and trailing dashes", () => {
    expect(slugifyOrganizationName("-Acme Corp-")).toBe("acme-corp");
  });

  it("leaves an already-valid slug untouched", () => {
    expect(slugifyOrganizationName("already-a-slug")).toBe("already-a-slug");
  });

  it("caps the length", () => {
    const longName = "a".repeat(100);
    expect(slugifyOrganizationName(longName).length).toBeLessThanOrEqual(63);
  });
});

describe("isValidOrganizationSlugFormat", () => {
  it("accepts lowercase letters, numbers, and dashes", () => {
    expect(isValidOrganizationSlugFormat("acme-corp-2")).toBe(true);
  });

  it("rejects uppercase letters", () => {
    expect(isValidOrganizationSlugFormat("Acme-Corp")).toBe(false);
  });

  it("rejects spaces and punctuation", () => {
    expect(isValidOrganizationSlugFormat("acme corp!")).toBe(false);
  });

  it("rejects an empty string", () => {
    expect(isValidOrganizationSlugFormat("")).toBe(false);
  });
});

describe("validateOrganizationSlug", () => {
  it("returns null for a valid, non-reserved slug", () => {
    expect(validateOrganizationSlug("acme-corp")).toBeNull();
  });

  it("flags an empty slug", () => {
    expect(validateOrganizationSlug("")).toBe("empty");
  });

  it("flags an invalid format", () => {
    expect(validateOrganizationSlug("Acme Corp")).toBe("invalid-format");
  });

  it("flags a reserved slug", () => {
    expect(validateOrganizationSlug("admin")).toBe("reserved");
  });
});
