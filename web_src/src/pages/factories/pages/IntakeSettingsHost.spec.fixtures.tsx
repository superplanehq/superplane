import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import {
  deleteIntake,
  updateIntake,
  useAvailableIntegrations,
  useCanvas,
  useComponents,
  useDescribeRun,
  useEventExecutions,
  useInfiniteCanvasRuns,
  useIntegrationResources,
  useTriggers,
} from "./intakeSettingsHostSpecMocks";
import { IntakeSettingsHost } from "./IntakeSettingsHost";
import {
  DEFAULT_DATADOG_INTAKE_SETTINGS,
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  DEFAULT_SENTRY_INTAKE_SETTINGS,
  type IntakeSettingsTab,
} from "./intakeSourceSettingsModel";
import { lineIntakeSourceById, type ConfiguredLineIntakeSource } from "./lineIntakeModel";

export const GITHUB_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-github",
  appId: "app-github-issues-intake",
  healthy: true,
  paused: false,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS },
  source: lineIntakeSourceById("github-issues")!,
};

export const GITHUB_INTAKE_CANVAS = {
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

export const JIRA_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-jira",
  appId: "app-jira-intake",
  healthy: false,
  paused: false,
  health: "HEALTH_MISSING_INTEGRATION",
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Jira issues" },
  source: lineIntakeSourceById("jira-issues")!,
};

export const SENTRY_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-sentry",
  appId: "app-sentry-intake",
  healthy: true,
  paused: false,
  settings: { ...DEFAULT_SENTRY_INTAKE_SETTINGS, sentryProjectIds: ["payments"] },
  source: lineIntakeSourceById("sentry-exceptions")!,
  integrationId: "sentry-1",
  resourceId: "payments",
};

export const SENTRY_INTAKE_CANVAS = {
  metadata: { id: "app-sentry-intake", name: "Sentry exceptions", factoryId: "factory-1" },
  spec: {
    nodes: [
      {
        id: "sentry-exceptions-trigger",
        name: "On Issue",
        type: "TYPE_TRIGGER",
        component: "sentry.onIssue",
      },
      {
        id: "sentry-exceptions-create",
        name: "Create Task",
        type: "TYPE_ACTION",
        component: "createWorkOrder",
      },
    ],
    edges: [{ channel: "default", sourceId: "sentry-exceptions-trigger", targetId: "sentry-exceptions-create" }],
  },
};

export const PRODUCTIVE_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-productive",
  appId: "app-productive-intake",
  healthy: true,
  paused: false,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Productive tasks" },
  source: lineIntakeSourceById("productive-tasks")!,
};

export const DEPENDABOT_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-dependabot",
  appId: "app-dependabot-intake",
  healthy: true,
  paused: false,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Dependabot alerts", dependabotSeverities: [] },
  source: lineIntakeSourceById("dependabot-alerts")!,
};

export const DEPENDABOT_INTAKE_CANVAS = {
  metadata: { id: "app-dependabot-intake", name: "Dependabot alerts", factoryId: "factory-1" },
  spec: {
    nodes: [
      {
        id: "dependabot-alerts-trigger",
        name: "On Dependabot Alert",
        type: "TYPE_TRIGGER",
        component: "github.onDependabotAlert",
        configuration: { repository: "superplanehq/superplane", actions: ["created"] },
      },
      {
        id: "dependabot-alerts-create",
        name: "Create Task",
        type: "TYPE_ACTION",
        component: "createWorkOrder",
      },
    ],
    edges: [{ channel: "default", sourceId: "dependabot-alerts-trigger", targetId: "dependabot-alerts-create" }],
  },
};

export const DATADOG_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-datadog",
  appId: "app-datadog-intake",
  healthy: true,
  paused: false,
  settings: { ...DEFAULT_DATADOG_INTAKE_SETTINGS, datadogService: "checkout" },
  source: lineIntakeSourceById("datadog")!,
  integrationId: "datadog-1",
  resourceId: "checkout",
};

export function connectedJiraIntake(overrides: Partial<ConfiguredLineIntakeSource> = {}): ConfiguredLineIntakeSource {
  return {
    ...JIRA_INTAKE,
    healthy: true,
    health: undefined,
    ...overrides,
  };
}

