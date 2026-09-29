import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesFactory } from "@/api-client";
import { TooltipProvider } from "@/ui/tooltip";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsGeneralPage } from "./FactorySettingsGeneralPage";

const mutateAsync = vi.fn();
const setVisibilityMutateAsync = vi.fn();
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
  useSetFactoryVisibility: () => ({ mutateAsync: setVisibilityMutateAsync, isPending: false }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => canUpdate, isLoading: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPage(factory: FactoriesFactory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{
              organizationId: "org-1",
              factoryId: factory.id ?? "factory-1",
              factory,
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
    setVisibilityMutateAsync.mockReset();
    setVisibilityMutateAsync.mockResolvedValue({});
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

  it("asks before it makes the workspace public", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("switch", { name: "Change to public" }));
    expect(screen.getByTestId("factory-settings-visibility-dialog")).toHaveTextContent("Make this workspace public?");
    expect(setVisibilityMutateAsync).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("factory-settings-visibility-cancel"));
    expect(setVisibilityMutateAsync).not.toHaveBeenCalled();

    await user.click(screen.getByRole("switch", { name: "Change to public" }));
    await user.click(screen.getByTestId("factory-settings-visibility-confirm"));
    expect(setVisibilityMutateAsync).toHaveBeenCalledWith(true);
  });

  it("shows the line board link when the workspace is public", () => {
    renderPage({ ...REFUND_FACTORY, public: true });

    const link = screen.getByTestId("factory-settings-visibility-board-link");
    expect(link).toHaveAttribute("href", "/org-1/workspaces/rf/lines/line-plan-and-implement");
    expect(link).toHaveTextContent("/org-1/workspaces/rf/lines/line-plan-and-implement");
  });
});
