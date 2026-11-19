import { describe, expect, it, vi } from "bun:test";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { TooltipProvider } from "@/ui/tooltip";

import { SsoSettingsView } from "./SsoSettingsView";
import type { SsoSettingsActions, SsoSettingsModel } from "./ssoSettingsTypes";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: null }),
}));

const pendingDomain = {
  id: "domain-1",
  domain: "example.com",
  txtName: "_superplane-verification.example.com",
  txtValue: "superplane-domain-verification=token",
};

const verifiedDomain = { ...pendingDomain, verifiedAt: "2026-11-14T00:00:00Z" };

const draftConnection = {
  id: "conn",
  status: "STATUS_DRAFT" as const,
  displayName: "Okta",
  preset: "PRESET_OKTA" as const,
  issuer: "https://example.okta.com",
  clientId: "client-1",
  clientSecretSet: true,
  redirectUri: "https://app.example.com/callback",
  scimBaseUrl: "https://app.example.com/scim/v2/conn",
};

function renderSettings(model: SsoSettingsModel, actions: Partial<SsoSettingsActions> = {}) {
  const calls = {
    onAddDomain: vi.fn(),
    onVerifyDomain: vi.fn(),
    onDeleteDomain: vi.fn(),
    onSave: vi.fn(),
    onTest: vi.fn(),
    onActivate: vi.fn(),
    onDisable: vi.fn(),
    onDelete: vi.fn(),
    onCreateToken: vi.fn(),
    onRevokeToken: vi.fn(),
    onDismissToken: vi.fn(),
    ...actions,
  };
  render(
    <TooltipProvider>
      <SsoSettingsView model={model} actions={calls} />
    </TooltipProvider>,
  );
  return calls;
}

