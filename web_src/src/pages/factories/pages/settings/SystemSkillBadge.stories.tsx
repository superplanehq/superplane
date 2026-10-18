import type { Meta, StoryObj } from "@storybook/react-vite";
import { SystemSkillBadge } from "./SystemSkillBadge";

const meta = {
  title: "Factory/Agent/System skill badge",
  component: SystemSkillBadge,
  decorators: [
    (Story) => (
      <div className="flex items-center gap-2 p-8">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof SystemSkillBadge>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Latest: Story = { args: { state: "STATE_LATEST" } };
export const Custom: Story = { args: { state: "STATE_CUSTOM" } };
export const UpdateAvailable: Story = { args: { state: "STATE_UPDATE_AVAILABLE" } };
export const Unavailable: Story = { args: { state: "STATE_UNAVAILABLE" } };
