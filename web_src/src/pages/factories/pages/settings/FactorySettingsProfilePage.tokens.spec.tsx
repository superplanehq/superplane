import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsProfilePage } from "./FactorySettingsProfilePage";

const createToken = vi.fn();
const revokeToken = vi.fn();

const EXISTING_TOKEN = {
  id: "token-1",
  name: "Deploy script",
  createdAt: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
};

vi.mock("@/hooks/useUserTokens", () => ({
  useUserTokens: () => ({ data: [EXISTING_TOKEN], isLoading: false }),
  useCreateUserToken: () => ({ mutateAsync: createToken, isPending: false }),
  useRevokeUserToken: () => ({ mutateAsync: revokeToken, isPending: false }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({
    data: { id: "user-1", name: "Ada Lovelace", email: "ada@example.com" },
    isLoading: false,
    error: null,
  }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { has_password: false } }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPage() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={["/settings/profile"]}>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{ organizationId: "org-1", factoryId: REFUND_FACTORY.id ?? "", factory: REFUND_FACTORY }}
          >
            <FactorySettingsProfilePage />
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsProfilePage API tokens", () => {
  beforeEach(() => {
    createToken.mockReset();
    revokeToken.mockReset();
    revokeToken.mockResolvedValue({});
  });

  it("lists tokens and marks an unused token as never used", () => {
    renderPage();

    const row = screen.getByTestId("user-token-row");
    expect(within(row).getByText("Deploy script")).toBeInTheDocument();
    expect(within(row).getByText("Never")).toBeInTheDocument();
  });

  it("creates a token from the dialog and shows the secret one time", async () => {
    const user = userEvent.setup();
    createToken.mockResolvedValue({ token: { id: "token-2", name: "Release bot" }, plaintext: "plain-secret-value" });
    renderPage();

    await user.click(screen.getByTestId("user-token-create-btn"));
    await user.type(screen.getByLabelText("Token name"), "Release bot");
    await user.click(screen.getByTestId("user-token-create-submit"));

    expect(createToken).toHaveBeenCalledWith({ name: "Release bot" });
    expect(await screen.findByTestId("user-token-reveal-value")).toHaveTextContent("plain-secret-value");

    await user.click(screen.getByTestId("user-token-reveal-done"));
    await waitFor(() => expect(screen.queryByTestId("user-token-reveal-value")).not.toBeInTheDocument());
  });

  it("keeps the token until the revoke dialog is confirmed", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId("user-token-row-menu"));
    await user.click(screen.getByTestId("user-token-revoke-btn"));

    expect(screen.getByText('Revoke "Deploy script"?')).toBeInTheDocument();
    expect(revokeToken).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("user-token-revoke-confirm"));
    expect(revokeToken).toHaveBeenCalledWith("token-1");
  });
});
