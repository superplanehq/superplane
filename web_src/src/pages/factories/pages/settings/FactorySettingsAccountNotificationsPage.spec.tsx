import type { MeNotificationSettings } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsAccountNotificationsPage } from "./FactorySettingsAccountNotificationsPage";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";

const ALL_SCOPE_SETTINGS: MeNotificationSettings = {
  workspaces: { scope: "WORKSPACE_SCOPE_ALL", eventTypes: [], filters: [] },
};

const NONE_SCOPE_SETTINGS: MeNotificationSettings = {
  workspaces: { scope: "WORKSPACE_SCOPE_NONE", eventTypes: [], filters: [] },
};

const queryState: {
  isPending: boolean;
  isError: boolean;
  settings: MeNotificationSettings | undefined;
} = {
  isPending: false,
  isError: false,
  settings: ALL_SCOPE_SETTINGS,
};

vi.mock("@/hooks/useNotificationSettings", () => ({
  useNotificationSettings: () => ({
    data: queryState.settings,
    isPending: queryState.isPending,
    isError: queryState.isError,
  }),
  useUpdateNotificationSettings: () => ({
    mutateAsync: vi.fn(async (settings: MeNotificationSettings) => settings),
    isPending: false,
  }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactories: () => ({ data: [REFUND_FACTORY] }),
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { email: "ada@superplane.dev" } }),
}));

function renderPage(organizationId = "org-1") {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={["/settings/account/notifications"]}>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{ organizationId, factoryId: REFUND_FACTORY.id ?? "", factory: REFUND_FACTORY }}
          >
            <FactorySettingsAccountNotificationsPage />
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsAccountNotificationsPage", () => {
  it("keeps the form when a refetch fails and cached settings remain", () => {
    queryState.settings = ALL_SCOPE_SETTINGS;
    queryState.isPending = false;
    queryState.isError = true;
    renderPage();

    expect(screen.getByTestId("account-redesign-notifications")).toBeInTheDocument();
    expect(screen.queryByText("Failed to load notification settings.")).not.toBeInTheDocument();
  });

  it("shows an error when the first load has no settings", () => {
    queryState.settings = undefined;
    queryState.isPending = false;
    queryState.isError = true;
    renderPage();

    expect(screen.getByText("Failed to load notification settings.")).toBeInTheDocument();
    expect(screen.queryByTestId("account-redesign-notifications")).not.toBeInTheDocument();
  });

  it("remounts the form when the organization changes", () => {
    queryState.settings = NONE_SCOPE_SETTINGS;
    queryState.isPending = false;
    queryState.isError = false;
    const view = renderPage("org-1");
    expect(screen.getByTestId("account-redesign-notifications-off")).toBeInTheDocument();

    queryState.settings = ALL_SCOPE_SETTINGS;
    view.rerender(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter initialEntries={["/settings/account/notifications"]}>
          <TooltipProvider>
            <FactorySettingsLayoutContext.Provider
              value={{ organizationId: "org-2", factoryId: REFUND_FACTORY.id ?? "", factory: REFUND_FACTORY }}
            >
              <FactorySettingsAccountNotificationsPage />
            </FactorySettingsLayoutContext.Provider>
          </TooltipProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.queryByTestId("account-redesign-notifications-off")).not.toBeInTheDocument();
    expect(screen.getByText("All workspaces")).toBeInTheDocument();
  });
});
