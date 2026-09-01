import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { SPENT_CREDIT_USAGE_REPORT } from "../../__fixtures__/usageReportFixtures";
import { FactorySettingsLayout } from "../settings/FactorySettingsLayout";

/**
 * Current Organization Spending UI when hosted credit is empty.
 * Hosted runs fail at execute. This page is the recovery screen.
 */
const meta = {
  title: "Factories/Pages/Hosted Credit Empty",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const spendingPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/spending`;

const STORYBOOK_HOSTED_CREDIT_PRODUCTS = [
  { id: "prod-500", name: "Hosted credit 500", amountCents: "50000" },
  { id: "prod-25", name: "Hosted credit 25", amountCents: "2500" },
  { id: "prod-100", name: "Hosted credit 100", amountCents: "10000" },
];

export const OrganizationSpending: Story = {
  name: "Organization Spending",
  render: () => (
    <FactoriesHarness
      pathSuffix={spendingPath}
      factoriesFixture={{
        ...defaultFactoriesFixture,
        organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
        hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
      }}
    />
  ),
};

/** Tasks list with remaining hosted credit empty. The banner sits above the board. */
export const Tasks: Story = {
  render: () => {
    window.localStorage.setItem("sp:work-orders:layout", "board");
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: SPENT_CREDIT_USAGE_REPORT,
          hostedCreditProducts: STORYBOOK_HOSTED_CREDIT_PRODUCTS,
        }}
      />
    );
  },
};