export function resetHostMocks() {
  resetIntakeSettingsHostMocks({
    useCanvas,
    useTriggers,
    useComponents,
    useAvailableIntegrations,
    useIntegrationResources,
    useDescribeRun,
    useEventExecutions,
    useInfiniteCanvasRuns,
    updateIntake,
    deleteIntake,
  });
}

export { deleteIntake, updateIntake, useCanvas, useInfiniteCanvasRuns };

export function resetIntakeSettingsHostMocks(mocks: {
  useCanvas: ReturnType<typeof vi.fn>;
  useTriggers: ReturnType<typeof vi.fn>;
  useComponents: ReturnType<typeof vi.fn>;
  useAvailableIntegrations: ReturnType<typeof vi.fn>;
  useIntegrationResources: ReturnType<typeof vi.fn>;
  useDescribeRun: ReturnType<typeof vi.fn>;
  useEventExecutions: ReturnType<typeof vi.fn>;
  useInfiniteCanvasRuns: ReturnType<typeof vi.fn>;
  updateIntake: ReturnType<typeof vi.fn>;
  deleteIntake: ReturnType<typeof vi.fn>;
}) {
  mocks.useCanvas.mockReturnValue({
    data: GITHUB_INTAKE_CANVAS,
    isPending: false,
    isError: false,
    refetch: vi.fn().mockResolvedValue(undefined),
  });
  mocks.useTriggers.mockReturnValue({
    data: [
      { name: "github.onIssue", label: "On Issue" },
      { name: "github.onDependabotAlert", label: "On Dependabot Alert" },
      { name: "sentry.onIssue", label: "On Issue" },
    ],
    isLoading: false,
  });
  mocks.useComponents.mockReturnValue({
    data: [
      { name: "if", label: "If" },
      { name: "createWorkOrder", label: "Create Task" },
    ],
    isLoading: false,
  });
  mocks.useAvailableIntegrations.mockReturnValue({
    data: [{ name: "jira", label: "Jira", hostedAppInstall: true }],
    isLoading: false,
  });
  mocks.useIntegrationResources.mockImplementation(integrationResourceResult);
  mocks.useDescribeRun.mockReturnValue({ data: undefined, isLoading: false, isFetched: true });
  mocks.useEventExecutions.mockReturnValue({ data: { executions: [] }, isLoading: false });
  mocks.updateIntake.mockResolvedValue({ id: "intake-github" });
  mocks.deleteIntake.mockResolvedValue(undefined);
  localStorage.clear();
  mocks.useInfiniteCanvasRuns.mockReturnValue(intakeRunsResult());
}

function integrationResourceResult(_organizationId: string, integrationId: string, resourceType: string) {
  if (resourceType === "project" && integrationId === "sentry-1") {
    return resourceResult([
      { id: "payments", name: "Payments" },
      { id: "growth", name: "Growth" },
    ]);
  }
  if (resourceType === "issueStatus") {
    return resourceResult([
      { id: "todo", name: "To Do" },
      { id: "qa", name: "QA" },
      { id: "done", name: "Done" },
    ]);
  }
  if (resourceType === "service") {
    return resourceResult([
      { id: "checkout", name: "checkout" },
      { id: "billing", name: "billing" },
    ]);
  }
  if (resourceType === "project") {
    return resourceResult([
      { id: "ENG", name: "Engineering" },
      { id: "OPS", name: "Operations" },
    ]);
  }
  if (resourceType === "label") {
    return resourceResult([
      { id: "bug", name: "bug" },
      { id: "docs", name: "docs" },
    ]);
  }
  return resourceResult([]);
}

function resourceResult(data: Array<{ id: string; name: string }>) {
  return { data, isLoading: false, isError: false, refetch: vi.fn() };
}

function intakeRunsResult() {
  return {
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
  };
}

export function renderHost(
  props: {
    intake?: ConfiguredLineIntakeSource;
    repository?: string;
    vcsIntegrationId?: string;
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
              repository={props.repository}
              vcsIntegrationId={props.vcsIntegrationId}
              initialTab={props.initialTab}
              onClose={props.onClose ?? vi.fn()}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}
