import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { WorkOrderPopupTabsCompare } from "./WorkOrderPopupTabsCompare";

const meta = {
  title: "Factories/Pages/Work Order Split Run/Tab layouts",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
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

export const TwoColumn: Story = {
  name: "Work order — two columns",
  render: () => <WorkOrderPopupTabsCompare />,
};
