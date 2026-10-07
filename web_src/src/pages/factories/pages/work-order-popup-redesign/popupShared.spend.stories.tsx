import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "../../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { OwnerTimeCostRow } from "./popupShared";

const owner = { id: "user-1", name: "Ada Lovelace", initials: "AL" };

const meta = {
  title: "Factories/Task Popup/Spend breakdown",
  parameters: { layout: "padded" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="max-w-xl bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const HeaderSpendHover: Story = {
  name: "Header spend hover",
  render: () => (
    <OwnerTimeCostRow
      fixture={{ owner, costUsd: "$0.73", tokensLabel: "2.7k tokens" }}
      usageByModel={[{ provider: "anthropic", model: "claude-sonnet-4-6", totalTokens: "2700", costCents: "45" }]}
      usageByMachineType={[{ machineType: "e1-large-amd64", durationSeconds: "90", costCents: "28" }]}
    />
  ),
};
