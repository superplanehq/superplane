import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { experimentalFeaturesKeys } from "@/hooks/useExperimentalFeatures";
import { FEATURE_WORKSPACE_SKILLS } from "@/lib/experimentalFeatures";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import {
  confidenceGraph,
  implementConfigGraph,
  issueGraph,
  mergeConfidenceAvailableIntegrations,
} from "./MergeConfidenceConfigModal.spec.fixtures";
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
  useComponent: () => ({
    data: { configuration: [{ name: "model", label: "Model", type: "string" }] },
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useIntegrations", () => {
  const actual = unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations");
  return {
    ...actual,
    useAvailableIntegrations: () => ({
      data: mergeConfidenceAvailableIntegrations,
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

function renderModal(
  onDelete: () => void = vi.fn(),
  onSaveNode = vi.fn(),
  graph: IntakeAutomationGraph = confidenceGraph(),
  options: { initialNodeId?: string; onClose?: () => void } = {},
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const features = [FEATURE_WORKSPACE_SKILLS];
  client.setQueryData(experimentalFeaturesKeys.registry(), {
    features: features.map((id) => ({ id, released: false })),
  });
  if (graph.organizationId) {
    client.setQueryData(organizationKeys.details(graph.organizationId), {
      spec: { enabledExperimentalFeatures: [] },
    });
  }
  const element = (nextGraph: IntakeAutomationGraph = graph) => (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <MergeConfidenceConfigModal
              title="Merge confidence"
              graph={nextGraph}
              canvasId="merge-confidence"
              onClose={options.onClose ?? vi.fn()}
              onSaveNode={onSaveNode}
              onDelete={onDelete}
              initialNodeId={options.initialNodeId}
              factoryKey="SUPER"
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(element());
  return { ...view, rerenderGraph: (nextGraph: IntakeAutomationGraph) => view.rerender(element(nextGraph)) };
}

beforeEach(() => {
  useInfiniteCanvasRuns.mockReturnValue(emptyRunsQuery());
});

afterEach(() => {
  localStorage.clear();
});

describe("MergeConfidenceConfigModal", () => {
  it("saves Implement configuration without an experimental feature", async () => {
    const user = userEvent.setup();
    let finishSave = () => {};
    const saving = new Promise<void>((resolve) => {
      finishSave = resolve;
    });
    const onClose = vi.fn();
    const onSaveNode = vi.fn(() => saving);
    const steps = [{ name: "Implementation", type: "prompt", prompt: "Built-in implementation instructions" }];
    const graph = implementConfigGraph(steps);
    const view = renderModal(vi.fn(), onSaveNode, graph, {
      initialNodeId: "implementation-agent-no-issue",
      onClose,
    });

    expect(Boolean(screen.queryByTestId("merge-confidence-config-canvas"))).toBe(false);
    expect(Boolean(screen.queryByRole("region", { name: "Steps" }))).toBe(false);
    expect(Boolean(screen.queryByRole("button", { name: "Collapse step" }))).toBe(false);
    expect(screen.getByDisplayValue("sonnet")).toBeInTheDocument();
    const concurrency = screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" });
    expect(concurrency).toHaveValue(5);
    await user.clear(concurrency);
    await user.type(concurrency, "3");
    await user.tab();

    expect(onSaveNode).not.toHaveBeenCalled();
    expect(screen.queryByRole("navigation", { name: "Automation settings" })).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Configuration" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("textbox", { name: /^Name/ })).not.toBeInTheDocument();
    expect(screen.queryByText("Key")).not.toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Default model" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "MCP servers" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Skills" })).not.toBeInTheDocument();
    expect(screen.queryByText("Start implementation")).not.toBeInTheDocument();
    expect(screen.queryByText("Implement From Task Description")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add step" })).not.toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Runs" })).toHaveAttribute("aria-selected", "false");
    const draftPullRequest = screen.getByRole("switch", { name: "Create draft pull requests" });
    expect(draftPullRequest).toBeChecked();
    await user.click(draftPullRequest);
    expect(draftPullRequest).not.toBeChecked();
    expect(onSaveNode).not.toHaveBeenCalled();
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    expect(screen.getByRole("tab", { name: "Runs" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("group", { name: "Default model" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Configuration" }));
    expect(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" })).toHaveValue(3);
    expect(screen.getByRole("switch", { name: "Create draft pull requests" })).not.toBeChecked();
    view.rerenderGraph(implementConfigGraph(steps));
    expect(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" })).toHaveValue(3);
    expect(onSaveNode).not.toHaveBeenCalled();
    const save = screen.getByTestId("implement-settings-save");
    await user.click(save);
    expect(save).toHaveTextContent("Saving");
    expect(save).toBeDisabled();
    expect(screen.getByRole("switch", { name: "Create draft pull requests" })).toBeDisabled();
    expect(screen.getByRole("spinbutton", { name: "Maximum simultaneous runs" })).toBeDisabled();
    await waitFor(() => expect(onSaveNode).toHaveBeenCalledTimes(1));
    finishSave();
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(onSaveNode).toHaveBeenCalledWith([
      expect.objectContaining({
        nodeId: "implementation-agent-no-issue",
        configuration: { model: "sonnet", executionTimeoutSeconds: 3600, steps },
        concurrency: { max: 3, key: "implementation" },
      }),
      expect.objectContaining({
        nodeId: "create-pr",
        name: "Create Pull Request",
        configuration: { draft: false, repository: "{{ install_params.appRepository }}" },
        integration: { id: "github-connection" },
        concurrency: { max: 1, key: "{{ $.data.branch }}" },
      }),
    ]);
  });

  it("opens the first component beside the automation", () => {
    renderModal();

    expect(screen.getByRole("heading", { name: "Merge confidence" })).toBeInTheDocument();
    expect(screen.getByTestId("merge-confidence-config-body")).toHaveAttribute("data-split", "true");
    expect(screen.getByTestId("merge-confidence-config-canvas")).toBeInTheDocument();
    expect(screen.getByTestId("merge-confidence-config-tab-automation")).toHaveAttribute("data-state", "active");
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["Automation", "Runs"]);
    expect(screen.queryByTestId("factory-automation-runs-sidebar")).not.toBeInTheDocument();
    const form = screen.getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("heading", { name: "On Pull Request" })).toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-settings")).not.toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-config-footer")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Search commands and components" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Fit all components in view" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Errors" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Warnings" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "More actions" })).toBeInTheDocument();
  });

  it("selects the requested step when the canvas loads", () => {
    const view = renderModal(
      vi.fn(),
      vi.fn(),
      { nodes: [], edges: [], specNodes: [] },
      { initialNodeId: "assess-risk" },
    );

    expect(screen.queryByTestId("merge-confidence-config-form")).not.toBeInTheDocument();

    view.rerenderGraph(confidenceGraph());

    expect(
      within(screen.getByTestId("merge-confidence-config-form")).getByRole("heading", {
        name: "Assess Merge Confidence",
      }),
    ).toBeInTheDocument();
  });

  it("keeps the first trigger when the requested step is not on the canvas", () => {
    renderModal(vi.fn(), vi.fn(), confidenceGraph(), { initialNodeId: "missing-step" });

    expect(
      within(screen.getByTestId("merge-confidence-config-form")).getByRole("heading", { name: "On Pull Request" }),
    ).toBeInTheDocument();
  });

  it("selects the trigger when the automation loads with the agent listed first", () => {
    const loaded = confidenceGraph();
    const agentFirst: IntakeAutomationGraph = {
      ...loaded,
      specNodes: [...(loaded.specNodes ?? [])].reverse(),
    };
    const view = renderModal(vi.fn(), vi.fn(), { nodes: [], edges: [], specNodes: [] });

    expect(screen.queryByTestId("merge-confidence-config-form")).not.toBeInTheDocument();

    view.rerenderGraph(agentFirst);

    expect(
      within(screen.getByTestId("merge-confidence-config-form")).getByRole("heading", { name: "On Pull Request" }),
    ).toBeInTheDocument();
  });

  it("opens the component settings beside the automation when a step is selected", async () => {
    const user = userEvent.setup();
    renderModal();

    await user.click(screen.getByText("Assess Merge Confidence"));

    expect(screen.getByTestId("merge-confidence-config-body")).toHaveAttribute("data-split", "true");
    expect(screen.getByTestId("merge-confidence-config-body").firstElementChild).toBe(
      screen.getByTestId("merge-confidence-config-form"),
    );
    expect(screen.getByTestId("merge-confidence-config-canvas")).toBeInTheDocument();
    expect(screen.queryByTestId("factory-automation-runs-sidebar")).not.toBeInTheDocument();
    const form = screen.getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("heading", { name: "Assess Merge Confidence" })).toBeInTheDocument();
    expect(within(form).queryByText("Concurrency")).not.toBeInTheDocument();
    expect(within(form).getByText("Model")).toBeInTheDocument();
    expect(within(form).getByDisplayValue("sonnet")).toBeInTheDocument();
    expect(within(form).getByRole("region", { name: "Steps" })).toBeInTheDocument();
    expect(within(form).queryByText("MCP servers")).not.toBeInTheDocument();
    expect(within(form).queryByText("Skills")).not.toBeInTheDocument();
    expect(within(form).queryByRole("button", { name: "Save Agent" })).not.toBeInTheDocument();
    expect(within(form).queryByTestId("node-name-input")).not.toBeInTheDocument();
    expect(within(form).queryByText("Integration")).not.toBeInTheDocument();
    expect(within(form).queryByTestId("merge-confidence-settings")).not.toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-config-footer")).not.toBeInTheDocument();
  });

  it("widens the step settings when you drag the split", async () => {
    const user = userEvent.setup();
    renderModal();
    await user.click(screen.getByText("Assess Merge Confidence"));

    const form = screen.getByTestId("merge-confidence-config-form");
    const handle = screen.getByRole("separator", { name: "Resize step settings" });
    expect(form).toHaveStyle({ width: "50%" });

    const body = screen.getByTestId("merge-confidence-config-body");
    vi.spyOn(body, "getBoundingClientRect").mockReturnValue({
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      bottom: 800,
      right: 1000,
      width: 1000,
      height: 800,
      toJSON: () => ({}),
    });

    fireEvent.pointerDown(handle, { clientX: 500, pointerId: 1 });
    fireEvent.pointerMove(window, { clientX: 640, pointerId: 1 });
    fireEvent.pointerUp(window, { clientX: 640, pointerId: 1 });

    expect(form).toHaveStyle({ width: "64%" });
  });

  it("saves a new step immediately and a step name when you leave the field", async () => {
    const user = userEvent.setup();
    const onSaveNode = vi.fn();
    renderModal(vi.fn(), onSaveNode);

    await user.click(screen.getByText("Assess Merge Confidence"));
    const form = screen.getByTestId("merge-confidence-config-form");
    await user.click(within(form).getByRole("button", { name: "Add step" }));

    expect(onSaveNode).toHaveBeenCalledTimes(1);
    await user.type(within(form).getByTestId("planning-review-step-name-0"), "Review");
    expect(onSaveNode).toHaveBeenCalledTimes(1);

    await user.tab();

    expect(onSaveNode).toHaveBeenCalledTimes(2);
    expect(onSaveNode).toHaveBeenLastCalledWith(
      expect.objectContaining({
        nodeId: "assess-risk",
        configuration: expect.objectContaining({
          steps: [expect.objectContaining({ name: "Review", type: "bash" })],
        }),
      }),
    );
  });

  it("opens On Issue with the intake event checks", async () => {
    const user = userEvent.setup();
    renderModal(vi.fn(), vi.fn(), issueGraph());

    await user.click(screen.getByTestId("factory-node-on-issue"));

    const form = screen.getByTestId("merge-confidence-config-form");
    expect(within(form).getByRole("heading", { name: "On Issue" })).toBeInTheDocument();
    expect(within(form).getByText("Repository")).toBeInTheDocument();
    expect(within(form).getByText("Application repository")).toBeInTheDocument();
    expect(within(form).getByText("Start a run when:")).toBeInTheDocument();
    expect(within(form).getByText("Filters")).toBeInTheDocument();
    expect(within(form).getByRole("checkbox", { name: "Issue has one of these labels" })).toBeDisabled();
    expect(within(form).getByRole("checkbox", { name: "Author is a repository collaborator" })).toBeDisabled();
    expect(within(form).queryByText("Label filter")).not.toBeInTheDocument();
    expect(within(form).queryByText("Assignment")).not.toBeInTheDocument();
    expect(within(form).queryByRole("button", { name: "Add Label" })).not.toBeInTheDocument();
    expect(within(form).getByRole("checkbox", { name: "A new issue is opened" })).toBeChecked();
    expect(within(form).getByRole("checkbox", { name: "A closed issue is re-opened" })).not.toBeChecked();
    expect(
      within(form).getByRole("checkbox", { name: 'The "superplane" label is added to the issue' }),
    ).not.toBeChecked();
    expect(within(form).getByRole("checkbox", { name: "Edited" })).toBeChecked();
    expect(within(form).queryByRole("checkbox", { name: "Closed" })).not.toBeInTheDocument();
    expect(within(form).queryByText("Actions")).not.toBeInTheDocument();
    expect(within(form).queryByTestId("node-name-input")).not.toBeInTheDocument();
    expect(within(form).queryByText("Integration")).not.toBeInTheDocument();
    expect(within(form).queryByText("Customize run title")).not.toBeInTheDocument();
  });

  it("opens the trigger settings from the component fields and can close them", async () => {
    const user = userEvent.setup();
    const view = renderModal();

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

    view.rerenderGraph(confidenceGraph());

    expect(screen.queryByTestId("merge-confidence-config-form")).not.toBeInTheDocument();
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

    await user.click(screen.getByTestId(`factory-automation-run-${OLDER_RUN_ID}`));
    await user.click(screen.getByTestId("merge-confidence-config-tab-automation"));
    await user.click(screen.getByTestId("merge-confidence-config-tab-runs"));

    expect(screen.getByTestId("merge-confidence-config-canvas")).toHaveAttribute("data-selected-run-id", OLDER_RUN_ID);
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

  it("renders a custom body for the selected trigger and keeps other steps", async () => {
    const user = userEvent.setup();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <MergeConfidenceConfigModal
                title="Merge confidence"
                graph={confidenceGraph()}
                canvasId="merge-confidence"
                onClose={vi.fn()}
                onSaveNode={vi.fn()}
                renderNodeBody={(node) =>
                  node.id === "on-pr-risk" ? <p data-testid="custom-trigger-body">Custom trigger settings</p> : null
                }
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const form = screen.getByTestId("merge-confidence-config-form");
    expect(within(form).getByTestId("custom-trigger-body")).toBeInTheDocument();
    expect(within(form).queryByText("Start a run when:")).not.toBeInTheDocument();

    await user.click(screen.getByText("Assess Merge Confidence"));

    expect(screen.queryByTestId("custom-trigger-body")).not.toBeInTheDocument();
    expect(within(screen.getByTestId("merge-confidence-config-form")).getByText("Model")).toBeInTheDocument();
  });

  it("hides the Runs tab when the caller has no runs source", () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <MergeConfidenceConfigModal
                title="Merge confidence"
                graph={confidenceGraph()}
                onClose={vi.fn()}
                onSaveNode={vi.fn()}
                showRunsTab={false}
              />
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["Automation"]);
    expect(screen.queryByTestId("merge-confidence-config-tab-runs")).not.toBeInTheDocument();
  });
});
