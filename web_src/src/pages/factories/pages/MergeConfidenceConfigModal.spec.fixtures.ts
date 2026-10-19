import { QueryClient } from "@tanstack/react-query";
import { prepareData } from "@/pages/app/workflowPageHelpers";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

export const mergeConfidenceAvailableIntegrations = [
  {
    name: "github",
    label: "GitHub",
    capabilities: [
      {
        type: "TYPE_ACTION",
        name: "github.createPullRequest",
        label: "Create Pull Request",
        configuration: [{ name: "draft", label: "Draft", type: "boolean" }],
      },
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
      {
        type: "TYPE_TRIGGER",
        name: "github.onIssue",
        label: "On Issue",
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
                  { label: "Opened", value: "opened" },
                  { label: "Edited", value: "edited" },
                  { label: "Reopened", value: "reopened" },
                  { label: "Labeled", value: "labeled" },
                  { label: "Closed", value: "closed" },
                ],
              },
            },
          },
          {
            name: "labels",
            label: "Issue has one of these labels",
            type: "list",
            typeOptions: {
              list: {
                itemLabel: "Label",
                itemDefinition: { type: "string" },
              },
            },
          },
          { name: "labelFilterMode", label: "Label filter", type: "select" },
          { name: "assignment", label: "Assignment", type: "select" },
          { name: "authorsWithAccess", label: "Author is a repository collaborator", type: "boolean" },
          {
            name: "superplaneLabelAdded",
            label: 'The "superplane" label is added to the issue',
            type: "boolean",
          },
        ],
      },
    ],
  },
];

export function implementConfigGraph(
  steps: Array<{ name: string; type: string; prompt: string }>,
): IntakeAutomationGraph {
  return {
    nodes: [],
    edges: [],
    organizationId: "implement-org",
    specNodes: [
      { id: "onrun-implement", name: "Start implementation", type: "TYPE_TRIGGER", component: "onRun" },
      {
        id: "implementation-agent-no-issue",
        name: "Implement From Task Description",
        type: "TYPE_ACTION",
        component: "runnerClaudeCode",
        configuration: { model: "sonnet", executionTimeoutSeconds: 3600, steps },
        concurrency: { max: 5, key: "implementation" },
      },
      {
        id: "create-pr",
        name: "Create Pull Request",
        type: "TYPE_ACTION",
        component: "github.createPullRequest",
        configuration: { draft: true },
      },
    ],
  };
}

export function confidenceGraph(): IntakeAutomationGraph {
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

export function issueGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData({
    workflow: {
      metadata: { id: "github-issues", name: "GitHub issues", factoryId: "factory-1" },
      spec: {
        nodes: [
          {
            id: "on-issue",
            name: "On Issue",
            type: "TYPE_TRIGGER",
            component: "github.onIssue",
            configuration: {
              repository: "{{ install_params.appRepository }}",
              actions: ["opened", "edited"],
            },
          },
        ],
        edges: [],
      },
    },
    triggers: [{ name: "github.onIssue", label: "On Issue" }],
    components: [],
    nodeEventsMap: {},
    nodeExecutionsMap: {},
    nodeQueueItemsMap: {},
    workflowId: "github-issues",
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
        id: "on-issue",
        name: "On Issue",
        type: "TYPE_TRIGGER",
        component: "github.onIssue",
        configuration: {
          repository: "{{ install_params.appRepository }}",
          actions: ["opened", "edited"],
        },
      },
    ],
  };
}
