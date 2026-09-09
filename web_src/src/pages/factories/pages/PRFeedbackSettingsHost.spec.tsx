import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as ComponentDataModule from "@/hooks/useComponentData";
import type * as FactoryPRFeedbackDataModule from "@/hooks/useFactoryPRFeedbackData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { TooltipProvider } from "@/ui/tooltip";

import { PR_DISCUSSION_HANDLER } from "../__fixtures__/columnAutomationsFixture";
import { PRFeedbackSettingsHost } from "./PRFeedbackSettingsHost";

const { useCanvas, useTriggers, useComponents, useInfiniteCanvasRuns } = vi.hoisted(() => ({
  useCanvas: vi.fn(),
  useTriggers: vi.fn(),
  useComponents: vi.fn(),
  useInfiniteCanvasRuns: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@/hooks/useCanvasData", async (importOriginal) => ({
  ...(await importOriginal<typeof CanvasDataModule>()),
  useCanvas,
  useTriggers,
  useInfiniteCanvasRuns,
}));

vi.mock("@/hooks/useComponentData", async (importOriginal) => ({
  ...(await importOriginal<typeof ComponentDataModule>()),
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", async (importOriginal) => ({
  ...(await importOriginal<typeof IntegrationsModule>()),
  useConnectedIntegrations: () => ({ data: [], isLoading: false, error: null }),
  useAvailableIntegrations: () => ({ data: [], isLoading: false }),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", async (importOriginal) => ({
  ...(await importOriginal<typeof FactoryPRFeedbackDataModule>()),
  useFactoryPRFeedbackHandlers: () => ({
    data: [PR_DISCUSSION_HANDLER],
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateFactoryPRFeedbackHandler: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDeleteFactoryPRFeedbackHandler: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

const PR_FEEDBACK_CANVAS = {
  metadata: { id: "app-pr-discussion", name: "Address PR feedback" },
  spec: {
    nodes: [
      { id: "comment", name: "On PR Comment", type: "TYPE_TRIGGER", component: "github.onPRComment" },
      { id: "find", name: "Find Pull Request", type: "TYPE_ACTION", component: "findPullRequest" },
    ],
    edges: [{ channel: "default", sourceId: "comment", targetId: "find" }],
  },
};

function renderHost() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PRFeedbackSettingsHost
              organizationId="org-1"
              factoryId="factory-1"
              factoryKey="RF"
              lineId="line-plan"
              canUpdate
              handlerId={PR_DISCUSSION_HANDLER.id}
              initialTab="automation"
              onClose={vi.fn()}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PRFeedbackSettingsHost", () => {
  beforeEach(() => {
    useCanvas.mockReturnValue({
      data: PR_FEEDBACK_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    useTriggers.mockReturnValue({ data: [{ name: "github.onPRComment", label: "On PR Comment" }], isLoading: false });
    useComponents.mockReturnValue({
      data: [{ name: "findPullRequest", label: "Find Pull Request" }],
      isLoading: false,
    });
    localStorage.clear();
    useInfiniteCanvasRuns.mockReturnValue({
      data: {
        pages: [
          {
            runs: [
              {
                id: "run-pr-1",
                canvasId: "app-pr-discussion",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                createdAt: "2026-05-01T12:00:00Z",
                rootEvent: { id: "event-pr-1", nodeId: "comment", customName: "Please fix the lint error" },
              },
            ],
          },
        ],
      },
      isPending: false,
      isError: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      fetchNextPage: vi.fn(),
      refetch: vi.fn(),
    });
  });

  it("shows the shared automation workspace with factory run links", () => {
    renderHost();

    expect(useCanvas).toHaveBeenCalledWith("org-1", "app-pr-discussion", { enabled: true });
    const automation = within(screen.getByTestId("pr-feedback-settings")).getByTestId("pr-feedback-automation");
    expect(within(automation).getAllByText("Find Pull Request").length).toBeGreaterThan(0);
    const headerRow = screen.getByTestId("settings-automation-header-row");
    expect(within(headerRow).queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Automation menu" })).not.toBeInTheDocument();
    const edit = within(automation).getByRole("link", { name: "Edit automation" });
    expect(edit).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/apps/app-pr-discussion?configure=1&agent=1&from=lines&lineId=line-plan",
    );
    expect(useInfiniteCanvasRuns).toHaveBeenCalledWith("app-pr-discussion", {}, true);
    const sidebar = within(automation).getByTestId("canvas-runs-sidebar");
    expect(within(sidebar).getByRole("link", { name: "Please fix the lint error" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/apps/app-pr-discussion?run=run-pr-1&from=lines&lineId=line-plan",
    );
  });
});
