import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { FactorySettingsAccountProfilePage } from "./FactorySettingsAccountProfilePage";

const refreshAccount = vi.fn(async () => undefined);
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
  });

  it("shows Associated accounts for GitHub PR credit", () => {
    renderPage();

    expect(screen.getByTestId("account-redesign-associated-accounts")).toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-associated-github")).toHaveTextContent(
      "Velocity uses this GitHub account to credit your pull requests.",
    );
    expect(screen.getByRole("button", { name: "Link GitHub" })).toBeInTheDocument();
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
