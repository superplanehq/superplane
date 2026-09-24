import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import { HostedCreditHeaderKicker } from "./HostedCreditHeaderKicker";

/**
 * Compact hosted-credit chip next to the page title. A healthy trial uses the
 * violet palette. Trial ended, no plan, low credit, and empty credit use the
 * amber palette. The action opens Organization Billing.
 */
const meta = {
  title: "Factories/Components/HostedCreditHeaderKicker",
  component: HostedCreditHeaderKicker,
  parameters: { layout: "padded" },
  args: {
    spendingHref: "/org/workspaces/RF/settings/organization/billing",
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="max-w-4xl bg-background p-6">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof HostedCreditHeaderKicker>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Welcome credit remains. The violet chip sits next to the page title. */
export const Trial: Story = {
  name: "Trial",
  args: {
    kind: "trial",
    remainingCreditCents: 4124,
    welcomeCreditExpiresAt: new Date(Date.now() + 14 * 24 * 60 * 60 * 1000).toISOString(),
  },
};

/** Welcome credit expired. The amber chip sits next to the page title. */
export const TrialEnded: Story = {
  name: "Trial ended",
  args: { kind: "trial-expired" },
};

/** The organization has no plan. The amber chip sits next to the page title. */
export const NoPlan: Story = {
  name: "No plan",
  args: { kind: "lapsed" },
};

/** Purchased hosted credit remains at or below $20. The amber chip shows the balance. */
export const LowCredit: Story = {
  name: "Low credit",
  args: { kind: "low", remainingCreditCents: 1500 },
};

/** Hosted credit is spent. The amber chip shows the label and the action. */
export const EmptyCredit: Story = {
  name: "Empty credit",
  args: { kind: "empty" },
};

/** Billing is off. Low and empty chips render without the action pill or billing link. */
export const LowCreditNoAction: Story = {
  name: "Low credit, no action",
  args: { kind: "low", remainingCreditCents: 1500, canAddCredit: false },
};
