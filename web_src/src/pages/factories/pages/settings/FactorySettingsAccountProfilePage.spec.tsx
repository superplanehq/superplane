import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { AccountLinkedAccount } from "@/contexts/accountContextState";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as AccountSettings from "@/lib/accountSettings";
import { TooltipProvider } from "@/ui/tooltip";

import { FactorySettingsAccountProfilePage } from "./FactorySettingsAccountProfilePage";

const accountState: {
  linked: AccountLinkedAccount[];
  providers: Array<{ provider: string; username?: string; email?: string }>;
} = {
  linked: [],
  providers: [],
};
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
      providers: accountState.providers,
      linked_accounts: accountState.linked,
    },
    refreshAccount,
  }),
}));

vi.mock("./DeleteAccountDangerZone", () => ({
  DeleteAccountDangerZone: () => null,
}));

vi.mock("@/lib/accountSettings", async (importOriginal) => {
  const actual = await importOriginal<typeof AccountSettings>();
  return {
    ...actual,
    disconnectLinkedAccount: (provider: string) => disconnectLinkedAccount(provider),
    linkedAccountConnectHref: (provider: string, redirect: string) =>
      `/auth/${provider}?intent=connect&redirect=${encodeURIComponent(redirect)}`,
    ssoLinkHref: (provider: string, redirect: string) =>
      `/auth/${provider}?intent=link&redirect=${encodeURIComponent(redirect)}`,
  };
});

vi.mock("@/lib/toast", () => ({
  showSuccessToast: (message: string) => showSuccessToast(message),
  showErrorToast: (message: string) => showErrorToast(message),
}));

function renderPage(path = "/settings/account/profile") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <ThemeProvider>
          <TooltipProvider>
            <FactorySettingsAccountProfilePage />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsAccountProfilePage GitHub identity", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    accountState.linked = [];
    accountState.providers = [];
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign, pathname: "/settings/account/profile", search: "" },
    });
  });

  it("offers connect and pull-request credit when the member linked nothing", async () => {
    renderPage();

    const github = screen.getByTestId("account-redesign-sso-github");
    expect(github).toHaveTextContent("Not connected");
    expect(github).toHaveTextContent("Connect GitHub to sign in, or link it to credit pull requests.");
    expect(screen.queryByTestId("account-redesign-velocity-github")).not.toBeInTheDocument();

    await userEvent.click(within(github).getByRole("button", { name: "Link for pull request credit" }));
    expect(assign).toHaveBeenCalledWith("/auth/github?intent=connect&redirect=%2Fsettings%2Faccount%2Fprofile");
  });

  it("starts SSO connect from the empty GitHub row", async () => {
    renderPage();

    await userEvent.click(
      within(screen.getByTestId("account-redesign-sso-github")).getByRole("button", { name: "Connect" }),
    );
    expect(assign).toHaveBeenCalledWith("/auth/github?intent=link&redirect=%2Fsettings%2Faccount%2Fprofile");
  });

  it("shows the linked-only state", () => {
    accountState.linked = [{ provider: "github", username: "shiroyasha" }];
    renderPage();

    const github = screen.getByTestId("account-redesign-sso-github");
    expect(github).toHaveTextContent("Linked as shiroyasha");
    expect(within(github).getByRole("button", { name: "Add as sign-in method" })).toBeInTheDocument();
    expect(within(github).queryByRole("button", { name: "Link for pull request credit" })).not.toBeInTheDocument();
  });

  it("shows SSO-connected GitHub without a second card", () => {
    accountState.providers = [{ provider: "github", username: "shiroyasha" }];
    accountState.linked = [{ provider: "github", username: "shiroyasha" }];
    renderPage();

    const github = screen.getByTestId("account-redesign-sso-github");
    expect(github).toHaveTextContent("Connected as shiroyasha");
    expect(github).toHaveTextContent(
      "You can sign in with this account. SuperPlane also uses it to credit pull requests.",
    );
    expect(within(github).getByRole("button", { name: "Disconnect" })).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-velocity-github")).not.toBeInTheDocument();
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
    renderPage("/settings/account/profile?auth_error=linked_account_in_use");

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith("Another SuperPlane account already uses this GitHub account.");
    });
  });

  it("confirms a completed link and reloads the account", async () => {
    renderPage("/settings/account/profile?linked_account=linked");

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("GitHub account linked.");
    });
    expect(refreshAccount).toHaveBeenCalled();
  });
});
