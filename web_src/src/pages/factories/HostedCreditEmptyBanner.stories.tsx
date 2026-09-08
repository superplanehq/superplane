import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import { HostedCreditEmptyBanner } from "./HostedCreditEmptyBanner";

/**
 * Compact banner warning about hosted credit. Tasks and Missions show this
 * above the board with a link to Organization Spending; the workspace board
 * shows it with a (not yet wired) go-to-billing button. Red styling ("empty")
 * is severe — hosted runs cannot start. Amber styling ("low") warns before
 * that happens.
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

/** Polar billing is on. The action links to spending. */
export const BillingOn: Story = {
  name: "Billing on",
  args: { level: "empty", billingEnabled: true },
};

/** Polar billing is off. The action opens spending for an installation admin. */
export const BillingOff: Story = {
  name: "Billing off",
  args: { level: "empty", billingEnabled: false },
};

/** Remaining hosted credit is at or below $20. Amber, less severe than empty. */
export const LowCredit: Story = {
  name: "Low credit",
  args: { level: "low", billingEnabled: true },
};

/** The workspace board has no billing page yet, so the action is a no-op button. */
export const BoardGoToBillingButton: Story = {
  name: "Board (go to billing button)",
  args: {
    level: "empty",
    billingEnabled: true,
    spendingHref: undefined,
    onGoToBilling: () => console.log("Go to billing"),
  },
};
