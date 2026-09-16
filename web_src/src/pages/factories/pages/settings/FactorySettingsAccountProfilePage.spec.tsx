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

describe("FactorySettingsAccountProfilePage GitHub sign-in", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("credits pull requests through the GitHub sign-in row", () => {
    renderPage();

    expect(screen.queryByTestId("account-redesign-velocity-github")).not.toBeInTheDocument();
    expect(screen.getByTestId("account-redesign-sso-github")).toHaveTextContent(
      "Used to sign in and to credit pull requests. This identity can also sign in to another SuperPlane account.",
    );
    expect(screen.getByTestId("account-redesign-signin")).toHaveTextContent(
      "A GitHub identity can also sign in to another SuperPlane account.",
    );
    expect(screen.getByRole("button", { name: "Sign in with GitHub" })).toBeInTheDocument();
  });

  it("reports an identity another account already uses", async () => {
    renderPage("/settings/account/profile?auth_error=linked_account_in_use");

    await waitFor(() => {
      expect(showErrorToast).toHaveBeenCalledWith("Another SuperPlane account already uses this GitHub account.");
    });
  });

  it("confirms a completed GitHub sign-in link and reloads the account", async () => {
    renderPage("/settings/account/profile?auth_link_result=connected&provider=github");

    await waitFor(() => {
      expect(showSuccessToast).toHaveBeenCalledWith("GitHub connected.");
    });
    expect(refreshAccount).toHaveBeenCalled();
  });
});
