import type { MeNotificationSettings } from "@/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";
import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import { FactorySettingsNotificationsPage } from "./FactorySettingsNotificationsPage";

const ALL_SCOPE_SETTINGS: MeNotificationSettings = {
  workspaces: { scope: "WORKSPACE_SCOPE_ALL", eventTypes: [], filters: [] },
};

const FILTERED_SCOPE_SETTINGS: MeNotificationSettings = {
  workspaces: {
    scope: "WORKSPACE_SCOPE_FILTERED",
    eventTypes: [],
    filters: [{ workspaceId: REFUND_FACTORY.id, eventTypes: ["TYPE_WORK_ORDER_ASSIGNED"] }],
  },
};

const currentSettings: { value: MeNotificationSettings } = { value: ALL_SCOPE_SETTINGS };

vi.mock("@/hooks/useNotificationSettings", () => ({
  useNotificationSettings: () => ({ data: currentSettings.value, isLoading: false }),
  useUpdateNotificationSettings: () => ({
    mutateAsync: vi.fn(async (settings: MeNotificationSettings) => settings),
    isPending: false,
  }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactories: () => ({ data: [REFUND_FACTORY] }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

function renderPage() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={["/settings/notifications"]}>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{ organizationId: "org-1", factoryId: REFUND_FACTORY.id ?? "", factory: REFUND_FACTORY }}
          >
            <FactorySettingsNotificationsPage />
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsNotificationsPage — workspace scope control", () => {
  it("renders the scope choices as labeled cards, not a filter toggle", () => {
    currentSettings.value = ALL_SCOPE_SETTINGS;
    renderPage();

    expect(screen.getByRole("radiogroup", { name: "Workspace scope" })).toBeInTheDocument();

    expect(screen.getByTestId("notifications-scope-all")).toHaveTextContent("All workspaces");
    expect(screen.getByTestId("notifications-scope-filtered")).toHaveTextContent("Choose workspaces");
    expect(screen.getByTestId("notifications-scope-none")).toHaveTextContent("Off");
    expect(screen.getByTestId("notifications-scope-none")).toHaveTextContent("Do not send any work order emails.");
  });

  it("shows an explicit 'no emails' message instead of an empty area when Off is selected", async () => {
    currentSettings.value = ALL_SCOPE_SETTINGS;
    const user = userEvent.setup();
    renderPage();

    expect(screen.queryByTestId("notifications-scope-off-message")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("notifications-scope-none"));

    expect(screen.getByTestId("notifications-scope-off-message")).toHaveTextContent(
      "You will not receive any work order emails.",
    );
    expect(screen.getByTestId("notifications-scope-none")).toHaveAttribute("aria-checked", "true");
  });

  it("still shows the workspace picker and per-workspace event toggles for the 'Choose workspaces' scope", () => {
    currentSettings.value = FILTERED_SCOPE_SETTINGS;
    renderPage();

    expect(screen.getByTestId("notifications-scope-filtered")).toHaveAttribute("aria-checked", "true");
    expect(screen.getByTestId("notifications-workspace-picker")).toBeInTheDocument();
    expect(screen.getByTestId(`notifications-type-${REFUND_FACTORY.id}-TYPE_WORK_ORDER_ASSIGNED`)).toBeInTheDocument();
  });
});
