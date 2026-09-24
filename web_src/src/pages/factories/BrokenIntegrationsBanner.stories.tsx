import type { Meta, StoryObj } from "@storybook/react-vite";

import { BrokenIntegrationsBanner } from "./BrokenIntegrationsBanner";
import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import type { BrokenIntegration } from "./lib/brokenIntegrations";

/**
 * Amber banner for organization integrations that need attention. Tasks
 * shows this above the board so a broken connection is not only visible on
 * the integration settings page.
 */
const meta = {
  title: "Factories/Components/BrokenIntegrationsBanner",
  component: BrokenIntegrationsBanner,
  parameters: { layout: "padded" },
  args: {
    integrationsBasePath: "/org/workspaces/RF/settings/organization/integrations",
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="max-w-4xl bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof BrokenIntegrationsBanner>;

export default meta;

type Story = StoryObj<typeof meta>;

const uninstalledGitHubApp: BrokenIntegration = {
  id: "gh-1",
  name: "github-main",
  integrationName: "github",
  reason: "error",
  description: "App was uninstalled",
  actionLabel: "Reinstall app",
};

const expiredLlmKey: BrokenIntegration = {
  id: "oa-1",
  name: "openai-main",
  integrationName: "openai",
  reason: "error",
  description: "API key expired",
  actionLabel: "Replace key",
};

const incompleteSetup: BrokenIntegration = {
  id: "sl-1",
  name: "slack-main",
  integrationName: "slack",
  reason: "incomplete",
  actionLabel: "Finish setup",
};

/** An uninstalled GitHub App: the next step is to reinstall it. */
export const GitHubAppUninstalled: Story = {
  name: "GitHub App uninstalled",
  args: { integrations: [uninstalledGitHubApp] },
};

/** Several integrations need attention at once, each with its own repair step. */
export const MultipleIntegrations: Story = {
  name: "Multiple integrations",
  args: { integrations: [uninstalledGitHubApp, expiredLlmKey, incompleteSetup] },
};

/** A read-only viewer sees the problem, but not an action to fix it. */
export const ReadOnly: Story = {
  name: "Read-only viewer",
  args: { integrations: [uninstalledGitHubApp], canManageIntegrations: false },
};
