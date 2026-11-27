import type { Meta, StoryObj } from "@storybook/react-vite";
import type { StorybookOrgIntegration } from "@/pages/home/__fixtures__/handlers";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import { PRIMARY_FACTORY_ROUTE_SEGMENT } from "../__fixtures__/factoryPageIds";

const connections: StorybookOrgIntegration[] = [
  {
    metadata: { id: "github-current", name: "puppies-inc", integrationName: "github" },
    status: {
      state: "ready",
      webhookFailures: [
        {
          id: "current-issues",
          state: "failed",
          repositories: ["puppies-inc/api"],
          events: ["issues"],
          limitEvent: "issues",
        },
      ],
    },
  },
  {
    metadata: { id: "github-old", name: "puppies-inc-old", integrationName: "github" },
    status: {
      state: "ready",
      webhookFailures: [
        {
          id: "old-reviews",
          state: "failed",
          repositories: ["puppies-inc/api"],
          events: ["pull_request_review"],
          message: "GitHub could not create the webhook. Check repository access and try again.",
        },
      ],
    },
  },
  {
    metadata: { id: "jira-engineering", name: "Engineering Jira", integrationName: "jira" },
    status: { state: "error", stateDescription: "Reconnect to Jira to receive new issues." },
  },
];

const meta = {
  title: "Factories/Pages/Webhook warnings",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

/** The production board identifies connections that share a provider and repository. */
export const MultipleConnections: Story = {
  render: () => (
    <FactoriesHarness
      mode="app"
      pathSuffix={`workspaces/${PRIMARY_FACTORY_ROUTE_SEGMENT}`}
      orgIntegrations={structuredClone(connections)}
    />
  ),
};
