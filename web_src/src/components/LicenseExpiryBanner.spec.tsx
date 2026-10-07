import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";

import LicenseExpiryBanner from "./LicenseExpiryBanner";

const { accountRef } = vi.hoisted(() => ({
  accountRef: { current: null as Record<string, unknown> | null },
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: accountRef.current }),
}));

const expiringLicense = {
  edition: "enterprise",
  features: ["groups"],
  state: "active",
  expires_at: new Date(Date.now() + 5 * 24 * 60 * 60 * 1000).toISOString(),
};

function renderBanner() {
  const wrapper = ({ children }: { children: ReactNode }) => createElement(MemoryRouter, null, children);
  return render(createElement(LicenseExpiryBanner), { wrapper });
}

describe("LicenseExpiryBanner", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("shows the expiry notice to an installation administrator", () => {
    accountRef.current = { installation_admin: true, license: expiringLicense };
    renderBanner();
    expect(screen.getByTestId("license-expiry-banner")).toBeTruthy();
  });

  it("stays hidden when the installation hides the expiry banner", () => {
    accountRef.current = {
      installation_admin: true,
      license: { ...expiringLicense, hide_expiry_banner: true },
    };
    renderBanner();
    expect(screen.queryByTestId("license-expiry-banner")).toBeNull();
  });
});
