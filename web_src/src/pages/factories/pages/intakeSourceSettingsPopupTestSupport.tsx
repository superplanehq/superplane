import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { IntakeSourceSettingsPopup, type IntakeSettingsConnection } from "./IntakeSourceSettingsPopup";
import {
  DEFAULT_GITHUB_INTAKE_SETTINGS,
  DEFAULT_SENTRY_INTAKE_SETTINGS,
  type IntakeSettingsTab,
} from "./intakeSourceSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import type { LineIntakeSourceId } from "./lineIntakeModel";

/** Same pipeline the canvas editor uses, so the popup renders editor nodes. */
function githubIntakeGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData({
    workflow: {
      metadata: { id: "app-github-issues-intake", name: "GitHub issues", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" },
          {
            id: "filter",
            name: "Matches filters?",
            type: "TYPE_ACTION",
            component: "if",
            configuration: { expression: "true" },
          },
          { id: "create", name: "Create Task", type: "TYPE_ACTION", component: "createWorkOrder" },
        ],
        edges: [
          { channel: "default", sourceId: "trigger", targetId: "filter" },
          { channel: "true", sourceId: "filter", targetId: "create" },
        ],
      },
    },
    triggers: [{ name: "github.onIssue", label: "On Issue" }],
    components: [
      { name: "if", label: "If" },
      { name: "createWorkOrder", label: "Create Task" },
    ],
    nodeEventsMap: {},
    nodeExecutionsMap: {},
    nodeQueueItemsMap: {},
    workflowId: "app-github-issues-intake",
    queryClient: new QueryClient(),
    user: null,
    canvasMode: "live",
  });

  return {
    nodes,
    edges,
    factoryId: "factory-1",
    specNodes: [{ id: "trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" }],
  };
}

const githubAutomationGraph = githubIntakeGraph();

export function renderPopup(
  props: {
    onSave?: (next: typeof DEFAULT_GITHUB_INTAKE_SETTINGS) => void;
    onClose?: () => void;
    editAutomationHref?: string;
    canvasId?: string;
    runHrefFor?: (runId: string) => string;
    agent?: PlanningReviewAgentSlot;
    initialTab?: IntakeSettingsTab;
    labelOptions?: string[];
    labelOptionsLoading?: boolean;
    sourceId?: LineIntakeSourceId;
    connection?: IntakeSettingsConnection;
    settings?: typeof DEFAULT_GITHUB_INTAKE_SETTINGS;
    organizationId?: string;
    integrationId?: string;
    resourceId?: string;
    paused?: boolean;
    pauseError?: string;
    deleteError?: string;
    onPause?: () => void | Promise<void>;
    onResume?: () => void | Promise<void>;
    onDelete?: () => void | Promise<void>;
    saveError?: string;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <IntakeSourceSettingsPopup
              settings={
                props.settings ??
                (props.sourceId === "sentry-exceptions"
                  ? DEFAULT_SENTRY_INTAKE_SETTINGS
                  : props.sourceId === "jira-issues"
                    ? { ...DEFAULT_GITHUB_INTAKE_SETTINGS, name: "Jira issues" }
                    : DEFAULT_GITHUB_INTAKE_SETTINGS)
              }
              sourceId={props.sourceId}
              connection={props.connection}
              organizationId={props.organizationId}
              integrationId={props.integrationId}
              resourceId={props.resourceId}
              labelOptions={props.labelOptions ?? ["bug", "enhancement"]}
              labelOptionsLoading={props.labelOptionsLoading}
              automationGraph={githubAutomationGraph}
              onSave={props.onSave ?? vi.fn()}
              saveError={props.saveError}
              paused={props.paused}
              pauseError={props.pauseError}
              deleteError={props.deleteError}
              onPause={props.onPause}
              onResume={props.onResume}
              onDelete={props.onDelete}
              editAutomationHref={props.editAutomationHref}
              canvasId={props.canvasId}
              runHrefFor={props.runHrefFor}
              agent={props.agent}
              onClose={props.onClose ?? vi.fn()}
              initialTab={props.initialTab}
              fixed={false}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

export function intakeConnection(overrides: Partial<IntakeSettingsConnection> = {}): IntakeSettingsConnection {
  return {
    binding: { integrationId: "", resourceId: "" },
    integrationsBasePath: "/org-1/workspaces/rf/settings/organization/integrations",
    integrations: [
      {
        metadata: { id: "jira-1", name: "Atlassian", integrationName: "jira" },
        status: { state: "ready" },
      },
    ],
    projects: [],
    onBindingChange: vi.fn(),
    onConnect: vi.fn(),
    ...overrides,
  };
}
