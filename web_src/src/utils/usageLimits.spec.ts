import { describe, expect, it } from "vitest";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/utils/usageLimits";

describe("usageLimits", () => {
  it("maps organization canvas limit errors to a usage notice with link", () => {
    const notice = getUsageLimitNotice({ error: { message: "organization canvas limit exceeded" } }, "org-123");

    expect(notice).not.toBeNull();
    expect(notice?.title).toBe("Canvas limit reached");
    expect(notice?.href).toBe("/org-123/settings/billing");
    expect(notice?.actionLabel).toBe("View usage");
  });

  it("returns null for unknown errors", () => {
    expect(getUsageLimitNotice({ error: { message: "something else" } }, "org-123")).toBeNull();
  });

  it("uses mapped usage text when available and falls back otherwise", () => {
    expect(getUsageLimitToastMessage({ error: { message: "organization user limit exceeded" } }, "fallback")).toBe(
      "Member limit reached for this organization.",
    );
    expect(getUsageLimitToastMessage(undefined, "fallback")).toBe("fallback");
  });

  it("maps plain string errors to usage notices", () => {
    const notice = getUsageLimitNotice("organization user limit exceeded", "org-123");

    expect(notice).not.toBeNull();
    expect(notice?.title).toBe("Member limit reached");
    expect(notice?.href).toBe("/org-123/settings/billing");
  });
});
