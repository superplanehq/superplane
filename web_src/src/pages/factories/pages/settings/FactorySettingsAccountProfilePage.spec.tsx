import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { AccountGitHubPrAuthorization } from "@/contexts/accountContextState";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { FactorySettingsAccountProfilePage } from "./FactorySettingsAccountProfilePage";

const refreshAccount = vi.fn(async () => undefined);
const showSuccessToast = vi.fn();
const showErrorToast = vi.fn();
let githubPrAuthorization: AccountGitHubPrAuthorization | undefined;

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: {
      id: "account-1",
      name: "Igor",
      email: "igor@superplane.com",
      avatar_url: "",
      installation_admin: false,
      has_password: true,
      providers: [],
      linked_accounts: [],
      github_pr_authorization: githubPrAuthorization,
      organizations_pending_deletion: [],
    },
    refreshAccount,
  }),
}));

vi.mock("./DeleteAccountDangerZone", () => ({
  DeleteAccountDangerZone: () => null,
}));

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

const githubPrSwitchName = "Create pull requests with my GitHub account";

describe("FactorySettingsAccountProfilePage associated accounts", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    githubPrAuthorization = undefined;
  });

  it("shows Associated accounts for GitHub PR credit", () => {
    renderPage();

    expect(screen.getByTestId("account-redesign-associated-accounts")).toHaveTextContent(
      "Link accounts from other services. Each account has its own use. This does not change how you sign in.",
    );
    expect(screen.getByTestId("account-redesign-associated-github")).toHaveTextContent(
      "Velocity uses your GitHub account to credit your pull requests.",
    );
    expect(screen.getByTestId("account-redesign-associated-bitbucket")).toHaveTextContent(
      "Velocity uses your Bitbucket account to credit your pull requests.",
    );
    expect(screen.getByRole("button", { name: "Link GitHub" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Link Bitbucket" })).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-sso-github")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Sign in with GitHub" })).not.toBeInTheDocument();
  });

  it("reports when another account already uses the GitHub link", async () => {
    renderPage("/settings/account/profile?auth_error=linked_account_in_use");

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith(
        "Another member in one of your organizations already uses this GitHub account.",
      );
    });
  });

  it("confirms a completed GitHub link and reloads the account", async () => {
    renderPage("/settings/account/profile?linked_account=linked");

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("GitHub account linked.");
    });
    expect(refreshAccount).toHaveBeenCalled();
  });
});

describe("FactorySettingsAccountProfilePage GitHub pull request authorization", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    githubPrAuthorization = undefined;
  });

  it("disables the option when the GitHub App has no client credentials", () => {
    githubPrAuthorization = { available: false, enabled: false, needs_reauthorization: false };
    renderPage();

    const toggle = screen.getByRole("switch", { name: githubPrSwitchName });
    expect(toggle).toBeDisabled();
    expect(toggle).not.toBeChecked();
    expect(screen.getByTestId("account-redesign-github-pr-authorization")).toHaveTextContent(
      "An administrator must configure the SuperPlane GitHub App before you can use this option.",
    );
  });

  it("shows the GitHub login when the option is on", () => {
    githubPrAuthorization = { available: true, enabled: true, login: "octocat", needs_reauthorization: false };
    renderPage();

    expect(screen.getByRole("switch", { name: githubPrSwitchName })).toBeChecked();
    expect(screen.getByTestId("account-redesign-github-pr-authorization")).toHaveTextContent(
      "SuperPlane opens implementation pull requests as octocat",
    );
  });

  it("asks for a new authorization when the grant stopped working", () => {
    githubPrAuthorization = { available: true, enabled: false, login: "octocat", needs_reauthorization: true };
    renderPage();

    expect(screen.getByRole("switch", { name: githubPrSwitchName })).not.toBeChecked();
    expect(screen.getByTestId("account-redesign-github-pr-authorization")).toHaveTextContent(
      "Your GitHub authorization stopped working. Turn this on again to authorize SuperPlane.",
    );
  });

  it("turns the option off and reloads the account", async () => {
    githubPrAuthorization = { available: true, enabled: true, login: "octocat", needs_reauthorization: false };
    const requests: Array<{ url: string; method?: string }> = [];
    const currentFetch = globalThis.fetch;
    globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
      requests.push({ url: String(input), method: init?.method });
      return new Response(null, { status: 204 });
    }) as typeof fetch;

    try {
      renderPage();
      fireEvent.click(screen.getByRole("switch", { name: githubPrSwitchName }));

      await waitFor(() => {
        expect(refreshAccount).toHaveBeenCalled();
      });
      expect(requests).toEqual([{ url: "/account/github-authorization", method: "DELETE" }]);
    } finally {
      globalThis.fetch = currentFetch;
    }
  });

  it("confirms a completed GitHub authorization", async () => {
    renderPage("/settings/account/profile?github_pr_authorization=enabled&provider=github");

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith(
        "SuperPlane now opens your pull requests with your GitHub account.",
      );
    });
    expect(refreshAccount).toHaveBeenCalled();
  });
});
