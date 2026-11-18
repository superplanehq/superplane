import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";

import { meListModelStatus } from "@/api-client";
import { TooltipProvider } from "@/ui/tooltip";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { factorySettingsSectionRoutes } from "./factorySettingsSectionRoutes";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import {
  FactorySettingsOrganizationModelStatusPage,
  ModelStatusMatrix,
} from "./FactorySettingsOrganizationModelStatusPage";

vi.mock("@/api-client", () => ({ meListModelStatus: vi.fn() }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ hasPermission: () => true, isLoading: false }),
}));

function renderPage(client?: QueryClient, throughRoute = false) {
  return render(
    <QueryClientProvider client={client ?? new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/org-1/workspaces/workspace-1/settings/organization/model-status"]}>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{ organizationId: "org-1", factoryId: REFUND_FACTORY.id ?? "factory-1", factory: REFUND_FACTORY }}
          >
            {throughRoute ? (
              <Routes>
                <Route path="/:organizationId" element={<div>Organization home</div>} />
                <Route path="/:organizationId/workspaces/:factoryKey/settings">{factorySettingsSectionRoutes}</Route>
              </Routes>
            ) : (
              <FactorySettingsOrganizationModelStatusPage />
            )}
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsOrganizationModelStatusPage", () => {
  for (const enabled of [false, true]) {
    it(`${enabled ? "opens" : "blocks"} the shipped route when Model status is ${enabled ? "enabled" : "disabled"}`, async () => {
      const load = vi.mocked(meListModelStatus);
      load.mockClear();
      load.mockResolvedValue({ data: { window: "1h", cells: [] }, error: undefined });
      const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
      client.setQueryData(organizationKeys.details("org-1"), {
        spec: { enabledExperimentalFeatures: enabled ? ["model_status"] : [] },
      });
      client.setQueryData(experimentalFeaturesKeys.registry(), { features: [{ id: "model_status", released: false }] });
      const view = renderPage(client, true);
      if (enabled) {
        await waitFor(() => expect(screen.getByLabelText("Window")).toBeInTheDocument());
        expect(load).toHaveBeenCalled();
      } else {
        expect(screen.getByText("Organization home")).toBeInTheDocument();
        expect(load).not.toHaveBeenCalled();
      }
      view.unmount();
      client.clear();
    });
  }

  it("loads the 1-hour window by default", async () => {
    const load = vi.mocked(meListModelStatus);
    load.mockResolvedValue({ data: { window: "1h", cells: [] }, error: undefined });
    const view = renderPage();
    await waitFor(() => expect(load).toHaveBeenCalled());
    expect(load).toHaveBeenCalledWith(
      expect.objectContaining({
        query: { window: "1h" },
        headers: expect.objectContaining({ "x-organization-id": "org-1" }),
      }),
    );
    const windowControl = screen.getByLabelText("Window");
    expect(windowControl).toHaveTextContent("1 hour");
    expect(windowControl).toHaveAttribute("id", "model-status-window");
    expect(
      screen.getByText("No SuperPlane-hosted models and no hosted agent calls in this window."),
    ).toBeInTheDocument();
    view.unmount();
  });

  it("shows the score and No data when a pair has no calls", () => {
    render(
      <ModelStatusMatrix
        cells={[
          { provider: "anthropic", model: "claude-sonnet-4-6", score: 100 },
          { provider: "anthropic", model: "claude-opus-4-6" },
        ]}
      />,
    );
    expect(screen.getByTestId("model-status-cell-anthropic-claude-sonnet-4-6")).toHaveTextContent("100");
    expect(screen.getByTestId("model-status-cell-anthropic-claude-opus-4-6")).toHaveTextContent("No data");
    expect(screen.getByTestId("model-status-cell-openai-claude-sonnet-4-6")).toHaveTextContent("No data");
  });
});
