import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSettingsHost } from "./IntakeSettingsHost";
import { DEFAULT_GITHUB_INTAKE_SETTINGS, type IntakeSettingsTab } from "./intakeSourceSettingsModel";
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
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Sentry exceptions" },
  source: lineIntakeSourceById("sentry-exceptions")!,
};

export const PRODUCTIVE_INTAKE: ConfiguredLineIntakeSource = {
  intakeId: "intake-productive",
  appId: "app-productive-intake",
  healthy: true,
  paused: false,
  settings: { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Productive tasks" },
  source: lineIntakeSourceById("productive-tasks")!,
};

export function connectedJiraIntake(overrides: Partial<ConfiguredLineIntakeSource> = {}): ConfiguredLineIntakeSource {
  return {
    ...JIRA_INTAKE,
    healthy: true,
    health: undefined,
    ...overrides,
  };
}

export function renderHost(
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
