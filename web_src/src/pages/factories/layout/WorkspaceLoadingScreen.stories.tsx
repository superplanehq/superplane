import type { Meta, StoryObj } from "@storybook/react-vite";

import { WorkspaceLoadingScreen } from "./WorkspaceLoadingScreen";

const meta = {
  title: "Factories/Workspace loading",
  component: WorkspaceLoadingScreen,
  parameters: { layout: "fullscreen" },
  args: { message: "Loading the board" },
} satisfies Meta<typeof WorkspaceLoadingScreen>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Light: Story = {
  decorators: [
    (Story) => (
      <div className="theme-factories">
        <Story />
      </div>
    ),
  ],
};

export const Dark: Story = {
  decorators: [
    (Story) => (
      <div className="theme-factories dark">
        <Story />
      </div>
    ),
  ],
};
