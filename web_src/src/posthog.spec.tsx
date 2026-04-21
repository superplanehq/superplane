import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { init, identify, capture, reset } = vi.hoisted(() => ({
  init: vi.fn(),
  identify: vi.fn(),
  capture: vi.fn(),
  reset: vi.fn(),
}));

vi.mock("posthog-js", () => ({
  default: { init, identify, capture, reset },
}));

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: ReactNode; to: string }) => <a href={to}>{children}</a>,
  useNavigate: () => vi.fn(),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { metadata: { name: "Acme Corp" } } }),
  useOrganizationUsage: () => ({ data: null, error: null }),
}));

vi.mock("@/contexts/PermissionsContext", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/lib/env", () => ({
  isUsagePageForced: () => false,
}));

import { AccountProvider } from "@/contexts/AccountContext";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";

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
    vi.resetModules();
  });

  it("calls init when SUPERPLANE_POSTHOG_KEY is set", async () => {
    (window as Window & { SUPERPLANE_POSTHOG_KEY?: string }).SUPERPLANE_POSTHOG_KEY = "test-key";
    await import("@/posthog");
    expect(init).toHaveBeenCalledWith(
      "test-key",
      expect.objectContaining({ autocapture: false, capture_pageview: false }),
    );
  });

  it("does not call init when SUPERPLANE_POSTHOG_KEY is not set", async () => {
    delete (window as Window & { SUPERPLANE_POSTHOG_KEY?: string }).SUPERPLANE_POSTHOG_KEY;
    await import("@/posthog");
    expect(init).not.toHaveBeenCalled();
  });
});

describe("account identification", () => {
  beforeEach(() => {
    identify.mockClear();
    stubFetch(mockAccount);
  });

  afterEach(() => {
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
        is_internal: false,
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
