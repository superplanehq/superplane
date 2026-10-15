import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as ComponentDataModule from "@/hooks/useComponentData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { ColumnAutomationViewHost } from "./ColumnAutomationViewPopup";

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

vi.mock("@/hooks/useCanvasData", () => ({
  ...unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData"),
  useCanvas,
  useTriggers,
  useWidgets: () => ({ data: [], isLoading: false }),
  useInfiniteCanvasRuns,
  useDescribeRun: () => ({ data: undefined, isLoading: false, isFetching: false, isPending: false, isError: false }),
  useEventExecutions: () => ({ data: { executions: [] }, isLoading: false }),
  useCanvasStaging: () => ({
    data: { hasStaging: false, stale: false },
    isPending: false,
    refetch: vi.fn().mockResolvedValue({ data: { hasStaging: false, stale: false } }),
  }),
}));

vi.mock("@/hooks/useComponentData", () => ({
  ...unmockedSrc<typeof ComponentDataModule>("hooks/useComponentData"),
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  ...unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations"),
  useConnectedIntegrations: () => ({ data: [], isLoading: false, error: null }),
  useAvailableIntegrations: () => ({ data: [], isLoading: false }),
}));

const PULL_REQUEST_ACTIONS = {
  name: "actions",
  label: "Actions",
  type: "multi-select",
  typeOptions: {
    multiSelect: {
      options: [
        { label: "Opened", value: "opened" },
        { label: "Closed", value: "closed" },
        { label: "Merged", value: "merged" },
        { label: "Declined", value: "declined" },
      ],
    },
  },
};

const GITHUB_PULL_REQUEST_TRIGGER = {
  name: "github.onPullRequest",
  label: "On Pull Request",
  configuration: [
    { name: "customName", label: "Run title", type: "string" },
    { name: "repository", label: "Repository", type: "string", required: true },
    PULL_REQUEST_ACTIONS,
    {
      name: "ignoreDrafts",
      label: "Ignore draft pull requests",
      type: "boolean",
      description: "Do not start a run when the pull request is a draft.",
    },
    {
      name: "onlyFactoryPullRequests",
      label: "Only pull requests in this factory",
      type: "boolean",
      description: "Start a run only when the pull request belongs to this factory.",
    },
  ],
};

const BITBUCKET_PULL_REQUEST_TRIGGER = {
  name: "bitbucket.onPullRequest",
  label: "On Pull Request",
  configuration: [
    { name: "repository", label: "Repository", type: "string", required: true },
    PULL_REQUEST_ACTIONS,
    {
      name: "onlyFactoryPullRequests",
      label: "Only pull requests in this factory",
      type: "boolean",
      description: "Start a run only when the pull request belongs to this factory.",
    },
  ],
};

function closureCanvas(component: "github.onPullRequest" | "bitbucket.onPullRequest", actions: string[]) {
  return {
    metadata: { id: "app-pr-closure", name: "PR Closure", factoryId: "factory-1" },
    spec: {
      nodes: [
        {
          id: "on-pr-closed",
          name: "On Pull Request Closed",
          type: "TYPE_TRIGGER",
          component,
          configuration: {
            actions,
            onlyFactoryPullRequests: true,
            repository: "{{ install_params.appRepository }}",
          },
        },
        {
          id: "complete-work-order",
          name: "Complete Task",
          type: "TYPE_ACTION",
          component: "updateWorkOrderStatus",
        },
      ],
      edges: [{ channel: "default", sourceId: "on-pr-closed", targetId: "complete-work-order" }],
    },
  };
}

function renderHost() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/org-1/workspaces/rf/lines/line-plan"]}>
        <ThemeProvider>
          <TooltipProvider>
            <ColumnAutomationViewHost
              organizationId="org-1"
              factoryId="factory-1"
              factoryKey="RF"
              lineId="line-plan"
              canvasId="app-pr-closure"
              title="PR Closure"
              onClose={vi.fn()}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function expectSplitConfig() {
  const dialog = screen.getByTestId("merge-confidence-config");
  expect(screen.queryByTestId("column-automation-view")).not.toBeInTheDocument();
  expect(within(dialog).getByTestId("merge-confidence-config-body")).toHaveAttribute("data-split", "true");
  expect(within(dialog).getByTestId("merge-confidence-config-canvas")).toBeInTheDocument();
  expect(
    within(dialog)
      .getAllByRole("tab")
      .map((tab) => tab.textContent),
  ).toEqual(["Automation", "Runs"]);
  return within(dialog).getByTestId("merge-confidence-config-form");
}

describe("ColumnAutomationViewHost", () => {
  beforeEach(() => {
    useCanvas.mockReturnValue({
      data: closureCanvas("github.onPullRequest", ["closed"]),
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    useTriggers.mockReturnValue({
      data: [GITHUB_PULL_REQUEST_TRIGGER, BITBUCKET_PULL_REQUEST_TRIGGER],
      isLoading: false,
    });
    useComponents.mockReturnValue({
      data: [{ name: "updateWorkOrderStatus", label: "Update Task Status" }],
      isLoading: false,
    });
    useInfiniteCanvasRuns.mockReturnValue({
      data: { pages: [{ runs: [] }] },
      isPending: false,
      isError: false,
      hasNextPage: false,
      isFetchingNextPage: false,
      fetchNextPage: vi.fn(),
      refetch: vi.fn(),
    });
    localStorage.clear();
  });

  it("opens pull-request closure on the split config screen", () => {
    renderHost();

    const form = expectSplitConfig();
    expect(within(form).getByRole("heading", { name: "On Pull Request Closed" })).toBeInTheDocument();
    expect(within(form).getByText("Start a run when:")).toBeInTheDocument();
    expect(within(form).getByRole("checkbox", { name: "A pull request is opened" })).toBeInTheDocument();
    expect(within(form).queryByTestId("node-name-input")).not.toBeInTheDocument();
    expect(within(form).queryByText("Actions")).not.toBeInTheDocument();
  });

  it("keeps the Bitbucket trigger on the generic settings form", () => {
    useCanvas.mockReturnValue({
      data: closureCanvas("bitbucket.onPullRequest", ["merged", "declined"]),
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });

    renderHost();

    const form = expectSplitConfig();
    expect(within(form).getByRole("heading", { name: "On Pull Request Closed" })).toBeInTheDocument();
    expect(within(form).getByTestId("node-name-input")).toBeInTheDocument();
    expect(within(form).getByText("Actions")).toBeInTheDocument();
    expect(within(form).queryByText("Start a run when:")).not.toBeInTheDocument();
    expect(within(form).queryByRole("checkbox", { name: "A pull request is opened" })).not.toBeInTheDocument();
  });
});
