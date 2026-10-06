import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient } from "@tanstack/react-query";

import { prepareData } from "@/pages/app/workflowPageHelpers";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

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
      },
    ],
  };
}

const meta = {
  title: "Factories/Components/MergeConfidenceConfigModal",
  component: MergeConfidenceConfigModal,
  parameters: { layout: "fullscreen" },
  args: {
    title: "Merge confidence",
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

export const Automation: Story = {};
