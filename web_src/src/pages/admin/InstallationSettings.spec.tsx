import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import InstallationSettings from "./InstallationSettings";

type SignupWaitlistWindow = Window & {
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID?: string;
  SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID?: string;
};

const waitlistWindow = window as SignupWaitlistWindow;

const installationSettingsResponse = {
  allow_private_network_access: false,
  signups_enabled: false,
  invitation_duration_days: 7,
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

const llmSettingsResponse = {
  welcome_grant_cents: 5000,
  markup_bps: 2000,
  warning_threshold_bps: 2000,
  default_hosted_provider: "",
  default_hosted_model: "",
  providers: [
    {
      provider: "anthropic",
      enabled: false,
      api_key_configured: false,
      management_key_configured: false,
      base_url: "",
      allowed_models: [],
    },
    {
      provider: "openai",
      enabled: false,
      api_key_configured: false,
      management_key_configured: false,
      base_url: "",
      allowed_models: [],
    },
    {
      provider: "openrouter",
      enabled: false,
      api_key_configured: false,
      management_key_configured: false,
      base_url: "",
      allowed_models: [],
    },
  ],
};

const mockInstallationSettingsFetch = () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      const body = url.includes("/llm-settings")
        ? llmSettingsResponse
        : url.includes("/built-in-automations")
          ? { automations: [{ id: "intake:github-issues", name: "GitHub issues" }] }
          : installationSettingsResponse;
      return new Response(JSON.stringify(body), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }),
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

    expect(await screen.findByRole("heading", { name: "Reset one automation" })).toBeInTheDocument();
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Automations",
      "Email",
      "Invitations",
      "Hosted LLM",
      "Network",
    ]);
    expect(screen.queryByText(/onboarding templates/i)).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: "Signups" })).not.toBeInTheDocument();
    expect(screen.queryByText("Signup access")).not.toBeInTheDocument();
    expect(screen.queryByText("Public signups")).not.toBeInTheDocument();
    expect(screen.queryByTestId("installation-signups-switch")).not.toBeInTheDocument();
  });

  it("shows one settings section at a time", async () => {
    const user = userEvent.setup();
    mockInstallationSettingsFetch();

    renderInstallationSettings();

    expect(await screen.findByRole("heading", { name: "Reset one automation" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "SMTP configuration" })).not.toBeInTheDocument();
    expect(screen.queryByText("SuperPlane-hosted models")).not.toBeInTheDocument();
    expect(screen.queryByText("Network policy")).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Email" }));
    expect(await screen.findByRole("heading", { name: "SMTP configuration" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Reset one automation" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Hosted LLM" }));
    expect(await screen.findByText("SuperPlane-hosted models")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "SMTP configuration" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Network" }));
    expect(await screen.findByText("Network policy")).toBeInTheDocument();
    expect(screen.queryByText("SuperPlane-hosted models")).not.toBeInTheDocument();
  });

  it("shows signup access settings when waitlist config is complete", async () => {
    const user = userEvent.setup();
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1";
    waitlistWindow.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "form-1";
    mockInstallationSettingsFetch();

    renderInstallationSettings();

    expect((await screen.findAllByRole("tab")).map((tab) => tab.textContent)).toEqual([
      "Automations",
      "Email",
      "Invitations",
      "Hosted LLM",
      "Network",
      "Signups",
    ]);
    await user.click(screen.getByRole("tab", { name: "Signups" }));
    expect(await screen.findByText("Signup access")).toBeInTheDocument();
    expect(screen.getByText("Public signups")).toBeInTheDocument();
    expect(screen.getByTestId("installation-signups-switch")).toBeInTheDocument();
    expect(screen.queryByText("Network policy")).not.toBeInTheDocument();
  });
});

it("saves invitation duration and restores it after reload", async () => {
  const user = userEvent.setup();
  let settings = { ...installationSettingsResponse };
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "PATCH") {
        settings = { ...settings, ...JSON.parse(String(init.body)) };
      }
      return Response.json(settings);
    }),
  );
  const view = renderInstallationSettings();
  await user.click(await screen.findByRole("tab", { name: "Invitations" }));
  const duration = screen.getByRole("spinbutton", { name: "Invitation duration (days)" });
  expect(duration).toHaveValue(7);
  await user.clear(duration);
  expect(screen.getByRole("button", { name: "Save invitation settings" })).toBeDisabled();
  await user.type(duration, "14");
  await user.click(screen.getByRole("button", { name: "Save invitation settings" }));
  expect(await screen.findByRole("button", { name: "Save invitation settings" })).toBeDisabled();
  expect(settings.invitation_duration_days).toBe(14);
  view.unmount();
  renderInstallationSettings();
  await user.click(await screen.findByRole("tab", { name: "Invitations" }));
  expect(screen.getByRole("spinbutton", { name: "Invitation duration (days)" })).toHaveValue(14);
});
