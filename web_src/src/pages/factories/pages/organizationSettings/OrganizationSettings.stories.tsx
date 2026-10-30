import type { Meta, StoryObj } from "@storybook/react-vite";
import { userEvent, within } from "storybook/test";
import { MIXED_CREDIT_GRANTS } from "../../__fixtures__/creditGrantFixtures";
import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../__fixtures__/factoryPageResponses";
import { EMPTY_ORG_SPENDING_REPORT } from "../../__fixtures__/spendingReportFixtures";
import {
  BUSINESS_ORGANIZATION_BILLING,
  TEAM_ORGANIZATION_BILLING,
  DEFAULT_FACTORY_USAGE,
  PURCHASED_CREDIT_USAGE_REPORT,
  SPENT_CREDIT_USAGE_REPORT,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
  STORYBOOK_CUSTOM_CREDIT_PRODUCT,
} from "../../__fixtures__/usageReportFixtures";
import { FactorySettingsLayout } from "../settings/FactorySettingsLayout";

const meta = {
  title: "Factories/Pages/Settings/Organization",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const organizationSettingsPath = (page: string) => `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/${page}`;

export const General: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("general")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Members: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("members")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Integrations: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("integrations")}
      factoriesFixture={defaultFactoriesFixture}
    />
  ),
};

export const ApiKeys: Story = {
  name: "API keys",
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("api-keys")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Secrets: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("secrets")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const Spending: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("spending")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const SpendingEmpty: Story = {
  name: "Spending (empty)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("spending")}
      factoriesFixture={{ ...defaultFactoriesFixture, organizationSpendingReport: EMPTY_ORG_SPENDING_REPORT }}
    />
  ),
};

export const Billing: Story = {
  name: "Billing (Business)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        hostedCreditProducts: [...STORYBOOK_HOSTED_CREDIT_PRODUCTS, STORYBOOK_CUSTOM_CREDIT_PRODUCT],
        organizationBilling: BUSINESS_ORGANIZATION_BILLING,
        organizationCreditGrants: MIXED_CREDIT_GRANTS,
        organizationWorkspaceUsage: {
          ...PURCHASED_CREDIT_USAGE_REPORT,
          billingEnabled: true,
          hasBillingCustomer: true,
          invoices: [
            {
              id: "ord_storybook_business",
              createdAt: "2026-09-10T12:00:00Z",
              amountCents: "19900",
              status: "paid",
              productName: "Business",
            },
            {
              id: "ord_storybook_1",
              createdAt: "2026-09-01T12:00:00Z",
              amountCents: "2500",
              status: "paid",
              productName: "Hosted credit 25",
            },
            {
              id: "ord_storybook_refund",
              createdAt: "2026-08-15T12:00:00Z",
              amountCents: "5000",
              status: "refunded",
              productName: "Hosted credit 50",
            },
          ],
        },
      }}
    />
  ),
};

export const BillingCreditTopUp: Story = {
  ...Billing,
  name: "Billing (credit top-up)",
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body);
    await userEvent.click(await canvas.findByRole("button", { name: "Top up" }));
    await userEvent.click(await canvas.findByRole("button", { name: "$100" }));
    await canvas.findByText("$105.79");
  },
};

export const BillingCustomCreditTopUp: Story = {
  ...Billing,
  name: "Billing (custom credit top-up)",
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body);
    await userEvent.click(await canvas.findByRole("button", { name: "Top up" }));
    await userEvent.click(await canvas.findByRole("button", { name: "Custom" }));
    await userEvent.type(canvas.getByRole("textbox", { name: "Credit amount in USD" }), "100");
    await canvas.findByText("$105.79");
  },
};

export const BillingTeam: Story = {
  name: "Billing (Team seats)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{ ...defaultFactoriesFixture, organizationBilling: { ...TEAM_ORGANIZATION_BILLING } }}
    />
  ),
};

