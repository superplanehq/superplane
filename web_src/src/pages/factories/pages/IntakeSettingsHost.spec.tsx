import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as ComponentDataModule from "@/hooks/useComponentData";
import type * as FactoryIntakeDataModule from "@/hooks/useFactoryIntakeData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSettingsHost } from "./IntakeSettingsHost";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, type IntakeSettingsTab } from "./intakeSourceSettingsModel";
import { lineIntakeSourceById, type ConfiguredLineIntakeSource } from "./lineIntakeModel";

const {
  useCanvas,
  useTriggers,
  useComponents,
  useAvailableIntegrations,
  updateIntake,
  useInfiniteCanvasRuns,
  useDescribeRun,
  useEventExecutions,
} = vi.hoisted(() => ({
  useCanvas: vi.fn(),
  useTriggers: vi.fn(),
  useComponents: vi.fn(),
  useAvailableIntegrations: vi.fn(),
  updateIntake: vi.fn(),
  useInfiniteCanvasRuns: vi.fn(),
  useDescribeRun: vi.fn(),
  useEventExecutions: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", async (importOriginal) => ({
  ...(await importOriginal<typeof CanvasDataModule>()),
  useCanvas,
  useTriggers,
  useInfiniteCanvasRuns,
  useDescribeRun,
  useEventExecutions,
}));

vi.mock("@/hooks/useComponentData", async (importOriginal) => ({
  ...(await importOriginal<typeof ComponentDataModule>()),
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", async (importOriginal) => ({
  ...(await importOriginal<typeof IntegrationsModule>()),
  useAvailableIntegrations,
}));

vi.mock("@/hooks/useFactoryIntakeData", async (importOriginal) => ({
  ...(await importOriginal<typeof FactoryIntakeDataModule>()),
  useUpdateFactoryIntake: () => ({ mutateAsync: updateIntake, isPending: false, error: null }),
}));

const GITHUB_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-github",
  appId: "app-github-issues-intake",
  healthy: true,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS },
  source: lineIntakeSourceById("github-issues")!,
};

const GITHUB_INTAKE_CANVAS = {
  metadata: { id: "app-github-issues-intake", name: "GitHub issues" },
  spec: {
    nodes: [
      { id: "github-issues-trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" },
      {
        id: "github-issues-filter",
        name: "Matches filters?",
        type: "TYPE_ACTION",
        component: "if",
        configuration: { expression: "true" },
      },
      { id: "github-issues-create", name: "Create Task", type: "TYPE_ACTION", component: "createWorkOrder" },
    ],
    edges: [
      { channel: "default", sourceId: "github-issues-trigger", targetId: "github-issues-filter" },
      { channel: "true", sourceId: "github-issues-filter", targetId: "github-issues-create" },
    ],
  },
};

