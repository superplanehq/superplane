import type { Meta, StoryObj } from "@storybook/react-vite";
import type { OrganizationsDescribeOrganizationBillingResponse } from "@/api-client";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { ACTIVE_BILLING_PERIOD_ENDS_AT, DEFAULT_FACTORY_USAGE } from "../../__fixtures__/usageReportFixtures";

const billing: OrganizationsDescribeOrganizationBillingResponse = {
  billingEnabled: true,
  plan: "pay_as_you_go",
  catalogKey: "pay_as_you_go",
  planName: "Pay-as-you-go",
  planCategory: "paid",
  planPriceCents: "0",
  planSource: "system",
  hasPaymentMethod: true,
  hasBillingCustomer: true,
  requirePaymentMethod: true,
  creditPurchaseAllowed: true,
  occupiedSeats: 3,
  pendingInvitationSeats: 1,
  remainingCreditCents: "2840",
  welcomeRemainingCents: "340",
  purchasedRemainingCents: "2500",
  availablePlans: [
    {
      key: "pay_as_you_go",
      name: "Pay-as-you-go",
      priceCents: "0",
      requirePaymentMethod: true,
      parallelTaskLimit: 5,
      seatMaximum: 10,
    },
    {
      key: "starter",
      name: "Starter",
      priceCents: "2000",
      requirePaymentMethod: true,
      subscriptionRequired: true,
      seatMaximum: 10,
    },
    {
      key: "team",
      name: "Team",
      priceCents: "5000",
      requirePaymentMethod: true,
      subscriptionRequired: true,
      perSeat: true,
      seatMaximum: 50,
    },
  ],
};

function fixture(overrides: OrganizationsDescribeOrganizationBillingResponse = {}) {
  return {
    ...defaultFactoriesFixture,
    organizationBilling: { ...billing, ...overrides },
    organizationWorkspaceUsage: {
      ...DEFAULT_FACTORY_USAGE,
      billingEnabled: true,
      hasBillingCustomer: true,
      remainingCreditCents: "2840",
      grantTotalCents: "3500",
      purchasedCreditCents: "2500",
      hostedBilledCents: "660",
      invoices: [
        {
          id: "inv-1",
          productName: "Hosted credit",
          amountCents: "2500",
          status: "paid",
          createdAt: new Date().toISOString(),
        },
      ],
    },
    organizationCreditGrants: [
      { id: "credit-1", kind: "topup", amountCents: "2500", createdAt: new Date().toISOString() },
      {
        id: "credit-2",
        kind: "welcome",
        amountCents: "1000",
        createdAt: new Date(Date.now() - 86400000 * 5).toISOString(),
      },
    ],
    hostedCreditProducts: [{ id: "credit-25", name: "$25 hosted credit", amountCents: "2500" }],
  };
}

const meta = {
  title: "Factories/Settings/Billing",
  component: FactoriesHarness,
  parameters: { layout: "fullscreen" },
  args: {
    mode: "app",
    pathSuffix: `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/billing`,
    factoriesFixture: fixture(),
  },
} satisfies Meta<typeof FactoriesHarness>;
export default meta;
type Story = StoryObj<typeof meta>;

export const PayAsYouGo: Story = { globals: { theme: "dark" } };
export const Light: Story = { globals: { theme: "light" } };
export const Starter: Story = {
  globals: { theme: "dark" },
  args: {
    factoriesFixture: fixture({
      plan: "starter",
      catalogKey: "starter",
      planName: "Starter",
      planPriceCents: "2000",
      planSource: "polar",
      polarSubscriptionStatus: "active",
      currentPeriodEnd: ACTIVE_BILLING_PERIOD_ENDS_AT,
    }),
  },
};
export const EndingSubscription: Story = {
  globals: { theme: "dark" },
  args: {
    factoriesFixture: fixture({
      plan: "starter",
      catalogKey: "starter",
      planName: "Starter",
      planPriceCents: "2000",
      planSource: "polar",
      polarSubscriptionStatus: "active",
      cancelAtPeriodEnd: true,
      currentPeriodEnd: ACTIVE_BILLING_PERIOD_ENDS_AT,
    }),
  },
};
export const PaymentFailed: Story = {
  globals: { theme: "dark" },
  args: {
    factoriesFixture: fixture({
      plan: "starter",
      catalogKey: "starter",
      planName: "Starter",
      planPriceCents: "2000",
      planSource: "polar",
      polarSubscriptionStatus: "past_due",
      paymentFailedAt: new Date().toISOString(),
    }),
  },
};
export const NoCard: Story = {
  globals: { theme: "dark" },
  args: { factoriesFixture: fixture({ hasPaymentMethod: false }) },
};