export const BillingBuySeats: Story = {
  ...BillingTeam,
  name: "Billing (buy seats)",
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement.ownerDocument.body);
    await userEvent.click(await canvas.findByRole("button", { name: "Buy seats" }));
    await canvas.findByText("Estimated charge now: $12.34");
  },
};

export const BillingTeamAtMaximum: Story = {
  name: "Billing (Team plan maximum)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationBilling: {
          ...TEAM_ORGANIZATION_BILLING,
          purchasedSeats: 50,
          occupiedSeats: 48,
          availableSeats: 2,
          purchasableSeats: 0,
          currentMonthlyTotalCents: "145000",
        },
      }}
    />
  ),
};

export const BillingSeatPurchasePending: Story = {
  name: "Billing (seat payment pending)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationBilling: {
          ...TEAM_ORGANIZATION_BILLING,
          lastSeatPurchase: {
            id: "storybook-pending-seat-purchase",
            status: "pending",
            fromSeats: 10,
            toSeats: 11,
            createdAt: new Date().toISOString(),
          },
        },
      }}
    />
  ),
};

export const BillingSeatPurchaseFailed: Story = {
  name: "Billing (seat payment failed)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationBilling: {
          ...TEAM_ORGANIZATION_BILLING,
          lastSeatPurchase: {
            id: "storybook-failed-seat-purchase",
            status: "failed",
            fromSeats: 10,
            toSeats: 11,
            message: "The card was declined. Update the payment method and try again.",
            createdAt: new Date().toISOString(),
          },
        },
      }}
    />
  ),
};

export const BillingTeamEnding: Story = {
  name: "Billing (Team subscription ends)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationBilling: { ...TEAM_ORGANIZATION_BILLING, cancelAtPeriodEnd: true },
      }}
    />
  ),
};

export const BillingTeamLegacyCapacity: Story = {
  name: "Billing (Team zero capacity, unlimited maximum)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationBilling: {
          ...TEAM_ORGANIZATION_BILLING,
          purchasedSeats: 0,
          availableSeats: 0,
          currentMonthlyTotalCents: "0",
          planMaximumSeats: undefined,
          purchasableSeats: undefined,
        },
      }}
    />
  ),
};

export const BillingEmptyHistory: Story = {
  name: "Billing (trial welcome)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
        organizationCreditGrants: [],
        organizationWorkspaceUsage: {
          ...defaultFactoriesFixture.organizationWorkspaceUsage!,
          billingEnabled: true,
          hasBillingCustomer: false,
        },
      }}
    />
  ),
};

export const BillingEmptyNoCard: Story = {
  name: "Billing (empty credit, no card)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
        organizationWorkspaceUsage: {
          ...DEFAULT_FACTORY_USAGE,
          remainingCreditCents: "0",
          hostedBilledCents: "5000",
          remainingCreditWarning: true,
          billingEnabled: true,
          hasBillingCustomer: false,
        },
        organizationBilling: {
          ...defaultFactoriesFixture.organizationBilling,
          remainingCreditCents: "0",
          welcomeRemainingCents: "0",
        },
      }}
    />
  ),
};

export const BillingEmptyCredit: Story = {
  name: "Billing (empty credit, card on file)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("billing")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
        organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
        organizationBilling: {
          ...defaultFactoriesFixture.organizationBilling,
          remainingCreditCents: "0",
          welcomeRemainingCents: "0",
        },
      }}
    />
  ),
};

export const Usage: Story = {
  render: () => (
    <FactoriesHarness pathSuffix={organizationSettingsPath("usage")} factoriesFixture={defaultFactoriesFixture} />
  ),
};

export const UsageEmpty: Story = {
  name: "Usage (empty)",
  render: () => (
    <FactoriesHarness
      pathSuffix={organizationSettingsPath("usage")}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        usageHistoryByFactoryId: { [PRIMARY_FACTORY_ID]: [] },
      }}
    />
  ),
};
