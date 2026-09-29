import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

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

function renderPage(factory = REFUND_FACTORY) {
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

const badgeOnFactory = {
  ...REFUND_FACTORY,
  publicBadgeEnabled: true,
  publicBadgeShowCost: false,
  publicBadgeToken: "badge-token",
};

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

  it("shows the public badge toggle and saves it", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByRole("switch", { name: "Public badge" })).toBeInTheDocument();
    expect(screen.queryByRole("switch", { name: "Show cost per merged PR" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("switch", { name: "Public badge" }));
    expect(mutateAsync).toHaveBeenCalledWith({ publicBadgeEnabled: true });
  });

  it("shows the cost switch only when the badge is on and keeps the snippet stable", async () => {
    const user = userEvent.setup();
    mutateAsync.mockResolvedValue({ publicBadgeToken: "badge-token" });
    renderPage(badgeOnFactory);

    const cost = screen.getByRole("switch", { name: "Show cost per merged PR" });
    expect(cost).not.toBeChecked();

    const snippet = screen.getByTestId("factory-settings-public-badge-markdown");
    expect(snippet).toHaveValue(
      `[![PRs via SuperPlane](${window.location.origin}/api/v1/public/badges/badge-token.svg?period=30&size=small)](${window.location.origin})`,
    );

    await user.click(screen.getByTestId("factory-settings-public-badge-period"));
    await user.click(screen.getByRole("option", { name: "14 days" }));
    expect(snippet).toHaveValue(
      `[![PRs via SuperPlane](${window.location.origin}/api/v1/public/badges/badge-token.svg?period=14&size=small)](${window.location.origin})`,
    );

    await user.click(screen.getByTestId("factory-settings-public-badge-size"));
    await user.click(screen.getByRole("option", { name: "Full width" }));
    const afterSize = (snippet as HTMLInputElement).value;
    expect(afterSize).toContain("period=14&size=wide");

    await user.click(cost);
    expect(mutateAsync).toHaveBeenCalledWith({ publicBadgeShowCost: true });
    expect(snippet).toHaveValue(afterSize);
  });

  it("disables badge controls without update permission", () => {
    canUpdate = false;
    renderPage(badgeOnFactory);

    expect(screen.getByRole("switch", { name: "Public badge" })).toBeDisabled();
    expect(screen.getByRole("switch", { name: "Show cost per merged PR" })).toBeDisabled();
    expect(screen.getByTestId("factory-settings-public-badge-period")).toBeDisabled();
    expect(screen.getByTestId("factory-settings-public-badge-size")).toBeDisabled();
    expect(screen.getByTestId("factory-settings-public-badge-copy")).toBeDisabled();
    expect(screen.getByTestId("factory-settings-public-badge-markdown")).toBeDisabled();
  });
});
