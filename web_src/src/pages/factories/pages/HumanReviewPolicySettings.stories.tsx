import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { HumanReviewPolicySettings } from "./HumanReviewPolicySettings";

const meta = {
  title: "Factories/Components/HumanReviewPolicySettings",
  component: HumanReviewPolicySettings,
  args: {
    node: { id: "on-human-review", name: "Human review", configuration: { enabled: false, fallbackTeam: "" } },
    onSave: fn(),
  },
  decorators: [
    (Story) => (
      <div className="mx-auto flex max-w-xl bg-background">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof HumanReviewPolicySettings>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Disabled: Story = {};
export const Enabled: Story = {
  args: { node: { id: "on-human-review", configuration: { enabled: true, fallbackTeam: "acme/maintainers" } } },
};
export const SaveError: Story = {
  args: {
    onSave: async () => {
      throw new Error("The GitHub team could not load. Check the team name and App permissions.");
    },
  },
};
