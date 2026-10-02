import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import type { PlanningReviewDraft } from "./planningReviewMockup";
import { PlanningSettingsHost } from "./PlanningSettingsHost";

const agentState = vi.hoisted(() => ({
  nodeId: "refine-task",
}));

const refineDraft: PlanningReviewDraft = {
  title: "Refine Task",
  components: [
    {
      id: "refine-task",
      title: "Refine Task",
      description: "",
      expanded: true,
      configuration: {
        steps: [{ name: "Refine Task", type: "prompt", prompt: "Custom prompt" }],
      },
      concurrency: { max: "1", key: "" },
    },
  ],
};

vi.mock("@/hooks/useFactoryData", () => ({
  useFactory: () => ({
    isPending: false,
    isError: false,
    data: { lines: [], planning: {} },
    refetch: vi.fn(),
  }),
  useFactoryAutomations: () => ({ data: [{ id: "canvas-1", name: "Backlog" }] }),
  useUpdateFactory: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

vi.mock("./useIntakeAutomationCanvas", () => ({
  useIntakeAutomationCanvas: () => ({
    graph: { nodes: [], edges: [] },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("./useColumnCanvasAgentEditor", () => ({
  useColumnCanvasAgentEditor: () => ({
    agentNode: { id: agentState.nodeId, name: "Agent" },
    draft: refineDraft,
    isLoading: false,
    save: vi.fn(),
    showVisualEvidenceSetting: false,
  }),
}));

vi.mock("../lib/loadDefaultRefinementPrompt", () => ({
  loadDefaultRefinementPrompt: vi.fn(async () => ({
    name: "Refine Task",
    type: "prompt",
    prompt: "Factory default",
  })),
}));

function renderHost() {
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PlanningSettingsHost
              organizationId="org-1"
              factoryId="factory-1"
              factoryKey="refunds"
              initialTab="agent"
              onClose={vi.fn()}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PlanningSettingsHost restore default prompt", () => {
  beforeEach(() => {
    agentState.nodeId = "refine-task";
  });

  it("offers Restore default prompt only for the Refine Task agent", async () => {
    const { unmount } = render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <PlanningSettingsHost
                organizationId="org-1"
                factoryId="factory-1"
                factoryKey="refunds"
                initialTab="agent"
                onClose={vi.fn()}
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(await screen.findByTestId("planning-review-restore-default-prompt")).toBeInTheDocument();
    unmount();

    agentState.nodeId = "analyze";
    renderHost();

    expect(await screen.findByTestId("planning-review-editor")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();
  });
});
