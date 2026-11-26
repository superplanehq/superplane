import { fireEvent, render, screen } from "@testing-library/react";
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
  past_due_grace_days: 0,
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

const mockInstallationSettingsFetch = (
  settings: typeof installationSettingsResponse | null = installationSettingsResponse,
) => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/llm-settings")) return Response.json(llmSettingsResponse);
      if (url.includes("/built-in-automations")) {
        return Response.json({ automations: [{ id: "intake:github-issues", name: "GitHub issues" }] });
      }
      if (url === "/admin/api/installation/network-settings") {
        return settings === null
          ? new Response("Failed to load installation settings", { status: 500 })
          : Response.json(settings);
      }
      throw new Error(`Unexpected fetch: ${url}`);
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

describe("SMTP Save state", () => {
  const savedSMTPSettings = {
    ...installationSettingsResponse,
    smtp_enabled: true,
    smtp_host: "mail.example.com",
    smtp_port: 587,
    smtp_username: "mailer",
    smtp_from_name: "Notifications",
    smtp_from_email: "notifications@example.com",
    smtp_password_configured: true,
  };

  const openEmailSettings = async (settings: typeof installationSettingsResponse | null) => {
    const user = userEvent.setup();
    mockInstallationSettingsFetch(settings);
    renderInstallationSettings();
    await user.click(await screen.findByRole("tab", { name: "Email" }));
    return user;
  };

  it("keeps Save disabled after a failed load and an SMTP toggle", async () => {
    const user = await openEmailSettings(null);
    const save = screen.getByTestId("installation-smtp-save");
    expect(save).toBeDisabled();
    const toggle = screen.getByTestId("installation-smtp-switch");
    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-checked", "true");
    expect(save).toBeDisabled();
  });

  it.each([false, true])("detects and reverts the SMTP toggle with saved enabled=%s", async (enabled) => {
    const user = await openEmailSettings({ ...savedSMTPSettings, smtp_enabled: enabled });
    const save = screen.getByTestId("installation-smtp-save");
    const toggle = screen.getByTestId("installation-smtp-switch");
    expect(save).toBeDisabled();
    await user.click(toggle);
    expect(save).toBeEnabled();
    await user.click(toggle);
    expect(save).toBeDisabled();
  });

  it.each([
    ["host", "mail.example.com", "other.example.com"],
    ["username", "mailer", "other-user"],
    ["from name", "Notifications", "Other sender"],
    ["from email", "notifications@example.com", "other@example.com"],
  ])("normalizes whitespace and detects edits to SMTP %s", async (_field, saved, changed) => {
    await openEmailSettings(savedSMTPSettings);
    const save = screen.getByTestId("installation-smtp-save");
    const input = screen.getByDisplayValue(saved);
    expect(save).toBeDisabled();
    fireEvent.change(input, { target: { value: `  ${saved}  ` } });
    expect(save).toBeDisabled();
    fireEvent.change(input, { target: { value: changed } });
    expect(save).toBeEnabled();
    fireEvent.change(input, { target: { value: saved } });
    expect(save).toBeDisabled();
  });

  it.each([
    [587, [" 0587 ", "587.0"]],
    [0, ["", "   ", "0"]],
  ] as const)("compares SMTP port numerically with saved port=%s", async (port, equivalents) => {
    await openEmailSettings({ ...savedSMTPSettings, smtp_port: port });
    const save = screen.getByTestId("installation-smtp-save");
    const input = screen.getByPlaceholderText("587");
    expect(save).toBeDisabled();
    for (const value of equivalents) {
      fireEvent.change(input, { target: { value } });
      expect(save).toBeDisabled();
    }
    fireEvent.change(input, { target: { value: "25" } });
    expect(save).toBeEnabled();
    fireEvent.change(input, { target: { value: String(port) } });
    expect(save).toBeDisabled();
  });

  it.each(["new-password", "   "])("detects and clears a non-empty password %j", async (password) => {
    await openEmailSettings(savedSMTPSettings);
    const save = screen.getByTestId("installation-smtp-save");
    const input = screen.getByTestId("installation-smtp-password");
    expect(save).toBeDisabled();
    fireEvent.change(input, { target: { value: password } });
    expect(save).toBeEnabled();
    fireEvent.change(input, { target: { value: "" } });
    expect(save).toBeDisabled();
  });

  it("detects and reverts a TLS-only edit when SMTP is enabled", async () => {
    const user = await openEmailSettings(savedSMTPSettings);
    const save = screen.getByTestId("installation-smtp-save");
    const tls = screen.getByTestId("installation-smtp-tls-switch");
    expect(save).toBeDisabled();
    await user.click(tls);
    expect(save).toBeEnabled();
    await user.click(tls);
    expect(save).toBeDisabled();
  });

  it.each([false, true])("ignores TLS differences while SMTP is disabled with saved TLS=%s", async (useTLS) => {
    const user = await openEmailSettings({ ...savedSMTPSettings, smtp_enabled: false, smtp_use_tls: useTLS });
    const save = screen.getByTestId("installation-smtp-save");
    const toggle = screen.getByTestId("installation-smtp-switch");
    expect(save).toBeDisabled();
    await user.click(toggle);
    expect(save).toBeEnabled();
    const tls = screen.getByTestId("installation-smtp-tls-switch");
    expect(tls).toHaveAttribute("aria-checked", "true");
    await user.click(tls);
    await user.click(toggle);
    expect(save).toBeDisabled();
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

it("saves the payment grace period and restores it after reload", async () => {
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
  const grace = screen.getByRole("spinbutton", { name: "Payment grace period (days)" });
  expect(grace).toHaveValue(0);
  const save = screen.getByRole("button", { name: "Save payment settings" });
  expect(save).toBeDisabled();
  await user.clear(grace);
  await user.type(grace, "22");
  expect(save).toBeDisabled();
  await user.clear(grace);
  await user.type(grace, "7");
  const duration = screen.getByRole("spinbutton", { name: "Invitation duration (days)" });
  await user.clear(duration);
  await user.type(duration, "14");
  await user.click(save);
  expect(await screen.findByRole("button", { name: "Save payment settings" })).toBeDisabled();
  expect(settings.past_due_grace_days).toBe(7);
  expect(duration).toHaveValue(14);
  expect(settings.invitation_duration_days).toBe(7);
  await user.click(screen.getByRole("button", { name: "Save invitation settings" }));
  expect(await screen.findByRole("button", { name: "Save invitation settings" })).toBeDisabled();
  expect(settings.invitation_duration_days).toBe(14);
  view.unmount();
  renderInstallationSettings();
  await user.click(await screen.findByRole("tab", { name: "Invitations" }));
  expect(screen.getByRole("spinbutton", { name: "Payment grace period (days)" })).toHaveValue(7);
});
