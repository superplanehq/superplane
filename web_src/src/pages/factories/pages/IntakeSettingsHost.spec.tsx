import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as CanvasDataModule from "@/hooks/useCanvasData";
import type * as ComponentDataModule from "@/hooks/useComponentData";
import type * as FactoryIntakeDataModule from "@/hooks/useFactoryIntakeData";
import type * as IntegrationsModule from "@/hooks/useIntegrations";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSettingsHost } from "./IntakeSettingsHost";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  INTAKE_SETTINGS_COPY,
  type IntakeSettingsTab,
} from "./intakeSourceSettingsModel";
import { lineIntakeSourceById, type ConfiguredLineIntakeSource } from "./lineIntakeModel";

const {
  useCanvas,
  useTriggers,
  useComponents,
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
  updateIntake,
  deleteIntake,
  useInfiniteCanvasRuns,
  useDescribeRun,
  useEventExecutions,
  startDirectJiraConnect,
} = vi.hoisted(() => ({
  useCanvas: vi.fn(),
  useTriggers: vi.fn(),
  useComponents: vi.fn(),
  useAvailableIntegrations: vi.fn(),
  useConnectedIntegrations: vi.fn(),
  useCreateIntegration: vi.fn(),
  useIntegrationResources: vi.fn(),
  updateIntake: vi.fn(),
  deleteIntake: vi.fn(),
  useInfiniteCanvasRuns: vi.fn(),
  useDescribeRun: vi.fn(),
  useEventExecutions: vi.fn(),
  startDirectJiraConnect: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => {
  function MockMonacoEditor({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) {
    return <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />;
  }
  return { default: MockMonacoEditor, Editor: MockMonacoEditor };
});

vi.mock("@/hooks/useCanvasData", () => ({
  ...unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData"),
  useCanvas,
  useTriggers,
  useInfiniteCanvasRuns,
  useDescribeRun,
  useEventExecutions,
}));

vi.mock("@/hooks/useComponentData", () => ({
  ...unmockedSrc<typeof ComponentDataModule>("hooks/useComponentData"),
  useComponents,
}));

vi.mock("@/hooks/useIntegrations", () => ({
  ...unmockedSrc<typeof IntegrationsModule>("hooks/useIntegrations"),
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
}));

vi.mock("@/lib/startDirectJiraConnect", () => ({
  startDirectJiraConnect,
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({ open, setupReturnTo }: { open: boolean; setupReturnTo?: string }) =>
    open ? <div data-testid="intake-connect-dialog">{setupReturnTo}</div> : null,
}));

vi.mock("@/ui/ConfigureIntegrationDialog", () => ({
  ConfigureIntegrationDialog: ({ integrationId }: { integrationId: string | null }) =>
    integrationId ? <div data-testid="intake-configure-dialog">{integrationId}</div> : null,
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  ...unmockedSrc<typeof FactoryIntakeDataModule>("hooks/useFactoryIntakeData"),
  useUpdateFactoryIntake: () => ({ mutateAsync: updateIntake, isPending: false, error: null }),
  useDeleteFactoryIntake: () => ({ mutateAsync: deleteIntake, isPending: false, error: null }),
}));

const GITHUB_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-github",
  appId: "app-github-issues-intake",
  healthy: true,
  paused: false,
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

const JIRA_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-jira",
  appId: "app-jira-intake",
  healthy: false,
  paused: false,
  health: "HEALTH_MISSING_INTEGRATION",
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Jira issues" },
  source: lineIntakeSourceById("jira-issues")!,
};

function renderHost(
  props: {
    intake?: ConfiguredLineIntakeSource;
    initialTab?: IntakeSettingsTab;
    onClose?: () => void;
    path?: string;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[props.path ?? "/org-1/workspaces/rf/lines/line-plan"]}>
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
    useAvailableIntegrations.mockReturnValue({
      data: [{ name: "jira", label: "Jira", hostedAppInstall: true }],
      isLoading: false,
    });
    useConnectedIntegrations.mockReturnValue({
      data: [
        {
          metadata: { id: "jira-1", name: "Atlassian", integrationName: "jira" },
          status: { state: "ready" },
        },
        {
          metadata: { id: "jira-2", name: "Other Jira", integrationName: "jira" },
          status: { state: "ready" },
        },
        {
          metadata: { id: "jira-broken", name: "Broken Jira", integrationName: "jira" },
          status: { state: "error", stateDescription: "Authorization revoked, please reconnect the account" },
        },
      ],
      isLoading: false,
      refetch: vi.fn(),
    });
    useCreateIntegration.mockReturnValue({ mutateAsync: vi.fn(), reset: vi.fn() });
    useIntegrationResources.mockImplementation(
      (_organizationId: string, _integrationId: string, resourceType: string) => {
        if (resourceType === "project") {
          return {
            data: [
              { id: "ENG", name: "Engineering" },
              { id: "OPS", name: "Operations" },
            ],
            isLoading: false,
            isError: false,
            refetch: vi.fn(),
          };
        }
        return { data: [], isLoading: false, isError: false, refetch: vi.fn() };
      },
    );
    useDescribeRun.mockReturnValue({ data: undefined, isLoading: false, isFetched: true });
    useEventExecutions.mockReturnValue({ data: { executions: [] }, isLoading: false });
    updateIntake.mockResolvedValue({ id: "intake-github" });
    deleteIntake.mockResolvedValue(undefined);
    startDirectJiraConnect.mockReset();
    startDirectJiraConnect.mockResolvedValue(true);
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
    expect(within(dialog).queryByTestId("intake-connection")).not.toBeInTheDocument();
    expect(within(dialog).queryByLabelText("Name")).not.toBeInTheDocument();
    expect(within(dialog).getByRole("checkbox", { name: "A new issue is opened" })).toBeChecked();
    expect(within(dialog).getByRole("checkbox", { name: "A closed issue is re-opened" })).toBeChecked();
    expect(
      within(dialog).getByRole("checkbox", { name: 'The "superplane" label is added to the issue' }),
    ).toBeChecked();
    expect(
      within(dialog)
        .getAllByRole("tab")
        .map((tab) => tab.textContent),
    ).toEqual(["General", "Automation"]);
    expect(within(dialog).queryByRole("tab", { name: "Runs" })).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("intake-source-settings-pause")).not.toBeInTheDocument();
    expect(within(dialog).queryByTestId("intake-source-settings-delete")).not.toBeInTheDocument();
  });

  it("shows the automation of the intake canvas from the Automation tab", async () => {
    const user = userEvent.setup();
    renderHost();

    await user.click(screen.getByRole("tab", { name: "Automation" }));

    expect(useCanvas).toHaveBeenCalledWith("org-1", "app-github-issues-intake", { enabled: true });
    const automation = within(screen.getByTestId("intake-source-settings")).getByTestId("intake-source-automation");
    expect(within(automation).getByTestId("rf__node-github-issues-trigger")).toBeInTheDocument();
    expect(within(automation).getByText("Matches filters?")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Automation menu" })).not.toBeInTheDocument();
    const edit = within(automation).getByRole("link", { name: "Edit automation" });
    expect(edit).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/automations/app-github-issues-intake?configure=1&agent=1&from=lines&lineId=line-plan",
    );
    expect(useInfiniteCanvasRuns).toHaveBeenCalledWith("app-github-issues-intake", {}, true);
    const sidebar = within(automation).getByTestId("factory-automation-runs-sidebar");
    expect(within(sidebar).getByText("On mention on Issue")).toBeInTheDocument();
    expect(within(sidebar).getByRole("link", { name: "On mention on Issue" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/automations/app-github-issues-intake?run=run-intake-1&from=lines&lineId=line-plan",
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
        reopenedIssues: true,
        superplaneLabelAdded: true,
      },
    });
  });

  it("pauses and deletes a Sentry intake from settings", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({
      intake: {
        intakeId: "intake-sentry",
        appId: "app-sentry-intake",
        healthy: true,
        paused: false,
        settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Sentry exceptions" },
        source: lineIntakeSourceById("sentry-exceptions")!,
      },
      onClose,
    });

    await user.click(screen.getByTestId("intake-source-settings-pause"));
    expect(updateIntake).toHaveBeenCalledWith({ intakeId: "intake-sentry", paused: true });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
    await user.click(screen.getByTestId("intake-delete-confirm"));
    expect(deleteIntake).toHaveBeenCalledWith("intake-sentry");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("pauses, resumes, and deletes a Jira intake from settings", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({
      intake: {
        ...JIRA_INTAKE,
        healthy: true,
        health: undefined,
      },
      onClose,
    });

    await user.click(screen.getByTestId("intake-source-settings-pause"));
    expect(updateIntake).toHaveBeenCalledWith({ intakeId: "intake-jira", paused: true });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("intake-delete-dialog")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteDescription);
    await user.click(screen.getByTestId("intake-delete-confirm"));
    expect(deleteIntake).toHaveBeenCalledWith("intake-jira");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("resumes a paused Jira intake from settings", async () => {
    const user = userEvent.setup();
    renderHost({
      intake: {
        ...JIRA_INTAKE,
        healthy: true,
        health: undefined,
        paused: true,
      },
    });

    expect(screen.queryByTestId("intake-source-settings-pause")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("intake-source-settings-resume"));
    expect(updateIntake).toHaveBeenCalledWith({ intakeId: "intake-jira", paused: false });
  });

  it("shows a pause error in settings when pause fails", async () => {
    updateIntake.mockRejectedValue(new Error("pause failed"));
    const user = userEvent.setup();
    renderHost({
      intake: {
        intakeId: "intake-sentry",
        appId: "app-sentry-intake",
        healthy: true,
        paused: false,
        settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Sentry exceptions" },
        source: lineIntakeSourceById("sentry-exceptions")!,
      },
    });

    await user.click(screen.getByTestId("intake-source-settings-pause"));

    expect(updateIntake).toHaveBeenCalledWith({ intakeId: "intake-sentry", paused: true });
    expect(screen.getByRole("alert")).toHaveTextContent("pause failed");
  });

  it("shows a delete error in the confirmation dialog when delete fails", async () => {
    deleteIntake.mockRejectedValue(new Error("delete failed"));
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderHost({
      intake: {
        intakeId: "intake-sentry",
        appId: "app-sentry-intake",
        healthy: true,
        paused: false,
        settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Sentry exceptions" },
        source: lineIntakeSourceById("sentry-exceptions")!,
      },
      onClose,
    });

    await user.click(screen.getByTestId("intake-source-settings-delete"));
    await user.click(screen.getByTestId("intake-delete-confirm"));

    const dialog = screen.getByTestId("intake-delete-dialog");
    expect(within(dialog).getByTestId("intake-delete-error")).toHaveTextContent("delete failed");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("saves a new Jira connection and project", async () => {
    const user = userEvent.setup();
    renderHost({ intake: JIRA_INTAKE });

    expect(screen.getByTestId("intake-connection")).toBeInTheDocument();
    expect(screen.getByTestId("intake-connection-banner")).toHaveTextContent("This intake has no live connection.");
    await user.click(screen.getByTestId("intake-connection-jira-2"));
    await user.click(screen.getByTestId("intake-connection-project-OPS"));
    await user.click(screen.getByTestId("intake-source-settings-save"));

    expect(updateIntake).toHaveBeenCalledWith({
      intakeId: "intake-jira",
      settings: {
        confidencePct: 65,
        labels: [],
        labelFilterMode: "LABEL_FILTER_MODE_INCLUDE",
        assignment: "ASSIGNMENT_ANY",
        authorsWithAccess: false,
        newIssues: true,
        reopenedIssues: true,
        superplaneLabelAdded: true,
      },
      integrationId: "jira-2",
      resourceId: "OPS",
    });
  });

  it("selects the Jira connection returned from OAuth", () => {
    renderHost({
      intake: JIRA_INTAKE,
      path: "/org-1/workspaces/rf/lines/line-plan?intake=1&intakeId=intake-jira&settings=general&jiraIntegrationId=jira-2",
    });

    expect(screen.getByTestId("intake-connection-jira-2")).toBeInTheDocument();
    expect(screen.getByTestId("intake-connection-project-OPS")).toBeInTheDocument();
  });

  it("returns to intake settings after Connect Jira", async () => {
    startDirectJiraConnect.mockResolvedValue(true);
    const user = userEvent.setup();
    renderHost({ intake: JIRA_INTAKE });

    await user.click(screen.getByTestId("intake-connection-connect"));

    expect(startDirectJiraConnect).toHaveBeenCalledWith(
      expect.objectContaining({
        organizationId: "org-1",
        returnTo: "/org-1/workspaces/rf/lines/line-plan?intake=1&intakeId=intake-jira&settings=general",
      }),
    );
  });

  it("opens configure for a Jira connection that is not ready", async () => {
    const user = userEvent.setup();
    renderHost({
      intake: {
        ...JIRA_INTAKE,
        health: "HEALTH_INTEGRATION_NOT_READY",
        integrationId: "jira-broken",
      },
    });

    expect(screen.getByTestId("intake-connection-banner")).toHaveTextContent("This connection cannot receive items.");
    await user.click(screen.getByTestId("intake-connection-reconnect"));
    expect(screen.getByTestId("intake-configure-dialog")).toHaveTextContent("jira-broken");
  });
});
