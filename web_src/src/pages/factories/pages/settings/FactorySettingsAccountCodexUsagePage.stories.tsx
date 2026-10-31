import type { Meta, StoryObj } from "@storybook/react-vite";

import { CodexUsageConnections } from "./FactorySettingsAccountCodexUsagePage";
import { FactorySettingsPageFrame } from "./FactorySettingsCard";

const meta = {
  title: "Pages/Factories/Settings/Account/Codex",
  component: CodexUsageConnections,
  decorators: [
    (Story) => (
      <FactorySettingsPageFrame title="Codex">
        <Story />
      </FactorySettingsPageFrame>
    ),
  ],
} satisfies Meta<typeof CodexUsageConnections>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Connected: Story = {
  args: {
    connections: [
      {
        id: "connection-1",
        accountLabel: "ada@example.com",
        status: "ready",
        observedAt: new Date().toISOString(),
        limits: [
          {
            limitId: "codex",
            planType: "plus",
            primary: {
              usedPercent: 25,
              windowDurationMins: "300",
              resetsAt: new Date(Date.now() + 3600000).toISOString(),
            },
            secondary: {
              usedPercent: 60,
              windowDurationMins: "10080",
              resetsAt: new Date(Date.now() + 86400000).toISOString(),
            },
          },
        ],
      },
    ],
  },
};
export const WithAvailableCredits: Story = {
  args: {
    connections: Connected.args.connections.map((connection) => ({
      ...connection,
      limits: connection.limits.map((limit) => ({ ...limit, credits: { hasCredits: true, balance: "4712" } })),
    })),
  },
};
export const NoConnection: Story = { args: { connections: [] } };
export const UsageUnavailable: Story = {
  args: { connections: [{ id: "connection-1", accountLabel: "ada@example.com", status: "ready" }] },
};
export const ResetPassed: Story = {
  args: {
    connections: [
      {
        id: "connection-1",
        accountLabel: "ada@example.com",
        status: "reconnect_required",
        observedAt: new Date(Date.now() - 86400000).toISOString(),
        limits: [
          {
            limitId: "codex",
            primary: {
              usedPercent: 100,
              windowDurationMins: "300",
              resetsAt: new Date(Date.now() - 3600000).toISOString(),
            },
          },
        ],
      },
    ],
  },
};
