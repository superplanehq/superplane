import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import { HostedCreditEmptyBanner } from "./HostedCreditEmptyBanner";

/**
 * Compact amber banner for empty hosted credit. Tasks shows this above the
 * board. The action opens Organization Spending.
 */
const meta = {
  title: "Factories/Components/HostedCreditEmptyBanner",
  component: HostedCreditEmptyBanner,
  parameters: { layout: "padded" },
  args: {
    spendingHref: "/org/workspaces/RF/settings/organization/spending",
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="max-w-4xl bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof HostedCreditEmptyBanner>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Polar billing is on. The action adds hosted credit. */
export const BillingOn: Story = {
  name: "Billing on",
  args: { billingEnabled: true },
};

/** Polar billing is off. The action opens spending for an installation admin. */
export const BillingOff: Story = {
  name: "Billing off",
  args: { billingEnabled: false },
};
