import type { Meta, StoryObj } from "@storybook/react-vite";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { NodePanelCard } from "./NodePanelCard";
import { MockConsoleProvider, PanelFrame } from "./__stories__/storyHelpers";

/**
 * Single-node panel. Renders the real `NodePanelCard`, which only needs a
 * `ConsoleContext` (no data hooks) to resolve the node reference and gate the
 * manual Run button. The mock context supplies sample trigger/action nodes and
 * `canRunNodes: true`.
 */
const meta = {
  title: "Console/Node",
  component: NodePanelCard,
  parameters: { layout: "centered" },
  tags: ["autodocs"],
  argTypes: {
    readOnly: { control: "boolean" },
  },
  args: {
    readOnly: false,
    onDelete: () => console.log("delete"),
    onChange: (content) => console.log("change", content),
    onEditingChange: (editing) => console.log("editing", editing),
  },
  decorators: [
    (Story) => (
      <MockConsoleProvider>
        <PanelFrame height={200}>
          <Story />
        </PanelFrame>
      </MockConsoleProvider>
    ),
  ],
} satisfies Meta<typeof NodePanelCard>;

export default meta;
type Story = StoryObj<typeof meta>;

function panel(content: Record<string, unknown>): ConsolePanel {
  return { id: "panel-node", type: "node", content };
}

export const WithRunButton: Story = {
  args: {
    panel: panel({ title: "Deploy to prod", node: "deploy-prod", showRun: true }),
  },
};

export const NoRunButton: Story = {
  args: {
    panel: panel({ title: "Build image", node: "build-image" }),
  },
};

export const WithLabelOverride: Story = {
  args: {
    panel: panel({ title: "Tests", node: "run-tests", label: "Integration tests", showRun: true }),
  },
};

export const Unconfigured: Story = {
  args: {
    panel: panel({ title: "Pick a node", node: "" }),
  },
};

export const NodeNotFound: Story = {
  args: {
    panel: panel({ title: "Missing node", node: "does-not-exist" }),
  },
};

export const ReadOnly: Story = {
  args: {
    readOnly: true,
    panel: panel({ title: "Deploy to prod", node: "deploy-prod", showRun: true }),
  },
};

/** Org fixture: `pr-risk-review` console → `check-pr` node panel. */
export const PrRiskCheckPullRequest: Story = {
  args: {
    panel: {
      id: "check-pr",
      type: "node",
      content: {
        title: "Check pull request",
        node: "trigger-check-pr",
        showRun: true,
        triggerName: "run",
      },
    },
  },
};
