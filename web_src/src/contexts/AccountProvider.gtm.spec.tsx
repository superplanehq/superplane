import { render, waitFor } from "@testing-library/react";
import { StrictMode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
vi.mock("@/posthog", () => ({ posthog: { identify: vi.fn(), capture: vi.fn() } }));
const { AccountProvider } = await import("./AccountProvider");
import { confirmSignupAnalyticsPreference, savePendingSignupAnalyticsPreference } from "@/lib/signupAnalytics";

let accountID = 0;
describe("signup conversion after account confirmation", () => {
  afterEach(() => {
    Reflect.deleteProperty(window.location, "hostname");
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });
  beforeEach(() => {
    delete window.dataLayer;
    localStorage.clear();
    Object.defineProperty(window.location, "hostname", { configurable: true, value: "app.superplane.com" });
    window.history.replaceState({}, "", "/welcome");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        status: 200,
        headers: new Headers(),
        json: async () => ({ id: `account-${++accountID}`, email: "new@example.com" }),
      }),
    );
  });
  it("fires for a successful password registration after loading the account", async () => {
    confirmSignupAnalyticsPreference({ email: "new@example.com", productUpdatesOptIn: false });
    render(<AccountProvider>Onboarding</AccountProvider>);
    await waitFor(() => expect(window.dataLayer).toEqual([{ event: "sign_up" }]));
  });
  it("fires once for a new OAuth account under StrictMode and removes the result parameter", async () => {
    window.history.replaceState({}, "", "/welcome?auth_signup_result=created");
    render(
      <StrictMode>
        <AccountProvider>Onboarding</AccountProvider>
      </StrictMode>,
    );
    await waitFor(() => expect(window.dataLayer).toEqual([{ event: "sign_up" }]));
    expect(window.location.search).toBe("");
  });
  it("does not count an existing OAuth account", async () => {
    confirmSignupAnalyticsPreference({ email: "new@example.com", productUpdatesOptIn: true });
    window.history.replaceState({}, "", "/welcome?auth_signup_result=existing");
    render(<AccountProvider>Onboarding</AccountProvider>);
    await waitFor(() => expect(localStorage.getItem("superplane:pending_signup_analytics_preference")).toBeNull());
    expect(window.dataLayer).toBeUndefined();
  });
  it("does not count a welcome page visit or an unconfirmed signup attempt", async () => {
    savePendingSignupAnalyticsPreference({ email: "new@example.com", productUpdatesOptIn: true });
    render(<AccountProvider>Onboarding</AccountProvider>);
    await waitFor(() => expect(localStorage.getItem("superplane:pending_signup_analytics_preference")).toBeNull());
    expect(window.dataLayer).toBeUndefined();
  });
  it("does not count failed authentication", async () => {
    const json = vi.fn();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ status: 401, headers: new Headers(), json }));
    confirmSignupAnalyticsPreference({ email: "new@example.com", productUpdatesOptIn: true });
    render(<AccountProvider>Onboarding</AccountProvider>);
    await waitFor(() => expect(fetch).toHaveBeenCalled());
    expect(json).not.toHaveBeenCalled();
    expect(window.dataLayer).toBeUndefined();
  });
  it("does not count an impersonated account", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        status: 200,
        headers: new Headers(),
        json: async () => ({ id: "impersonated", email: "new@example.com", impersonation: { active: true } }),
      }),
    );
    window.history.replaceState({}, "", "/welcome?auth_signup_result=created");
    render(<AccountProvider>Onboarding</AccountProvider>);
    await waitFor(() => expect(fetch).toHaveBeenCalled());
    expect(window.dataLayer).toBeUndefined();
  });
});
