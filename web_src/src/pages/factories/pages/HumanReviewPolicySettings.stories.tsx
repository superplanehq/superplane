import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { HumanReviewPolicySettings } from "./HumanReviewPolicySettings";

const meta = {
  title: "Factories/Components/HumanReviewPolicySettings",
  component: HumanReviewPolicySettings,
  args: {
    node: { id: "on-human-review", name: "Request review", configuration: { enabled: false, fallbackTeam: "" } },
    reviewers: ["mytmlt", "maintainer"],
    onRetryReviewers: fn(),
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
  args: { node: { id: "on-human-review", configuration: { enabled: true, defaultReviewer: "mytmlt" } } },
};
export const LegacyTeam: Story = {
  args: { node: { id: "on-human-review", configuration: { enabled: true, fallbackTeam: "acme/maintainers" } } },
};
export const LoadingReviewers: Story = {
  args: { reviewers: [], reviewersLoading: true },
};
export const NoReviewers: Story = {
  args: { reviewers: [] },
};
export const ReviewerLoadError: Story = {
  args: {
    reviewers: [],
    reviewersError: "SuperPlane could not load reviewers. Check GitHub App access and try again.",
  },
};
export const SavedReviewerUnavailable: Story = {
  args: {
    node: { id: "on-human-review", configuration: { enabled: true, defaultReviewer: "mytmlt" } },
    reviewers: [],
  },
};
export const SaveError: Story = {
  args: {
    onSave: async () => {
      throw new Error("The default reviewer cannot access this repository. Select another reviewer.");
    },
  },
};
