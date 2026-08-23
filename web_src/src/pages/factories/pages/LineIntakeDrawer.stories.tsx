import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { LineIntakeDrawer } from "./LineIntakeDrawer";

const meta = {
  title: "Factories/Pages/Intake tree",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const GitHubIssuesExpanded: Story = {
  name: "GitHub issues expanded",
  render: () => (
    <ComponentStoryShell className="flex h-svh bg-slate-300 p-0 dark:bg-slate-900">
      <LineIntakeDrawer initialSourceId="github-issues" onClose={() => undefined} />
    </ComponentStoryShell>
  ),
};
