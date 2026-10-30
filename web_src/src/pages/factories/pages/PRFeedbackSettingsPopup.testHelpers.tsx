import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type { useConnectedIntegrations, useIntegrationResources } from "@/hooks/useIntegrations";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import { TooltipProvider } from "@/ui/tooltip";

import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { PRFeedbackDraftSettings, PRFeedbackSettingsTab } from "./prFeedbackSettingsModel";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

export function mockConnectedIntegrations(data: unknown[] = []) {
  return {
    data,
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useConnectedIntegrations>;
}

export function mockIntegrationResources(data: unknown[] = [], overrides: Record<string, unknown> = {}) {
  return {
    data,
    isLoading: false,
    isPending: false,
    isFetching: false,
    isError: false,
    ...overrides,
  } as unknown as ReturnType<typeof useIntegrationResources>;
}

export function discussionDraft(overrides: Partial<PRFeedbackDraftSettings> = {}): PRFeedbackDraftSettings {
  return {
    source: "discussion",
    name: "Address PR feedback",
    repository: "acme/payments",
    mention: "@superplaneagent",
    ignoreBots: true,
    addressAllBots: false,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    runnerIntegrationIds: [],
    ...overrides,
  };
}

export function checksDraft(overrides: Partial<PRFeedbackDraftSettings> = {}): PRFeedbackDraftSettings {
  return {
    source: "checks",
    name: "Fix pull request checks",
    repository: "acme/app",
    mention: "",
    ignoreBots: false,
    addressAllBots: false,
    allowedBots: [],
    checkNames: [],
    maximumAttempts: 3,
    runnerIntegrationIds: [],
    ...overrides,
  };
}

export function renderChecksPopup(
  onSave = vi.fn(),
  settings: PRFeedbackDraftSettings = checksDraft(),
  organizationId?: string,
  githubIntegrationId?: string,
  vcsProvider?: string,
) {
  render(
    <MemoryRouter>
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <PRFeedbackSettingsPopup
          organizationId={organizationId}
          githubIntegrationId={githubIntegrationId}
          vcsProvider={vcsProvider}
          settings={settings}
          onSave={onSave}
          onDelete={vi.fn()}
          onClose={vi.fn()}
          fixed={false}
        />
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { onSave };
}

const automationGraph = prFeedbackGraph();

function prFeedbackGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData({
    workflow: {
      metadata: { id: "app-pr-feedback", name: "Address PR feedback", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "comment", name: "On PR Comment", type: "TYPE_TRIGGER", component: "github.onPRComment" },
          { id: "find", name: "Find Pull Request", type: "TYPE_ACTION", component: "findPullRequest" },
          { id: "runner", name: "Address PR feedback", type: "TYPE_ACTION", component: "runnerClaude" },
        ],
        edges: [
          { channel: "default", sourceId: "comment", targetId: "find" },
          { channel: "found", sourceId: "find", targetId: "runner" },
        ],
      },
    },
    triggers: [{ name: "github.onPRComment", label: "On PR Comment" }],
    components: [
      {
        name: "findPullRequest",
        label: "Find Pull Request",
        outputChannels: [{ name: "found" }, { name: "notFound" }],
      },
      { name: "runnerClaude", label: "Run Claude Code", outputChannels: [{ name: "passed" }, { name: "failed" }] },
    ],
    nodeEventsMap: {},
    nodeExecutionsMap: {},
    nodeQueueItemsMap: {},
    workflowId: "app-pr-feedback",
    queryClient: new QueryClient(),
    user: null,
    canvasMode: "live",
  });

  return {
    nodes,
    edges,
    factoryId: "factory-1",
    specNodes: [{ id: "comment", name: "On PR Comment", type: "TYPE_TRIGGER", component: "github.onPRComment" }],
  };
}

export function renderAutomationPopup(props: { canvasId?: string } = {}) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PRFeedbackSettingsPopup
              settings={discussionDraft()}
              automationGraph={automationGraph}
              onSave={vi.fn()}
              onClose={vi.fn()}
              canvasId={props.canvasId}
              initialTab="automation"
              fixed={false}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

export function renderAgentSettingsPopup(
  settings: PRFeedbackDraftSettings,
  {
    initialPath = "/",
    initialTab = "general",
    withAgent = true,
    agentLoading = false,
    probe,
  }: {
    initialPath?: string;
    initialTab?: PRFeedbackSettingsTab;
    withAgent?: boolean;
    agentLoading?: boolean;
    probe?: ReactNode;
  } = {},
) {
  const onAgentSave = vi.fn<(draft: typeof PLANNING_REVIEW_DRAFT) => Promise<void>>(async () => {});
  const onSave = vi.fn();
  const onClose = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[initialPath]}>
        <PRFeedbackSettingsPopup
          settings={settings}
          automationGraph={automationGraph}
          agent={
            withAgent
              ? {
                  draft: agentLoading ? undefined : PLANNING_REVIEW_DRAFT,
                  isLoading: agentLoading,
                  organizationId: "org-1",
                  onSave: onAgentSave,
                }
              : undefined
          }
          onSave={onSave}
          onClose={onClose}
          initialTab={initialTab}
          fixed={false}
        />
        {probe}
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onAgentSave, onSave, onClose };
}