function renderHost(
  props: {
    intake?: ConfiguredLineIntakeSource;
    initialTab?: IntakeSettingsTab;
    onClose?: () => void;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSettingsHost
              organizationId="org-1"
              factoryId="factory-1"
              factoryKey="RF"
              lineId="line-plan"
              intake={props.intake ?? GITHUB_INTAKE}
              initialTab={props.initialTab}
              onClose={props.onClose ?? vi.fn()}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("IntakeSettingsHost", () => {
  beforeEach(() => {
    useCanvas.mockReturnValue({
      data: GITHUB_INTAKE_CANVAS,
      isPending: false,
      isError: false,
      refetch: vi.fn().mockResolvedValue(undefined),
    });
    useTriggers.mockReturnValue({ data: [{ name: "github.onIssue", label: "On Issue" }], isLoading: false });
    useComponents.mockReturnValue({
      data: [
        { name: "if", label: "If" },
        { name: "createWorkOrder", label: "Create Task" },
      ],
      isLoading: false,
    });
    useAvailableIntegrations.mockReturnValue({ data: [], isLoading: false });
    useDescribeRun.mockReturnValue({ data: undefined, isLoading: false, isFetched: true });
    useEventExecutions.mockReturnValue({ data: { executions: [] }, isLoading: false });
    updateIntake.mockResolvedValue({ id: "intake-github" });
    localStorage.clear();
    useInfiniteCanvasRuns.mockReturnValue({
      data: {
        pages: [
          {
            runs: [
              {
                id: "run-intake-1",
                canvasId: "app-github-issues-intake",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                createdAt: "2026-05-01T12:00:00Z",
                rootEvent: {
                  id: "event-intake-1",
                  nodeId: "github-issues-trigger",
                  customName: "On mention on Issue",
                },
                executions: [
                  {
                    id: "exec-filter-1",
                    nodeId: "github-issues-filter",
                    state: "STATE_FINISHED",
                    result: "RESULT_PASSED",
                  },
                ],
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

  it("opens the settings of the intake it is given", () => {
    renderHost();

    const dialog = screen.getByTestId("intake-source-settings");
    expect(within(dialog).getByRole("heading", { name: "Intake GitHub issues" })).toBeInTheDocument();
    expect(within(dialog).queryByLabelText("Name")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("checkbox", { name: "New and re-opened issues" })).toBeChecked();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["General", "Automation"]);
    expect(within(dialog).queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
  });

  it("shows the automation of the intake canvas from the Automation tab", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(useCanvas).toHaveBeenCalledWith("org-1", "app-github-issues-intake", { enabled: true });
    const automation = within(screen.getByTestId("intake-source-settings")).getByTestId("intake-source-automation");
    expect(within(automation).getByTestId("rf__node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByText("Matches filters?")).toBeInTheDocument();
    const edit = within(screen.getByTestId("intake-source-settings")).getByRole("link", { name: "Edit automation" });
    expect(edit).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?configure=1&agent=1&from=lines&lineId=line-plan",
    );
    expect(edit.className).toContain("rounded-md");
    expect(within(automation).queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
    expect(useInfiniteCanvasRuns).toHaveBeenCalledWith("app-github-issues-intake", {}, true);
    const sidebar = within(automation).getByTestId("canvas-runs-sidebar");
    expect(within(sidebar).getByText("On mention on Issue")).toBeInTheDocument();
    expect(within(sidebar).getByRole("link", { name: "On mention on Issue" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/RF/apps/app-github-issues-intake?run=run-intake-1&from=lines&lineId=line-plan",
    );
  });

  it("loads the selected run on the automation canvas", async () => {
    const user = userEvent.setup();
    renderHost({ initialTab: "automation" });

    const automation = within(screen.getByTestId("intake-source-settings")).getByTestId("intake-source-automation");
    expect(automation).not.toHaveAttribute("data-selected-run-id");
    expect(automation.querySelector(".sp-canvas-live")).toBeNull();

    await user.click(within(automation).getByRole("link", { name: "On mention on Issue" }));

    expect(automation).toHaveAttribute("data-selected-run-id", "run-intake-1");
    expect(automation.querySelector(".sp-canvas-live")).not.toBeNull();
    expect(within(automation).getByTestId("rf__node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByTestId("rf__node-github-issues-filter")).toBeInTheDocument();
    expect(within(automation).getByTestId("rf__node-github-issues-create")).toBeInTheDocument();
  });

  it("reports an intake without an automation instead of drawing one", async () => {
    useCanvas.mockReturnValue({ data: undefined, isPending: false });
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    const automation = screen.getByTestId("intake-source-automation");
    expect(automation).toHaveTextContent("This intake has no automation yet.");
    expect(within(automation).queryByTestId("rf__node-github-issues-trigger")).not.toBeInTheDocument();
  });

  it("saves the settings without renaming the intake", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-github",
      settings: {
        confidencePct: 65,
        labels: [],
        labelFilterMode: "LABEL_FILTER_MODE_INCLUDE",
        assignment: "ASSIGNMENT_ANY",
        authorsWithAccess: false,
        newIssues: true,
        assignedToAgent: false,
      },
    });
  });
});
