import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { GitHubWebhookFailureNotice } from "./GitHubWebhookFailureNotice";

const meta = {
  title: "Factories/Components/GitHub webhook failure",
  component: GitHubWebhookFailureNotice,
  parameters: { layout: "padded" },
  args: {
    webhook: {
      id: "failed-hook",
      repository: "superplanehq/superplane",
      eventTypes: ["pull_request", "check_run"],
      affectedAutomations: [
        { id: "review", name: "Review" },
        { id: "verify", name: "Verify" },
      ],
      state: "failed",
      message:
        "GitHub reached the webhook limit for pull requests. Remove unused webhooks in the repository settings, then try again.",
    },
    onRetry: () => console.log("Try again"),
    onRefresh: () => console.log("Refresh status"),
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="max-w-4xl bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof GitHubWebhookFailureNotice>;

export default meta;

type Story = StoryObj<typeof meta>;

export const WebhookLimitReached: Story = { name: "Webhook limit reached" };
export const RetryInProgress: Story = { name: "Retry in progress", args: { retrying: true } };
export const RetryTakingLonger: Story = { name: "Retry taking longer", args: { retryTimedOut: true } };
