import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { FEATURE_WORKSPACE_AGENT, FEATURE_WORKSPACE_MCP } from "@/lib/experimentalFeatures";
import { TooltipProvider } from "@/ui/tooltip";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  PRIMARY_FACTORY_ROUTE_SEGMENT,
  REFUND_FACTORY,
} from "../__fixtures__/factoryPageResponses";
import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { factoryAgentPath } from "../lib/factoryPagePaths";
import { MobileMorePage } from "./MobileMorePage";

let permissionsLoading = false;
let canUpdateFactories = true;
let enabledFeatures = new Set<string>();

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Ada" } }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) =>
      resource === "factories" && action === "update" ? canUpdateFactories : true,
    isLoading: permissionsLoading,
  }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: (featureId: string) => enabledFeatures.has(featureId),
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { metadata: { name: "Acme" } } }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryPRFeedbackHandlers: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryAutomations: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn() }),
}));

vi.mock("../lib/useHostedCreditEmptyBanner", () => ({
  useHostedCreditChrome: () => ({}),
}));

function renderMore() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <ThemeProvider>
        <TooltipProvider>
          <MemoryRouter initialEntries={[`/org-1/workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}/more`]}>
            <FactoriesLayoutContext.Provider
              value={{
                organizationId: "org-1",
                factoryId: PRIMARY_FACTORY_ID,
                factoryKey: PRIMARY_FACTORY_KEY,
                routeSegment: PRIMARY_FACTORY_ROUTE_SEGMENT,
                factory: REFUND_FACTORY,
                factories: [REFUND_FACTORY],
                openCreateWorkOrder: vi.fn(),
              }}
            >
              <Routes>
                <Route path="/org-1/workspaces/:factoryKey/more" element={<MobileMorePage />} />
              </Routes>
            </FactoriesLayoutContext.Provider>
          </MemoryRouter>
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>,
  );
}

describe("MobileMorePage", () => {
  beforeEach(() => {
    permissionsLoading = false;
    canUpdateFactories = true;
    enabledFeatures = new Set([FEATURE_WORKSPACE_AGENT]);
  });

  it("opens Agent when the user can update factories and SuperPlane Agent is on", () => {
    renderMore();

    expect(screen.getByRole("link", { name: "Agent" })).toHaveAttribute(
      "href",
      factoryAgentPath("org-1", PRIMARY_FACTORY_ROUTE_SEGMENT),
    );
  });

  it("hides Agent when only MCP is on", () => {
    enabledFeatures = new Set([FEATURE_WORKSPACE_MCP]);
    renderMore();

    expect(screen.queryByRole("link", { name: "Agent" })).not.toBeInTheDocument();
  });

  it("hides Agent while permissions load", () => {
    permissionsLoading = true;
    renderMore();

    expect(screen.queryByRole("link", { name: "Agent" })).not.toBeInTheDocument();
  });

  it("hides Agent without update permission", () => {
    canUpdateFactories = false;
    renderMore();

    expect(screen.queryByRole("link", { name: "Agent" })).not.toBeInTheDocument();
  });

  it("hides Agent when SuperPlane Agent is off", () => {
    enabledFeatures = new Set();
    renderMore();

    expect(screen.queryByRole("link", { name: "Agent" })).not.toBeInTheDocument();
  });
});
