import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { BuiltInAgentPlayground } from "./BuiltInAgentPlayground";
import { SAMPLE_PLAN, SEED_TASKS } from "./builtInAgentMocks";

/**
 * Storybook-only planning agent. The panel opens from the left rail and
 * pushes the board. Production navigation is unchanged.
 */
const meta = {
  title: "Factories/Prototypes/Built-in Agent",
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-svh bg-background p-0">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Closed: Story = {
  render: () => <BuiltInAgentPlayground seed={{ panelOpen: false }} />,
};

export const Empty: Story = {
  render: () => <BuiltInAgentPlayground seed={{ tasks: [], transcript: [] }} />,
};

export const WithTasks: Story = {
  name: "With tasks",
  render: () => <BuiltInAgentPlayground />,
};

export const PlanProposed: Story = {
  name: "Plan proposed",
  render: () => <BuiltInAgentPlayground seed={{ pendingPlan: SAMPLE_PLAN }} />,
};

export const DeleteConfirm: Story = {
  name: "Delete confirm",
  render: () => <BuiltInAgentPlayground seed={{ pendingDeleteId: SEED_TASKS[0]?.id ?? null }} />,
};

export const AgentError: Story = {
  name: "Agent error",
  render: () => <BuiltInAgentPlayground seed={{ hasError: true }} />,
};
