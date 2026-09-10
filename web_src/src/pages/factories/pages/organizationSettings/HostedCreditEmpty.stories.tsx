import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import {
  EXPIRED_WELCOME_USAGE_REPORT,
  LOW_TRIAL_USAGE_REPORT,
  SPENT_CREDIT_USAGE_REPORT,
  STORYBOOK_HOSTED_CREDIT_PRODUCTS,
} from "../../__fixtures__/usageReportFixtures";
import { FactorySettingsLayout } from "../settings/FactorySettingsLayout";

/**
 * Empty hosted credit on Tasks. The banner action opens Organization Billing.
 */
const meta = {
  title: "Factories/Pages/Hosted Credit Empty",
  component: FactorySettingsLayout,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof FactorySettingsLayout>;

export default meta;

type Story = StoryObj<typeof meta>;

const spendingPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/spending`;

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

/** Tasks list on the welcome-credit trial. The banner sits above the board. */
export const Trial: Story = {
  name: "Trial",
  render: () => {
    window.localStorage.setItem("sp:work-orders:layout", "board");
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={defaultFactoriesFixture}
      />
    );
  },
};

/** Tasks list when remaining trial credit is low. The banner names the stop outcome. */
export const TrialLowCredit: Story = {
  name: "Trial low credit",
  render: () => {
    window.localStorage.setItem("sp:work-orders:layout", "board");
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: LOW_TRIAL_USAGE_REPORT,
        }}
      />
    );
  },
};

/** Tasks list after welcome credit expires. The banner sits above the board. */
export const TrialEnded: Story = {
  name: "Trial ended",
  render: () => {
    window.localStorage.setItem("sp:work-orders:layout", "board");
    return (
      <FactoriesHarness
        pathSuffix={`workspaces/${PRIMARY_FACTORY_KEY}/work-orders`}
        factoriesFixture={{
          ...defaultFactoriesFixture,
          organizationWorkspaceUsage: EXPIRED_WELCOME_USAGE_REPORT,
        }}
      />
    );
  },
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
