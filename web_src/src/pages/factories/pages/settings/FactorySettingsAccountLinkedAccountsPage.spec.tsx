import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { FactorySettingsAccountLinkedAccountsPage } from "./FactorySettingsAccountLinkedAccountsPage";

const account = vi.hoisted(() => ({ current: null as Record<string, unknown> | null }));
const refreshAccount = vi.hoisted(() => vi.fn());
const disconnectAccountProvider = vi.hoisted(() => vi.fn());
const showSuccessToast = vi.hoisted(() => vi.fn());
const showErrorToast = vi.hoisted(() => vi.fn());
const assign = vi.hoisted(() => vi.fn());

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: account.current, refreshAccount }),
}));

vi.mock("@/lib/accountSettings", () => ({
  disconnectAccountProvider: (provider: string) => disconnectAccountProvider(provider),
  ssoLinkHref: (provider: string, redirect: string) =>
    `/auth/${provider}?intent=link&redirect=${encodeURIComponent(redirect)}`,
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: (message: string) => showSuccessToast(message),
  showErrorToast: (message: string) => showErrorToast(message),
  showInfoToast: vi.fn(),
}));

const PATH = "/org-1/workspaces/RF/settings/account/linked-accounts";

function renderPage(path = PATH) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <FactorySettingsAccountLinkedAccountsPage />
    </MemoryRouter>,
  );
}

describe("FactorySettingsAccountLinkedAccountsPage", () => {
  beforeEach(() => {
    refreshAccount.mockReset().mockResolvedValue(undefined);
    disconnectAccountProvider.mockReset().mockResolvedValue(undefined);
    showSuccessToast.mockReset();
    showErrorToast.mockReset();
    assign.mockReset();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign, pathname: PATH, search: "" },
    });
  });

  it("sends the member to the GitHub link flow and back to this page", async () => {
    const user = userEvent.setup();
    account.current = { has_password: true, providers: [] };
    renderPage();

    await user.click(screen.getByRole("button", { name: "Connect GitHub" }));

    expect(assign).toHaveBeenCalledWith(`/auth/github?intent=link&redirect=${encodeURIComponent(PATH)}`);
  });

  it("shows the connected GitHub login that Velocity joins on", () => {
    account.current = {
      has_password: true,
      providers: [{ provider: "github", username: "shiroyasha", email: "igisar@gmail.com" }],
    };
    renderPage();

    expect(screen.getByTestId("account-redesign-sso-github")).toHaveTextContent("Connected as shiroyasha");
    expect(screen.getByTestId("account-redesign-sso-google")).toHaveTextContent("Not connected");
  });

  it("disconnects only after the dialog is confirmed", async () => {
    const user = userEvent.setup();
    account.current = {
      has_password: true,
      providers: [{ provider: "github", username: "shiroyasha" }],
    };
    renderPage();

    await user.click(
      within(screen.getByTestId("account-redesign-sso-github")).getByRole("button", { name: "Disconnect" }),
    );
    expect(disconnectAccountProvider).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Disconnect GitHub" }));
    expect(disconnectAccountProvider).toHaveBeenCalledWith("github");
    await waitFor(() => expect(showSuccessToast).toHaveBeenCalledWith("GitHub disconnected."));
  });

  it("keeps the last sign-in method when the account has no password", () => {
    account.current = {
      has_password: false,
      providers: [{ provider: "github", username: "shiroyasha" }],
    };
    renderPage();

    expect(
      within(screen.getByTestId("account-redesign-sso-github")).getByRole("button", { name: "Disconnect" }),
    ).toBeDisabled();
    expect(screen.getByText("Keep at least one sign-in method.")).toBeInTheDocument();
  });

  it("reports a GitHub identity that belongs to another account", async () => {
    account.current = { has_password: true, providers: [] };
    renderPage(`${PATH}?auth_error=signin_method_in_use&provider=github`);

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith(
        "This GitHub account already belongs to another SuperPlane account. Delete that account first.",
      );
    });
  });

  it("refreshes the account after a successful link", async () => {
    account.current = { has_password: true, providers: [] };
    renderPage(`${PATH}?auth_link_result=connected&provider=github`);

    await waitFor(() => expect(showSuccessToast).toHaveBeenCalledWith("GitHub connected."));
    expect(refreshAccount).toHaveBeenCalled();
  });
});
