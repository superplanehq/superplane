import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { IntakeWebhookFailureNotice } from "./IntakeWebhookFailureNotice";

/**
 * Amber notice for a GitHub issues intake whose webhook SuperPlane could not
 * create. Shown above the board so the user knows why new issues do not
 * arrive, and how to repair the intake.
 */
const meta = {
  title: "Factories/Components/IntakeWebhookFailureNotice",
  component: IntakeWebhookFailureNotice,
  parameters: { layout: "padded" },
  args: {
    message:
      "GitHub did not create the webhook. This repository already has 20 webhooks for issues. Remove unused webhooks in the repository settings, then try again.",
    repository: "superplanehq/superplane",
    retrying: false,
    onRetry: () => console.log("Try again"),
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="max-w-4xl bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof IntakeWebhookFailureNotice>;

export default meta;

type Story = StoryObj<typeof meta>;

/** The repository is at the GitHub webhook limit, with a link to the hooks the user must remove. */
export const WebhookLimitReached: Story = {
  name: "Webhook limit reached",
};

/** While the retry runs, the action shows progress and cannot start again. */
export const RetryInProgress: Story = {
  name: "Retry in progress",
  args: { retrying: true },
};

/** Without an owner/name repository, the notice keeps only the retry action. */
export const MissingRepository: Story = {
  name: "Missing repository",
  args: { repository: undefined },
};
