import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";

import { meListModelStatus } from "@/api-client";
import { TooltipProvider } from "@/ui/tooltip";

import { REFUND_FACTORY } from "../../__fixtures__/factoryPageResponses";
import { FactorySettingsLayoutContext } from "./factorySettingsLayoutContext";
import {
  FactorySettingsOrganizationModelStatusPage,
  ModelStatusMatrix,
} from "./FactorySettingsOrganizationModelStatusPage";

vi.mock("@/api-client", () => ({ meListModelStatus: vi.fn() }));

function renderPage(client?: QueryClient) {
  return render(
    <QueryClientProvider client={client ?? new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <TooltipProvider>
          <FactorySettingsLayoutContext.Provider
            value={{ organizationId: "org-1", factoryId: REFUND_FACTORY.id ?? "factory-1", factory: REFUND_FACTORY }}
          >
            <FactorySettingsOrganizationModelStatusPage />
          </FactorySettingsLayoutContext.Provider>
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactorySettingsOrganizationModelStatusPage", () => {
  it("loads the 5-minute window by default", async () => {
    const load = vi.mocked(meListModelStatus);
    load.mockResolvedValue({ data: { window: "5m", cells: [] }, error: undefined });
    const view = renderPage();
    await waitFor(() => expect(load).toHaveBeenCalled());
    expect(load).toHaveBeenCalledWith(
      expect.objectContaining({
        query: { window: "5m" },
        headers: expect.objectContaining({ "x-organization-id": "org-1" }),
      }),
    );
    expect(screen.getByLabelText("Window")).toHaveTextContent("5 minutes");
    expect(
      screen.getByText("No SuperPlane-hosted models and no hosted agent calls in this window."),
    ).toBeInTheDocument();
    view.unmount();
  });

  it("shows the call count on a scored cell and No data when a pair has no calls", () => {
    render(
      <ModelStatusMatrix
        cells={[
          { provider: "anthropic", model: "claude-sonnet-4-6", score: 100, callCount: 3 },
          { provider: "anthropic", model: "claude-opus-4-6" },
        ]}
      />,
    );
    expect(screen.getByTestId("model-status-cell-anthropic-claude-sonnet-4-6")).toHaveTextContent("100");
    expect(screen.getByTestId("model-status-cell-anthropic-claude-sonnet-4-6")).toHaveTextContent("3 calls");
    expect(screen.getByTestId("model-status-cell-anthropic-claude-opus-4-6")).toHaveTextContent("No data");
    expect(screen.getByTestId("model-status-cell-openai-claude-sonnet-4-6")).toHaveTextContent("No data");
  });
});
