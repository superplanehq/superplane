import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { WebhookFailureNotice } from "./IntegrationWebhookFailures";

const meta = {
  title: "Settings/Integrations/Webhook failure",
  component: WebhookFailureNotice,
  parameters: { layout: "padded" },
  args: {
    failure: {
      id: "581dbfe6-1fd6-4f79-aa75-fef4658b8142",
      state: "failed",
      repositories: ["mytmlt/sentry-intake-test-project"],
      events: ["pull_request", "pull_request_review"],
      limitEvent: "pull_request",
      message:
        "GitHub allows 20 webhooks for the pull_request event in each repository. Remove an unused webhook, then try again.",
    },
    canRetry: true,
    onRetry: () => console.log("Retry webhook setup"),
  },
} satisfies Meta<typeof WebhookFailureNotice>;

export default meta;
type Story = StoryObj<typeof meta>;

export const LimitReached: Story = {
  render: function InteractiveNotice(args) {
    const [retrying, setRetrying] = useState(false);
    return (
      <WebhookFailureNotice
        {...args}
        retrying={retrying}
        onRetry={() => {
          console.log("Retry webhook setup");
          setRetrying(true);
        }}
      />
    );
  },
};
export const RetryFailed: Story = { args: { retryError: true } };
export const ReadOnly: Story = { args: { canRetry: false } };
export const Retrying: Story = { args: { retrying: true } };
