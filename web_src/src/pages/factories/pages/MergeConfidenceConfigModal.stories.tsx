import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { QueryClient } from "@tanstack/react-query";

import { prepareData } from "@/pages/app/workflowPageHelpers";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { MERGE_CONFIDENCE_CHECKS } from "./mergeConfidenceChecks";

function confidenceGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData({
    workflow: {
      metadata: { id: "merge-confidence", name: "Merge confidence", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "on-pr-risk", name: "On Pull Request", type: "TYPE_TRIGGER", component: "github.onPullRequest" },
          {
            id: "assess-risk",
            name: "Assess Merge Confidence",
            type: "TYPE_ACTION",
            component: "runnerClaudeCode",
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
    organizationId: "org-1",
    specNodes: [
      {
        id: "on-pr-risk",
        name: "On Pull Request",
        type: "TYPE_TRIGGER",
        component: "github.onPullRequest",
        configuration: {
          actions: ["opened", "synchronize", "reopened", "ready_for_review"],
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
        configuration: {
          model: "sonnet",
          executionTimeoutSeconds: 3600,
          steps: MERGE_CONFIDENCE_CHECKS.map((id) => ({
            name: id,
            type: "prompt",
            prompt: `Merge check: ${id}.\nReview this score.`,
          })),
        },
        concurrency: { max: 5, key: "{{ $.data.pull_request.number }}" },
      },
      {
        id: "publish-merge-confidence",
        name: "Publish Merge Confidence",
        component: "github.updatePullRequestBodySection",
        configuration: { sectionKey: "merge-confidence", enabled: true, activationGeneration: 1 },
      },
      {
        id: "comment-merge-confidence",
        name: "Comment Merge Confidence",
        component: "github.publishPullRequestComment",
        configuration: { commentKey: "merge-confidence", enabled: false, activationGeneration: 1 },
      },
      {
        id: "start-assessment-check",
        name: "Start Assessment Check",
        component: "github.setPullRequestCheckRun",
        integration: { id: "github-1" },
        configuration: { enabled: false, activationNodeId: "start-assessment-check", activationGeneration: 1 },
      },
    ],
  };
}

const meta = {
  title: "Factories/Components/MergeConfidenceConfigModal",
  component: MergeConfidenceConfigModal,
  parameters: {
    layout: "fullscreen",
    msw: {
      handlers: [
        http.get("*/api/v1/canvases/merge-confidence/runs", () => HttpResponse.json({ runs: [] })),
        http.get("*/api/v1/organizations/org-1/integrations/github-1/resources", () =>
          HttpResponse.json({ resources: [{ id: "granted", name: "Checks write" }] }),
        ),
      ],
    },
  },
  args: {
    title: "Merge confidence",
    canvasId: "merge-confidence",
    graph: confidenceGraph(),
    onClose: () => undefined,
    onSaveNode: () => undefined,
    onDelete: () => undefined,
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="relative min-h-[720px] bg-gray-50 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof MergeConfidenceConfigModal>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Configuration: Story = {};

export const PRClosureWithoutRuns: Story = {
  args: {
    title: "PR Closure",
    canvasId: "app-pr-closure",
    graph: {
      nodes: [],
      edges: [],
      organizationId: "org-1",
      factoryId: "factory-1",
      specNodes: [{ id: "on-pr-closed", component: "github.onPullRequest", type: "TYPE_TRIGGER" }],
    },
  },
  parameters: {
    msw: { handlers: [http.get("*/api/v1/canvases/app-pr-closure/runs", () => HttpResponse.json({ runs: [] }))] },
  },
};

export const Saving: Story = { args: { onSaveNode: () => new Promise<void>(() => undefined) } };

export const SaveError: Story = {
  args: {
    onSaveNode: () => {
      throw new Error("Save failed");
    },
  },
};

export const GitHubPermissionApprovalRequired: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("*/api/v1/organizations/org-1/integrations/github-1/resources", () =>
          HttpResponse.json({
            resources: [
              {
                id: "missing",
                name: "Accept the new GitHub App permissions in GitHub installation settings, then try again.",
              },
            ],
          }),
        ),
      ],
    },
  },
};
