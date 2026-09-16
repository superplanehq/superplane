import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient } from "@tanstack/react-query";

import { prepareData } from "@/pages/app/workflowPageHelpers";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { ColumnAutomationViewPopup } from "./ColumnAutomationViewPopup";
import { PLANNING_REVIEW_DRAFT } from "./planningReviewMockup";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

function implementGraph(): IntakeAutomationGraph {
  const { nodes, edges } = prepareData(
    {
      metadata: { id: "app-refund-implementer", name: "Implement", factoryId: "factory-1" },
      spec: {
        nodes: [
          { id: "on-run", name: "On run", type: "TYPE_TRIGGER", component: "onRun" },
          { id: "agent", name: "Implement From Task Description", type: "TYPE_ACTION", component: "runnerClaudeCode" },
        ],
        edges: [{ channel: "default", sourceId: "on-run", targetId: "agent" }],
      },
    },
    [{ name: "onRun", label: "On run" }],
    [{ name: "runnerClaudeCode", label: "Claude Code" }],
    {},
    {},
    {},
    "app-refund-implementer",
    new QueryClient(),
    null,
    "live",
  );
  return { nodes, edges, factoryId: "factory-1" };
}

const meta = {
  title: "Factories/Components/ColumnAutomationViewPopup",
  component: ColumnAutomationViewPopup,
  parameters: { layout: "fullscreen" },
  args: {
    title: "Implement",
    graph: implementGraph(),
    editHref: "/org-1/workspaces/RF/apps/app-refund-implementer?configure=1&agent=1",
    onClose: () => undefined,
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="relative min-h-[720px] bg-gray-50 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof ColumnAutomationViewPopup>;

export default meta;

type Story = StoryObj<typeof meta>;

export const View: Story = {
  name: "View",
};

export const Loading: Story = {
  name: "Loading",
  args: {
    graph: { nodes: [], edges: [] },
    loading: true,
    editHref: undefined,
  },
};

export const WithAgent: Story = {
  name: "Agent and Automation",
  args: {
    agent: { draft: PLANNING_REVIEW_DRAFT, organizationId: "org-1" },
  },
};

export const WithGeneralAndAgent: Story = {
  name: "General, Agent, and Automation",
  args: {
    general: <p className="px-6 py-6 text-sm text-muted-foreground">Name and source filters.</p>,
    agent: { draft: PLANNING_REVIEW_DRAFT, organizationId: "org-1" },
    initialTab: "general",
  },
};
