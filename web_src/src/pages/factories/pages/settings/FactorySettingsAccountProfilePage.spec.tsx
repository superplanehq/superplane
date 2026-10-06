import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { FactorySettingsAccountProfilePage } from "./FactorySettingsAccountProfilePage";

const refreshAccount = vi.fn(async () => undefined);
const showSuccessToast = vi.fn();
const showErrorToast = vi.fn();
let connectProviders: string[] = [];

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

describe("FactorySettingsAccountProfilePage associated accounts", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    connectProviders = [];
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL) => {
        const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
        if (url.includes("/auth/config")) {
          return Promise.resolve(
            new Response(JSON.stringify({ connectProviders }), {
              status: 200,
              headers: { "Content-Type": "application/json" },
            }),
          );
        }
        return Promise.resolve(new Response("{}", { status: 404 }));
      }),
    );
  });

  it("hides Link Bitbucket when Bitbucket is not configured", async () => {
    renderPage();

    expect(screen.getByRole("button", { name: "Link GitHub" })).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.queryByRole("button", { name: "Link Bitbucket" })).not.toBeInTheDocument();
    });
    expect(screen.queryByTestId("account-redesign-associated-bitbucket")).not.toBeInTheDocument();
  });

  it("shows Associated accounts for GitHub PR credit", async () => {
    connectProviders = ["bitbucket"];
    renderPage();

    expect(screen.getByTestId("account-redesign-associated-accounts")).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-associated-github")).toHaveTextContent(
      "Velocity uses this GitHub account to credit your pull requests.",
    );
    expect(screen.getByRole("button", { name: "Link GitHub" })).toBeInTheDocument();
    expect(await screen.findByRole("button", { name: "Link Bitbucket" })).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-associated-bitbucket")).toHaveTextContent(
      "This link does not change how you sign in.",
    );
    expect(screen.queryByTestId("account-redesign-sso-github")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Sign in with GitHub" })).not.toBeInTheDocument();
  });

  it("starts a Bitbucket link from the current account page", async () => {
    connectProviders = ["bitbucket"];
    const assign = vi.fn();
    const previousAssign = window.location.assign.bind(window.location);
    window.location.assign = assign;
    try {
      const user = userEvent.setup();
      renderPage("/acme/settings/account/profile");
      await user.click(await screen.findByRole("button", { name: "Link Bitbucket" }));
      expect(assign).toHaveBeenCalledWith(
        "/auth/bitbucket?intent=connect&redirect=%2Facme%2Fsettings%2Faccount%2Fprofile",
      );
    } finally {
      window.location.assign = previousAssign;
    }
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
