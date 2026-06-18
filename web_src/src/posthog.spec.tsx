import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { init, identify, capture, reset, setOnce } = vi.hoisted(() => ({
  init: vi.fn(),
  identify: vi.fn(),
  capture: vi.fn(),
  reset: vi.fn(),
  setOnce: vi.fn(),
}));

vi.mock("posthog-js", () => ({
  default: { init, identify, capture, reset, people: { set_once: setOnce } },
}));

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: ReactNode; to: string }) => <a href={to}>{children}</a>,
  useNavigate: () => vi.fn(),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { metadata: { name: "Acme Corp" } } }),
  useOrganizationUsage: () => ({ data: null, error: null }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/lib/env", () => ({
  isUsagePageForced: () => false,
}));

import { AccountProvider } from "@/contexts/AccountProvider";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { confirmSignupAnalyticsPreference, savePendingSignupAnalyticsPreference } from "@/lib/signupAnalytics";

const mockAccount = {
  id: "user-123",
  name: "John Doe",
  email: "john@example.com",
  installation_admin: false,
};

const stubFetch = (data: object, status = 200) => {
  const mock = vi.fn().mockResolvedValue({
    status,
    headers: { get: () => null },
    json: () => Promise.resolve(data),
  });
  vi.stubGlobal("fetch", mock);
  return mock;
};

describe("posthog init", () => {
  beforeEach(() => {
    init.mockClear();
    setOnce.mockClear();
    localStorage.clear();
    document.cookie = "superplane_initial_utm=; Max-Age=0; Path=/";
    vi.resetModules();
  });

  afterEach(() => {
    delete (window as Window & { SUPERPLANE_POSTHOG_KEY?: string }).SUPERPLANE_POSTHOG_KEY;
    localStorage.clear();
    document.cookie = "superplane_initial_utm=; Max-Age=0; Path=/";
    window.history.replaceState({}, "", "/");
  });

  it("calls init when SUPERPLANE_POSTHOG_KEY is set", async () => {
    (window as Window & { SUPERPLANE_POSTHOG_KEY?: string }).SUPERPLANE_POSTHOG_KEY = "test-key";
    await import("@/posthog");
    expect(init).toHaveBeenCalledWith(
      "test-key",
      expect.objectContaining({ autocapture: false, capture_pageview: false, person_profiles: "always" }),
    );
  });

  it("does not call init when SUPERPLANE_POSTHOG_KEY is not set", async () => {
    delete (window as Window & { SUPERPLANE_POSTHOG_KEY?: string }).SUPERPLANE_POSTHOG_KEY;
    await import("@/posthog");
    expect(init).not.toHaveBeenCalled();
  });

  it("sets initial UTM person properties when PostHog initializes", async () => {
    (window as Window & { SUPERPLANE_POSTHOG_KEY?: string }).SUPERPLANE_POSTHOG_KEY = "test-key";
    window.history.replaceState({}, "", "/signup?utm_source=youtube&utm_campaign=erictech_beta");

    await import("@/posthog");

    expect(setOnce).toHaveBeenCalledWith({
      $initial_utm_source: "youtube",
      $initial_utm_campaign: "erictech_beta",
    });
  });
});

describe("account identification", () => {
  beforeEach(() => {
    identify.mockClear();
    capture.mockClear();
    localStorage.clear();
    document.cookie = "superplane_initial_utm=; Max-Age=0; Path=/";
    stubFetch(mockAccount);
  });

  afterEach(() => {
    localStorage.clear();
    document.cookie = "superplane_initial_utm=; Max-Age=0; Path=/";
    window.history.replaceState({}, "", "/");
    delete (window as Window & { SUPERPLANE_POSTHOG_KEY?: string }).SUPERPLANE_POSTHOG_KEY;
    vi.unstubAllGlobals();
  });

  it("calls identify with account data when account loads", async () => {
    render(
      <AccountProvider>
        <div />
      </AccountProvider>,
    );
    await waitFor(() => {
      expect(identify).toHaveBeenCalledWith("user-123", {
        email: "john@example.com",
        name: "John Doe",
        installation_admin: false,
      });
    });
  });

  it("does not call identify when impersonation is active", async () => {
    const fetchMock = stubFetch({ ...mockAccount, impersonation: { active: true } });
    render(
      <AccountProvider>
        <div />
      </AccountProvider>,
    );
    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    expect(identify).not.toHaveBeenCalled();
  });

  it("captures signup product update preference on account load", async () => {
    confirmSignupAnalyticsPreference({
      email: mockAccount.email,
      productUpdatesOptIn: false,
    });

    render(
      <AccountProvider>
        <div />
      </AccountProvider>,
    );

    await waitFor(() => {
      expect(identify).toHaveBeenCalledWith("user-123", {
        email: "john@example.com",
        name: "John Doe",
        installation_admin: false,
        product_updates_opt_in: false,
      });
    });

    expect(capture).toHaveBeenCalledWith("auth:signup", {
      product_updates_opt_in: false,
      $set: {
        product_updates_opt_in: false,
      },
    });
  });

  it("captures unconfirmed signup preference when redirect marks account as created", async () => {
    window.history.replaceState({}, "", "/org-123?auth_signup_result=created");
    savePendingSignupAnalyticsPreference({
      productUpdatesOptIn: true,
    });

    render(
      <AccountProvider>
        <div />
      </AccountProvider>,
    );

    await waitFor(() => {
      expect(capture).toHaveBeenCalledWith("auth:signup", {
        product_updates_opt_in: true,
        $set: {
          product_updates_opt_in: true,
        },
      });
    });
  });

  it("clears unconfirmed signup preference when redirect marks account as existing", async () => {
    window.history.replaceState({}, "", "/org-123?auth_signup_result=existing");
    savePendingSignupAnalyticsPreference({
      productUpdatesOptIn: true,
    });

    render(
      <AccountProvider>
        <div />
      </AccountProvider>,
    );

    await waitFor(() => {
      expect(identify).toHaveBeenCalledWith("user-123", {
        email: "john@example.com",
        name: "John Doe",
        installation_admin: false,
      });
    });

    expect(capture).not.toHaveBeenCalledWith("auth:signup", expect.anything());
  });
});

describe("logout", () => {
  beforeEach(() => {
    reset.mockClear();
    stubFetch(mockAccount);
    Object.defineProperty(window, "location", {
      value: { href: "" },
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("calls reset when Sign Out is clicked", async () => {
    const user = userEvent.setup();
    render(
      <AccountProvider>
        <OrganizationMenuButton organizationId="org-123" />
      </AccountProvider>,
    );

    await user.click(screen.getByLabelText("Open organization menu"));
    await user.click(screen.getByText("Sign Out"));
    expect(reset).toHaveBeenCalledOnce();
  });
});