describe("SsoSettingsView", () => {
  it("shows the locked state when the license is absent and no connection exists", () => {
    renderSettings({ licensed: false, domains: [], tokens: [] });
    expect(screen.getByTestId("sso-locked")).toHaveTextContent("Single Sign-On needs a SuperPlane Enterprise license.");
    expect(screen.getByText("Enterprise")).toBeInTheDocument();
    expect(screen.queryByText("Email domains")).not.toBeInTheDocument();
  });

  it("tells the admin when Single Sign-On appears on the login page", () => {
    renderSettings({ licensed: true, domains: [], tokens: [] });
    expect(
      screen.getByText("Members see Single Sign-On on the login page after you activate the connection."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create connection" })).toBeEnabled();
    expect(screen.getAllByText("Create the connection first.")).toHaveLength(3);
    expect(screen.queryByTestId("sso-provider-urls")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sso-provider-values")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sso-scim")).not.toBeInTheDocument();
  });

  it("adds, verifies, and deletes a domain", async () => {
    const user = userEvent.setup();
    const calls = renderSettings({
      licensed: true,
      domains: [pendingDomain],
      tokens: [],
      connection: draftConnection,
    });
    await user.type(screen.getByLabelText("Domain"), "other.example");
    await user.click(screen.getByRole("button", { name: "Add domain" }));
    expect(calls.onAddDomain).toHaveBeenCalledWith("other.example");

    await user.click(screen.getByRole("button", { name: "Verify" }));
    expect(calls.onVerifyDomain).toHaveBeenCalledWith("domain-1");

    await user.click(screen.getByRole("button", { name: "Delete example.com" }));
    await user.click(screen.getByRole("button", { name: "Delete domain" }));
    expect(calls.onDeleteDomain).toHaveBeenCalledWith("domain-1");
  });

  it("keeps a verification error in the domain row", () => {
    renderSettings({
      licensed: true,
      domains: [pendingDomain],
      tokens: [],
      domainError: { id: "domain-1", message: "The TXT record was not found." },
    });
    expect(screen.getByText("The TXT record was not found.")).toBeInTheDocument();
    expect(screen.getByText("Add this TXT record at your DNS provider. Then select Verify.")).toBeInTheDocument();
  });

  it("creates a connection before the provider values exist", async () => {
    const user = userEvent.setup();
    const calls = renderSettings({ licensed: true, domains: [], tokens: [] });
    await user.click(screen.getByRole("button", { name: "Create connection" }));
    expect(calls.onSave).toHaveBeenCalledWith(
      expect.objectContaining({ displayName: "Okta", issuer: "", clientId: "", clientSecret: "" }),
    );
  });

  it("saves the provider values on the identity provider card", async () => {
    const user = userEvent.setup();
    const calls = renderSettings({
      licensed: true,
      domains: [],
      tokens: [],
      connection: { ...draftConnection, issuer: "", clientId: "", clientSecretSet: false },
    });
    expect(screen.getByTestId("sso-provider-values")).toBeInTheDocument();
    expect(screen.getByTestId("sso-identity-save")).toBeDisabled();
    await user.type(screen.getByLabelText("Issuer URL"), "https://example.okta.com");
    expect(screen.getByTestId("sso-identity-save")).toBeDisabled();
    await user.click(screen.getByTestId("sso-provider-save"));
    expect(calls.onSave).toHaveBeenCalledWith(expect.objectContaining({ issuer: "https://example.okta.com" }));
  });

  it("saves the provider name only after the display name changes", async () => {
    const user = userEvent.setup();
    const calls = renderSettings({
      licensed: true,
      domains: [],
      tokens: [],
      connection: draftConnection,
    });
    expect(screen.getByTestId("sso-identity-save")).toBeDisabled();
    expect(screen.getByTestId("sso-provider-save")).toBeDisabled();
    await user.clear(screen.getByLabelText("Display name"));
    await user.type(screen.getByLabelText("Display name"), "Company Okta");
    expect(screen.getByTestId("sso-provider-save")).toBeDisabled();
    await user.click(screen.getByTestId("sso-identity-save"));
    expect(calls.onSave).toHaveBeenCalledWith(expect.objectContaining({ displayName: "Company Okta" }));
  });

  it("disables the test until the provider values are saved", () => {
    renderSettings({
      licensed: true,
      domains: [verifiedDomain],
      tokens: [],
      connection: { ...draftConnection, issuer: "", clientId: "", clientSecretSet: false },
    });
    expect(screen.getByRole("button", { name: "Test connection" })).toBeDisabled();
    expect(screen.getByText("Add the issuer URL, client ID and client secret first.")).toBeInTheDocument();
  });

  it("explains why activation is disabled", () => {
    const { unmount } = render(
      <TooltipProvider>
        <SsoSettingsView
          model={{ licensed: true, domains: [], tokens: [], connection: draftConnection }}
          actions={noopActions()}
        />
      </TooltipProvider>,
    );
    expect(screen.getByText("Verify an email domain first.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Activate" })).toBeDisabled();
    unmount();

    renderSettings({
      licensed: true,
      domains: [verifiedDomain],
      tokens: [],
      connection: draftConnection,
    });
    expect(screen.getByText("Pass a connection test first.")).toBeInTheDocument();
  });

  it("starts a connection test", async () => {
    const user = userEvent.setup();
    const calls = renderSettings({
      licensed: true,
      domains: [verifiedDomain],
      tokens: [],
      connection: draftConnection,
    });
    await user.click(screen.getByRole("button", { name: "Test connection" }));
    expect(calls.onTest).toHaveBeenCalled();
  });

  it("enables activation after a successful test and a verified domain", () => {
    renderSettings({
      licensed: true,
      domains: [verifiedDomain],
      tokens: [],
      connection: {
        ...draftConnection,
        lastTestedAt: "2026-11-14T12:00:00Z",
        lastTest: { ok: true, email: "ada@example.com" },
      },
    });
    expect(screen.getByText(/Last test passed on/)).toHaveTextContent("ada@example.com");
    expect(screen.getByRole("button", { name: "Activate" })).toBeEnabled();
  });

  it("confirms required enforcement before it selects that option", async () => {
    const user = userEvent.setup();
    const calls = renderSettings({
      licensed: true,
      domains: [verifiedDomain],
      tokens: [],
      connection: draftConnection,
    });
    await user.click(screen.getByRole("radio", { name: /Required/ }));
    expect(screen.getByRole("dialog")).toHaveTextContent("Require Single Sign-On");
    await user.click(screen.getByRole("button", { name: "Require Single Sign-On" }));
    expect(screen.getByRole("radio", { name: /Required/ })).toBeChecked();
    await user.click(screen.getByTestId("sso-policy-save"));
    expect(calls.onSave).toHaveBeenCalledWith(expect.objectContaining({ enforcement: "ENFORCEMENT_REQUIRED" }));
  });

  it("shows the SCIM token once", () => {
    renderSettings({
      licensed: true,
      domains: [verifiedDomain],
      tokens: [{ id: "token-1", description: "Okta", createdAt: "2026-11-14T00:00:00Z" }],
      connection: { ...draftConnection, status: "STATUS_ACTIVE" },
      revealedToken: "scim-token-secret",
    });
    expect(screen.getByRole("dialog")).toHaveTextContent("Copy this token now. You cannot see it again.");
    expect(screen.getByDisplayValue("scim-token-secret")).toBeInTheDocument();
  });

  it("keeps disable and delete available when the license is absent", () => {
    renderSettings({
      licensed: false,
      domains: [verifiedDomain],
      tokens: [{ id: "token-1", description: "Okta" }],
      connection: { ...draftConnection, status: "STATUS_ACTIVE", lastTest: { ok: true } },
    });
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Disable" })).toBeEnabled();
    expect(screen.queryByRole("heading", { name: "Setup" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Delete connection" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Revoke" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Activate" })).not.toBeInTheDocument();
  });

  it("shows a stored test failure", () => {
    renderSettings({
      licensed: true,
      domains: [],
      tokens: [],
      connection: {
        ...draftConnection,
        lastTestedAt: "2026-11-14T12:00:00Z",
        lastTest: { ok: false, error: "The issuer did not respond." },
      },
    });
    expect(screen.getByText(/Last test failed on/)).toHaveTextContent("The issuer did not respond.");
  });

  it("disables changes when the user cannot update SSO", () => {
    renderSettings({
      licensed: true,
      canUpdate: false,
      domains: [verifiedDomain],
      tokens: [{ id: "token-1", description: "Okta" }],
      connection: { ...draftConnection, status: "STATUS_ACTIVE" },
    });
    expect(screen.getByRole("button", { name: "Disable" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Test connection" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Create token" })).toBeDisabled();
  });
});

function noopActions(): SsoSettingsActions {
  return {
    onAddDomain: vi.fn(),
    onVerifyDomain: vi.fn(),
    onDeleteDomain: vi.fn(),
    onSave: vi.fn(),
    onTest: vi.fn(),
    onActivate: vi.fn(),
    onDisable: vi.fn(),
    onDelete: vi.fn(),
    onCreateToken: vi.fn(),
    onRevokeToken: vi.fn(),
    onDismissToken: vi.fn(),
  };
}
