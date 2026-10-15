import type { Meta, StoryObj } from "@storybook/react-vite";
import { MemoryRouter } from "react-router";

import { AgentRunnerSettings } from "./AgentRunnerSettings";
import { agentNodeWithCommand, LONG_BASH_COMMAND } from "./planningReviewFindBox.storyHelpers";

const meta = {
  title: "Factories/Pages/Agent Runner Settings",
  component: AgentRunnerSettings,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof AgentRunnerSettings>;

export default meta;

type Story = StoryObj<typeof AgentRunnerSettings>;

function panel(command: string) {
  return (
    <MemoryRouter>
      <div className="flex h-svh w-full max-w-3xl flex-col border-r border-border bg-background">
        <AgentRunnerSettings
          node={agentNodeWithCommand(command)}
          onSave={(update) => console.log("save agent runner", update)}
        />
      </div>
    </MemoryRouter>
  );
}

export const OneLine: Story = {
  name: "One line script",
  render: () => panel('git clone --depth 1 "$REPO_URL" repo'),
};

export const LongScript: Story = {
  name: "Long script",
  render: () => panel(LONG_BASH_COMMAND),
};
