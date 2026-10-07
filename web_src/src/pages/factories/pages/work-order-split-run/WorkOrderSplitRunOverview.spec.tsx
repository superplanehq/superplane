import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { TooltipProvider } from "@/ui/tooltip";

import { analysisChat, HIGH_CONFIDENCE, INTENT } from "./WorkOrderIntentDocument.testHelpers";
import { WorkOrderSplitRunOverview } from "./WorkOrderSplitRunOverview";

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({ resolveUser: () => null, isLoading: false }),
}));

describe("Task planning defaults", () => {
  it("shows the planning review without experimental features", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } });
    client.setQueryData(experimentalFeaturesKeys.registry(), {
      features: [],
    });
    client.setQueryData(organizationKeys.details("original-org"), { spec: { enabledExperimentalFeatures: [] } });
    const overview = (organizationId: string) => (
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <TooltipProvider>
            <WorkOrderSplitRunOverview
              organizationId={organizationId}
              title="Review a task"
              description="Add a clearer empty state."
              artifacts={[INTENT]}
              checks={[HIGH_CONFIDENCE]}
              resultFooter={<button type="button">Start</button>}
              analysis={analysisChat({ view: { machineStatus: "waiting" } })}
            />
          </TooltipProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );
    render(overview("original-org"));
    expect(screen.getByRole("button", { name: "Open plan" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Suggest changes" })).toBeVisible();
    expect(screen.getByRole("region", { name: "Implementation" })).toBeVisible();
    expect(screen.queryByRole("textbox", { name: "Tell the agent more about this task" })).not.toBeInTheDocument();
  });
});
