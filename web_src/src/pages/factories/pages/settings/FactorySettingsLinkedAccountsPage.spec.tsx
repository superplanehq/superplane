import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AccountLinkedAccount } from "@/contexts/accountContextState";
import { TooltipProvider } from "@/ui/tooltip";

import { FactorySettingsLinkedAccountsPage } from "./FactorySettingsLinkedAccountsPage";

const accountState: { linked: AccountLinkedAccount[] } = { linked: [] };
const refreshAccount = vi.fn(async () => undefined);
const disconnectLinkedAccount = vi.fn(async (_provider: string) => undefined);
const assign = vi.fn();
const showSuccessToast = vi.fn();
const showErrorToast = vi.fn();

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: {
      id: "account-1",
      name: "Igor",
      email: "igor@superplane.com",
      avatar_url: "",
      installation_admin: false,
      has_password: true,
      linked_accounts: accountState.linked,
    },
    refreshAccount,
  }),
}));

vi.mock("@/lib/accountSettings", () => ({
  disconnectLinkedAccount: (provider: string) => disconnectLinkedAccount(provider),
  linkedAccountConnectHref: (provider: string, redirect: string) =>
    `/auth/${provider}?intent=connect&redirect=${encodeURIComponent(redirect)}`,
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: (message: string) => showSuccessToast(message),
  showErrorToast: (message: string) => showErrorToast(message),
}));

function renderPage(path = "/settings/account/linked-accounts") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <TooltipProvider>
        <FactorySettingsLinkedAccountsPage />
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("FactorySettingsLinkedAccountsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    accountState.linked = [];
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign, pathname: "/settings/account/linked-accounts", search: "" },
    });
  });

  it("sends a member who linked nothing to the connect flow", async () => {
    renderPage();

    expect(screen.getByText("Not linked.")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Link GitHub" }));

    expect(assign).toHaveBeenCalledWith("/auth/github?intent=connect&redirect=%2Fsettings%2Faccount%2Flinked-accounts");
  });

  it("shows the linked login", () => {
    accountState.linked = [{ provider: "github", username: "shiroyasha" }];
    renderPage();

    expect(screen.getByText("Linked as shiroyasha.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Link GitHub" })).not.toBeInTheDocument();
  });

  it("removes the link after the member confirms", async () => {
    accountState.linked = [{ provider: "github", username: "shiroyasha" }];
    renderPage();

    await userEvent.click(screen.getByRole("button", { name: "Remove" }));
    await userEvent.click(screen.getByRole("button", { name: "Remove link" }));

    await waitFor(() => {
      expect(disconnectLinkedAccount).toHaveBeenCalledWith("github");
    });
    expect(refreshAccount).toHaveBeenCalled();
    expect(showSuccessToast).toHaveBeenCalledWith("GitHub link removed.");
  });

  it("keeps the link when the member backs out", async () => {
    accountState.linked = [{ provider: "github", username: "shiroyasha" }];
    renderPage();

    await userEvent.click(screen.getByRole("button", { name: "Remove" }));
    await userEvent.click(screen.getByRole("button", { name: "Keep the link" }));

    expect(disconnectLinkedAccount).not.toHaveBeenCalled();
  });

  it("reports an identity another account already uses", async () => {
    renderPage("/settings/account/linked-accounts?auth_error=linked_account_in_use");

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith("Another SuperPlane account already uses this GitHub account.");
    });
  });

  it("confirms a completed link and reloads the account", async () => {
    renderPage("/settings/account/linked-accounts?linked_account=linked");

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("GitHub account linked.");
    });
    expect(refreshAccount).toHaveBeenCalled();
  });
});
