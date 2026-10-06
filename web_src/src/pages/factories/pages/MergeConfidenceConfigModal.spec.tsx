import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

const { useInfiniteCanvasRuns } = vi.hoisted(() => ({
  useInfiniteCanvasRuns: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@/hooks/useCanvasData", () => {
  const actual = unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData");
  return {
    ...actual,
    useInfiniteCanvasRuns,
    useDescribeRun: () => ({ data: undefined, isLoading: false, isFetching: false, isPending: false, isError: false }),
    useTriggers: () => ({ data: [], isLoading: false }),
    useWidgets: () => ({ data: [], isLoading: false }),
  };
});

vi.mock("@/hooks/useComponentData", () => ({
  useComponents: () => ({
    data: [
      {
        name: "runnerClaudeCode",
        label: "Run Claude Code",
        configuration: [{ name: "model", label: "Model", type: "string" }],
      },
    ],
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useIntegrations", () => {
  const actual = unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations");
  return {
    ...actual,
    useAvailableIntegrations: () => ({
      data: [
        {
          name: "github",
          label: "GitHub",
          capabilities: [
            {
              type: "TYPE_TRIGGER",
              name: "github.onPullRequest",
              label: "On Pull Request",
              configuration: [
                { name: "customName", label: "Run title", type: "string" },
                { name: "repository", label: "Repository", type: "string", required: true },
                {
                  name: "actions",
                  label: "Actions",
                  type: "multi-select",
                  typeOptions: {
                    multiSelect: {
                      options: [
                        { label: "Labeled", value: "labeled" },
                        { label: "Assigned", value: "assigned" },
                      ],
                    },
                  },
                },
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
            },
          ],
        },
      ],
      isLoading: false,
    }),
    useConnectedIntegrations: () => ({ data: [], isLoading: false }),
  };
});

const LATEST_RUN_ID = "11111111-1111-4111-8111-111111111111";
const OLDER_RUN_ID = "22222222-2222-4222-8222-222222222222";

function emptyRunsQuery() {
  return {
    data: { pages: [{ runs: [] }] },
    isPending: false,
    isError: false,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
    refetch: vi.fn(),
  };
}

function confidenceGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData({
    workflow: {
      metadata: { id: "merge-confidence", name: "Merge confidence", factoryId: "factory-1" },
      spec: {
        nodes: [
          {
            id: "on-pr-risk",
            name: "On Pull Request",
            type: "TYPE_TRIGGER",
            component: "github.onPullRequest",
          },
          {
            id: "assess-risk",
            name: "Assess Merge Confidence",
            type: "TYPE_ACTION",
            component: "runnerClaudeCode",
            configuration: { model: "sonnet" },
          },
        ],
        edges: [{ channel: "default", sourceId: "on-pr-risk", targetId: "assess-risk" }],
      },
    },
    triggers: [{ name: "github.onPullRequest", label: "On Pull Request" }],
    components: [{ name: "runnerClaudeCode", label: "Claude Code" }],
    nodeEventsMap: {},
    nodeExecutionsMap: {},
    nodeQueueItemsMap: {},
    workflowId: "merge-confidence",
    queryClient: new QueryClient(),
    user: null,
    canvasMode: "live",
  });
  return {
    nodes,
    edges,
    factoryId: "factory-1",
    specNodes: [
      {
        id: "on-pr-risk",
        name: "On Pull Request",
        type: "TYPE_TRIGGER",
        component: "github.onPullRequest",
        configuration: {
          actions: ["opened", "synchronize", "reopened", "ready_for_review", "labeled"],
          ignoreDrafts: true,
          onlyFactoryPullRequests: true,
          repository: "{{ install_params.appRepository }}",
        },
      },
      {
        id: "assess-risk",
        name: "Assess Merge Confidence",
        type: "TYPE_ACTION",
        component: "runnerClaudeCode",
        configuration: { model: "sonnet" },
      },
    ],
  };
}

function renderModal(onDelete: () => void = vi.fn()) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <MergeConfidenceConfigModal
              title="Merge confidence"
              graph={confidenceGraph()}
              canvasId="merge-confidence"
              onClose={vi.fn()}
              onSaveNode={vi.fn()}
              onDelete={onDelete}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  useInfiniteCanvasRuns.mockReturnValue(emptyRunsQuery());
});

afterEach(() => {
  localStorage.clear();
});

describe("MergeConfidenceConfigModal", () => {
  it("shows the automation and no step form", () => {
    renderModal();

    expect(screen.getByRole("heading", { name: "Merge confidence" })).toBeInTheDocument();
    expect(screen.getByTestId("merge-confidence-config-body")).toHaveAttribute("data-split", "false");
    expect(screen.getByTestId("merge-confidence-config-canvas")).toBeInTheDocument();
    expect(screen.getByTestId("merge-confidence-config-tab-automation")).toHaveAttribute("data-state", "active");
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["Automation", "Runs"]);
    expect(screen.queryByTestId("factory-automation-runs-sidebar")).not.toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-config-form")).not.toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-settings")).not.toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-config-footer")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Search commands and components" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Fit all components in view" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Errors" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Warnings" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "More actions" })).toBeInTheDocument();
  });

  it("opens the component settings beside the automation when a step is selected", async () => {
    const user = userEvent.setup();
    renderModal();

    await user.click(screen.getByText("Assess Merge Confidence"));

    expect(screen.getByTestId("merge-confidence-config-body")).toHaveAttribute("data-split", "true");
    expect(screen.getByTestId("merge-confidence-config-canvas")).toBeInTheDocument();
    expect(screen.queryByTestId("factory-automation-runs-sidebar")).not.toBeInTheDocument();
    const form = screen.getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("heading", { name: "Assess Merge Confidence" })).toBeInTheDocument();
    expect(within(form).getByText("Model")).toBeInTheDocument();
    expect(within(form).getByDisplayValue("sonnet")).toBeInTheDocument();
    expect(within(form).queryByTestId("merge-confidence-settings")).not.toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-config-footer")).not.toBeInTheDocument();
  });

  it("opens the trigger settings from the component fields and can close them", async () => {
    const user = userEvent.setup();
    renderModal();

    await user.click(screen.getByTestId("factory-node-on-pull-request"));

    const form = screen.getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("heading", { name: "On Pull Request" })).toBeInTheDocument();
    expect(within(form).getByText("Repository")).toBeInTheDocument();
    expect(within(form).getByText("Application repository")).toBeInTheDocument();
    expect(within(form).getByText("Start a run when:")).toBeInTheDocument();
    expect(within(form).getByText("Filters")).toBeInTheDocument();
    expect(within(form).queryByText("Actions")).not.toBeInTheDocument();
    expect(within(form).getByRole("checkbox", { name: "A pull request is opened" })).toBeChecked();
    expect(within(form).getByRole("checkbox", { name: "New commits are pushed" })).toBeChecked();
    expect(within(form).getByRole("checkbox", { name: "A pull request is reopened" })).toBeChecked();
    expect(within(form).getByRole("checkbox", { name: "A pull request is ready for review" })).toBeChecked();
    expect(within(form).getByRole("checkbox", { name: "Labeled" })).toBeChecked();
    expect(within(form).queryByRole("checkbox", { name: "Assigned" })).not.toBeInTheDocument();
    expect(within(form).getByRole("checkbox", { name: "Ignore draft pull requests" })).toBeChecked();
    expect(
      within(form).getByRole("checkbox", { name: "Run only when this factory created the pull request" }),
    ).toBeChecked();
    expect(within(form).queryByText("Only pull requests in this factory")).not.toBeInTheDocument();
    expect(within(form).queryByRole("switch")).not.toBeInTheDocument();
    expect(within(form).queryByText("Do not start a run when the pull request is a draft.")).not.toBeInTheDocument();
    expect(
      within(form).queryByText("Start a run only when the pull request belongs to this factory."),
    ).not.toBeInTheDocument();
    expect(within(form).queryByDisplayValue("{{ install_params.appRepository }}")).not.toBeInTheDocument();
    expect(within(form).queryByTestId("node-name-input")).not.toBeInTheDocument();
    expect(within(form).queryByText("Integration")).not.toBeInTheDocument();
    expect(within(form).queryByText("Customize run title")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Collapse step" }));

    expect(screen.queryByTestId("merge-confidence-config-form")).not.toBeInTheDocument();
    expect(screen.getByTestId("merge-confidence-config-canvas")).toBeInTheDocument();
  });

  it("selects the latest run when the Runs tab opens", async () => {
    useInfiniteCanvasRuns.mockReturnValue({
      ...emptyRunsQuery(),
      data: {
        pages: [
          {
            runs: [
              { id: LATEST_RUN_ID, createdAt: "2026-10-06T10:00:00Z" },
              { id: OLDER_RUN_ID, createdAt: "2026-10-05T10:00:00Z" },
            ],
          },
        ],
      },
    });
    const user = userEvent.setup();
    renderModal();

    await user.click(screen.getByTestId("merge-confidence-config-tab-runs"));

    expect(screen.getByTestId("merge-confidence-config-tab-runs")).toHaveAttribute("data-state", "active");
    expect(screen.getByTestId("factory-automation-runs-sidebar")).toBeInTheDocument();
    expect(screen.getByTestId("merge-confidence-config-canvas")).toHaveAttribute("data-selected-run-id", LATEST_RUN_ID);
    expect(screen.getByTestId(`factory-automation-run-${LATEST_RUN_ID}`)).toHaveAttribute("data-selected", "true");
    expect(screen.getByTestId(`factory-automation-run-${OLDER_RUN_ID}`)).not.toHaveAttribute("data-selected");
    expect(screen.queryByTestId("merge-confidence-config-form")).not.toBeInTheDocument();
  });

  it("asks to confirm before it deletes the automation", async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    renderModal(onDelete);

    await user.click(screen.getByRole("button", { name: "More actions" }));
    await user.click(screen.getByRole("menuitem", { name: "Delete automation" }));

    expect(screen.getByRole("alertdialog", { name: "Delete automation" })).toBeInTheDocument();
    expect(screen.getByText("Delete this automation? This cannot be undone.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete automation" }));

    expect(onDelete).toHaveBeenCalledTimes(1);
  });
});
