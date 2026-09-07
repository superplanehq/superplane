import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsGeneralPage } from "./FactorySettingsGeneralPage";

const mutateAsync = vi.fn();
let canUpdate = true;

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1" } }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useUpdateFactory: () => ({ mutateAsync, isPending: false }),
  useDeleteFactory: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canUpdate, isLoading: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPage() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{
              organizationId: "org-1",
              factoryId: REFUND_FACTORY.id ?? "factory-1",
              factory: REFUND_FACTORY,
            }}
          >
            <FactorySettingsGeneralPage />
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsGeneralPage", () => {
  beforeEach(() => {
    canUpdate = true;
    mutateAsync.mockReset();
    mutateAsync.mockResolvedValue({});
  });

  it("shows a character avatar, name, and slug in a profile-style card", () => {
    renderPage();

    expect(screen.getByTestId("factory-settings-workspace-avatar")).toHaveTextContent("S");
    expect(screen.getByTestId("factory-settings-name")).toHaveValue("Semaphore");
    expect(screen.getByLabelText("Slug")).toHaveValue("RF");
    expect(screen.getByTestId("factory-settings-key")).toHaveValue("RF");
    expect(screen.queryByLabelText("Workspace key")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Description")).not.toBeInTheDocument();
    expect(screen.queryByTestId("factory-settings-description")).not.toBeInTheDocument();
  });

  it("saves name and slug without sending a description", async () => {
    const user = userEvent.setup();
    renderPage();

    const save = screen.getByTestId("factory-settings-save");
    expect(save).toBeDisabled();

    await user.clear(screen.getByTestId("factory-settings-name"));
    await user.type(screen.getByTestId("factory-settings-name"), "Refunds");
    expect(screen.getByTestId("factory-settings-workspace-avatar")).toHaveTextContent("R");
    expect(save).toBeEnabled();

    await user.click(save);
    expect(mutateAsync).toHaveBeenCalledWith({ name: "Refunds" });
    expect(mutateAsync.mock.calls[0][0]).not.toHaveProperty("description");
  });
});
