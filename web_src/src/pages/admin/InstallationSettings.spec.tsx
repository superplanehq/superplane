import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";

import InstallationSettings from "./InstallationSettings";

type SignupWaitlistWindow = Window & {
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID?: string;
};

const waitlistWindow = window as SignupWaitlistWindow;

const installationSettingsResponse = {
  allow_private_network_access: false,
  signups_enabled: false,
  signups_blocked_by_environment: false,
  effective_blocked_http_hosts: [],
  effective_private_ip_ranges: [],
  blocked_http_hosts_overridden: false,
  private_ip_ranges_overridden: false,
  smtp_enabled: false,
  smtp_host: "",
  smtp_port: 0,
  smtp_username: "",
  smtp_from_name: "",
  smtp_from_email: "",
  smtp_use_tls: true,
  smtp_password_configured: false,
};

const mockInstallationSettingsFetch = () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify(installationSettingsResponse), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
};

const renderInstallationSettings = () =>
  render(
    <MemoryRouter initialEntries={["/admin/installation-settings"]}>
      <InstallationSettings />
    </MemoryRouter>,
  );

afterEach(() => {
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID;
  delete waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID;
  vi.unstubAllGlobals();
});

describe("InstallationSettings", () => {
  it("hides signup access settings when waitlist config is incomplete", async () => {
    mockInstallationSettingsFetch();

    renderInstallationSettings();

    expect(await screen.findByText("Network policy")).toBeInTheDocument();
    expect(screen.queryByText("Signup access")).not.toBeInTheDocument();
    expect(screen.queryByText("Public signups")).not.toBeInTheDocument();
    expect(screen.queryByTestId("installation-signups-switch")).not.toBeInTheDocument();
  });

  it("shows signup access settings when waitlist config is complete", async () => {
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "form-1";
    mockInstallationSettingsFetch();

    renderInstallationSettings();

    expect(await screen.findByText("Signup access")).toBeInTheDocument();
    expect(screen.getByText("Public signups")).toBeInTheDocument();
    expect(screen.getByTestId("installation-signups-switch")).toBeInTheDocument();
  });
});
