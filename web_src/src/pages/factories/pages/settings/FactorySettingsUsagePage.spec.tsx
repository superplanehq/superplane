import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsUsagePage } from "./FactorySettingsUsagePage";

const mutateAsync = vi.fn();

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/hooks/useFactoryUsage", () => ({
  useFactoryUsage: () => ({ data: {}, isLoading: false, error: null }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useUpdateFactory: () => ({
    mutateAsync,
    isPending: false,
  }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderPage(factory = REFUND_FACTORY) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={["/settings/usage"]}>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{ organizationId: "org-1", factoryId: factory.id ?? "", factory }}
          >
            <FactorySettingsUsagePage />
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsUsagePage hosted spend limit", () => {
  beforeEach(() => {
    mutateAsync.mockReset();
    mutateAsync.mockResolvedValue({});
  });

  it("does not render the Save button while No limit is on", () => {
    renderPage();

    expect(screen.getByLabelText("No limit")).toBeChecked();
    expect(screen.queryByRole("button", { name: "Save hosted spend limit" })).not.toBeInTheDocument();
  });

  it("does not save a zero cap when No limit is turned off", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByLabelText("No limit"));

    const amount = screen.getByLabelText("Limit in USD");
    expect(amount).toHaveValue("");
    expect(screen.getByRole("button", { name: "Save hosted spend limit" })).toBeDisabled();

    await user.type(amount, "50");
    await user.click(screen.getByRole("button", { name: "Save hosted spend limit" }));

    expect(mutateAsync).toHaveBeenCalledWith({ hostedSpendBudgetCents: 5000 });
  });

  it("saves an existing limit without seeding zero", async () => {
    const user = userEvent.setup();
    renderPage({ ...REFUND_FACTORY, hostedSpendBudgetCents: "2500" });

    expect(screen.getByLabelText("Limit in USD")).toHaveValue("25.00");
    await user.click(screen.getByRole("button", { name: "Save hosted spend limit" }));
    expect(mutateAsync).toHaveBeenCalledWith({ hostedSpendBudgetCents: 2500 });
  });

  it("clears a saved limit when No limit is turned back on", async () => {
    const user = userEvent.setup();
    renderPage({ ...REFUND_FACTORY, hostedSpendBudgetCents: "2500" });

    await user.click(screen.getByLabelText("No limit"));

    expect(mutateAsync).toHaveBeenCalledWith({ hostedSpendBudgetCents: null });
    expect(screen.queryByRole("button", { name: "Save hosted spend limit" })).not.toBeInTheDocument();
  });
});
