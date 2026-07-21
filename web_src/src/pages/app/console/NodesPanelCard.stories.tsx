import type { Meta, StoryObj } from "@storybook/react-vite";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { NodesPanelCard } from "./NodesPanelCard";
import { MockConsoleProvider, PanelFrame } from "./__stories__/storyDecorators";

/**
 * Adaptive node panel. Renders as a compact centered card when configured
 * with exactly one entry (matching the pre-merge single-node card) and as a
 * row list otherwise. Uses the real `NodesPanelCard` with the mock
 * `ConsoleContext` for node resolution.
 */
const meta = {
  title: "Console/Nodes",
  component: NodesPanelCard,
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
      <MockConsoleProvider value={{ canvasId: "" }}>
        <PanelFrame height={260}>
          <Story />
        </PanelFrame>
      </MockConsoleProvider>
    ),
  ],
} satisfies Meta<typeof NodesPanelCard>;

export default meta;
type Story = StoryObj<typeof meta>;

function panel(content: Record<string, unknown>): ConsolePanel {
  return { id: "panel-nodes", type: "nodes", content };
}

/** Legacy single-node panel (`type: "node"`) — kept for import compatibility. */
function legacyNodePanel(content: Record<string, unknown>): ConsolePanel {
  return { id: "panel-node-legacy", type: "node", content };
}

/**
 * Compact single-entry rendering (equivalent to the pre-merge single-node
 * card). The panel is a modern `nodes` shape with a single entry.
 */
export const SingleNode: Story = {
  args: {
    panel: panel({ title: "Deploy to prod", nodes: [{ node: "deploy-prod", showRun: true }] }),
  },
};

/**
 * Legacy `type: "node"` panels still render — the merged card folds them
 * into a one-entry list and uses the compact layout automatically.
 */
export const LegacyNodePanel: Story = {
  args: {
    panel: legacyNodePanel({ title: "Deploy to prod", node: "deploy-prod", showRun: true }),
  },
};

export const MultipleNodes: Story = {
  args: {
    panel: panel({
      title: "Key nodes",
      nodes: [
        { node: "deploy-prod", showRun: true },
        { node: "run-tests", showRun: true },
        { node: "build-image" },
        { node: "notify-slack" },
      ],
    }),
  },
};

export const WithDescriptions: Story = {
  args: {
    panel: panel({
      title: "Pipeline stages",
      nodes: [
        { node: "build-image", description: "Builds and pushes the container image" },
        { node: "run-tests", description: "Runs the integration test suite", showRun: true },
        { node: "deploy-prod", description: "Promotes the build to production", showRun: true },
      ],
    }),
  },
};

export const WithLabelOverrides: Story = {
  args: {
    panel: panel({
      title: "Renamed nodes",
      nodes: [
        { node: "deploy-prod", label: "Production deploy", showRun: true },
        { node: "run-tests", label: "QA gate", description: "Must pass before deploy" },
      ],
    }),
  },
};

/**
 * Prompt-submission layout with the redundant in-body node heading and
 * visible field labels removed. The trigger placeholder still explains the
 * textarea, while its associated label remains available to screen readers.
 */
export const PersonalizedInlineForm: Story = {
  args: {
    panel: panel({
      title: "Create work item",
      nodes: [
        {
          node: "deploy-prod",
          description: "Turn a short prompt into a queued delivery task.",
          showRun: true,
          triggerName: "manual",
          formMode: "inline",
          showNodeLabel: false,
          showFieldLabels: false,
          submitLabel: "Create task",
        },
      ],
    }),
  },
};

export const NodeNotFound: Story = {
  args: {
    panel: panel({
      title: "Mixed references",
      nodes: [{ node: "deploy-prod" }, { node: "ghost-node" }],
    }),
  },
};

export const Empty: Story = {
  args: {
    panel: panel({ title: "Key nodes", nodes: [] }),
  },
};
