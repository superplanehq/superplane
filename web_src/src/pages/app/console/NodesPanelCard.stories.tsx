import type { Meta, StoryObj } from "@storybook/react-vite";

import type { ConsolePanel } from "@/hooks/useCanvasData";

import { NodesPanelCard } from "./NodesPanelCard";
import { MockConsoleProvider, PanelFrame } from "./__stories__/storyDecorators";

/**
 * Multi-node ("Key Nodes") panel. Renders a compact list of canvas nodes with
 * an optional description line and per-row Run button. Uses the real
 * `NodesPanelCard` with the mock `ConsoleContext` for node resolution.
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
      <MockConsoleProvider>
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
