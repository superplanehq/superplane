import { describe, expect, it } from "vitest";

import { getSignupUnavailableReason } from "./signupUnavailableReason";

describe("getSignupUnavailableReason", () => {
  it("does not use waitlist state when signup is available", () => {
    expect(getSignupUnavailableReason(false, false, true)).toBeNull();
  });

  it("uses waitlist state only when the waitlist config is complete", () => {
    expect(getSignupUnavailableReason(true, false, true)).toBe("waitlist");
  });

  it("uses closed state when waitlist config is incomplete", () => {
    expect(getSignupUnavailableReason(true, false, false)).toBe("closed");
  });

  it("uses closed state when signups are blocked by environment", () => {
    expect(getSignupUnavailableReason(true, true, true)).toBe("closed");
  });
});
