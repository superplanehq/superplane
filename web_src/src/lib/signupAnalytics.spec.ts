import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  clearPendingSignupAnalyticsPreference,
  confirmSignupAnalyticsPreference,
  consumePendingSignupAnalyticsPreference,
  savePendingSignupAnalyticsPreference,
} from "./signupAnalytics";

describe("signup analytics preference", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-06-18T12:00:00Z"));
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
    vi.useRealTimers();
  });

  it("consumes a confirmed signup preference outside welcome", () => {
    confirmSignupAnalyticsPreference({
      email: "new-user@example.com",
      productUpdatesOptIn: false,
    });

    const preference = consumePendingSignupAnalyticsPreference({
      accountEmail: "new-user@example.com",
      currentPath: "/org-123",
    });

    expect(preference).toEqual({
      email: "new-user@example.com",
      productUpdatesOptIn: false,
    });
    expect(
      consumePendingSignupAnalyticsPreference({ accountEmail: "new-user@example.com", currentPath: "/org-123" }),
    ).toBeNull();
  });

  it("consumes an unconfirmed signup preference on welcome", () => {
    savePendingSignupAnalyticsPreference({
      productUpdatesOptIn: true,
    });

    expect(consumePendingSignupAnalyticsPreference({ currentPath: "/welcome" })).toEqual({
      email: undefined,
      productUpdatesOptIn: true,
    });
  });

  it("does not consume an unconfirmed signup preference outside welcome", () => {
    savePendingSignupAnalyticsPreference({
      productUpdatesOptIn: true,
    });

    expect(consumePendingSignupAnalyticsPreference({ currentPath: "/org-123" })).toBeNull();
  });

  it("consumes an unconfirmed signup preference when signup result is created", () => {
    savePendingSignupAnalyticsPreference({
      productUpdatesOptIn: false,
    });

    expect(consumePendingSignupAnalyticsPreference({ currentPath: "/org-123", signupResult: "created" })).toEqual({
      email: undefined,
      productUpdatesOptIn: false,
    });
  });

  it("clears an unconfirmed signup preference when signup result is existing", () => {
    savePendingSignupAnalyticsPreference({
      productUpdatesOptIn: true,
    });

    expect(consumePendingSignupAnalyticsPreference({ currentPath: "/org-123", signupResult: "existing" })).toBeNull();
    expect(consumePendingSignupAnalyticsPreference({ currentPath: "/welcome" })).toBeNull();
  });

  it("does not consume a signup preference for a different account email", () => {
    confirmSignupAnalyticsPreference({
      email: "new-user@example.com",
      productUpdatesOptIn: true,
    });

    expect(
      consumePendingSignupAnalyticsPreference({
        accountEmail: "other-user@example.com",
        currentPath: "/welcome",
      }),
    ).toBeNull();
  });

  it("clears expired signup preferences", () => {
    savePendingSignupAnalyticsPreference({
      productUpdatesOptIn: true,
    });
    vi.advanceTimersByTime(25 * 60 * 60 * 1000);

    expect(consumePendingSignupAnalyticsPreference({ currentPath: "/welcome" })).toBeNull();
  });

  it("clears pending signup preferences", () => {
    savePendingSignupAnalyticsPreference({
      productUpdatesOptIn: true,
    });

    clearPendingSignupAnalyticsPreference();

    expect(consumePendingSignupAnalyticsPreference({ currentPath: "/welcome" })).toBeNull();
  });
});
