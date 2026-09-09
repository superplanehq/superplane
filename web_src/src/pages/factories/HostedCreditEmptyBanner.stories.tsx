import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";
import { HostedCreditEmptyBanner } from "./HostedCreditEmptyBanner";

/**
 * Status banner for hosted credit. A healthy trial uses quiet chrome.
 * Low, empty, or expired credit uses an amber warning. Tasks and the
 * workspace board show this in the page header. The action opens
 * Organization Billing.
 */
const meta = {
  title: "Factories/Components/HostedCreditEmptyBanner",
  component: HostedCreditEmptyBanner,
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
} satisfies Meta<typeof HostedCreditEmptyBanner>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Polar billing is on. The action opens Billing. */
export const BillingOn: Story = {
  name: "Billing on",
  args: { billingEnabled: true },
};

/** Polar billing is off. The action opens Billing. */
export const BillingOff: Story = {
  name: "Billing off",
  args: { billingEnabled: false },
};

/** Welcome credit remains. The action opens Billing. */
export const Trial: Story = {
  name: "Trial",
  args: {
    billingEnabled: true,
    kind: "trial",
    remainingCreditCents: 4124,
    welcomeCreditExpiresAt: new Date(Date.now() + 14 * 24 * 60 * 60 * 1000).toISOString(),
  },
};

/** Welcome credit expires today. The banner uses warning chrome. */
export const TrialEndsToday: Story = {
  name: "Trial ends today",
  args: {
    billingEnabled: true,
    kind: "trial",
    remainingCreditCents: 4124,
    welcomeCreditExpiresAt: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
  },
};

/** Welcome credit is spent. The action opens Billing. */
export const TrialEmpty: Story = {
  name: "Trial empty",
  args: {
    billingEnabled: true,
    kind: "trial-empty",
  },
};

/** Welcome credit expired. The action opens Billing. */
export const TrialEnded: Story = {
  name: "Trial ended",
  args: {
    billingEnabled: true,
    kind: "trial-expired",
  },
};

/** Purchased hosted credit remains at or below $20. The action opens Billing. */
export const LowCredit: Story = {
  name: "Low credit",
  args: {
    billingEnabled: true,
    kind: "low",
    remainingCreditCents: 1500,
  },
};
