import type { ComponentType } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
} from "../../../__fixtures__/factoryPageResponses";
import { FactorySettingsNavProvider } from "../../settings/FactorySettingsNavProvider";
import { ORG_SPENDING_ONLY_NAV_GROUPS } from "../../settings/storybookFactorySettingsNav";
import { SpendingRedesignPage } from "./SpendingRedesignPage";
import {
  SPENDING_CATALOGS,
  SPENDING_CREDIT,
  SPENDING_CREDIT_WARNING,
  SPENDING_LEDGER,
  SPENDING_REDESIGN_NOW,
} from "./spendingRedesignMocks";
import { EMPTY_SPENDING_FILTERS, rangeForPreset } from "./spendingRedesignLib";

/**
 * Organization Spending explorer. Storybook-only: one Spending tab under
 * Organization, with a time range and two explorers. Mock data follows
 * `workspace_usage_events` (model tokens + runner VM time).
 */
const meta = {
  title: "Factories/Pages/Settings/Spending Redesign",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj;

const spendingPath = `workspaces/${PRIMARY_FACTORY_KEY}/settings/organization/spending`;

function LastThirtyDaysPage() {
  return (
    <SpendingRedesignPage
      catalogs={SPENDING_CATALOGS}
      credit={SPENDING_CREDIT}
      events={SPENDING_LEDGER}
      now={SPENDING_REDESIGN_NOW}
    />
  );
}

function LastDayPage() {
  return (
    <SpendingRedesignPage
      catalogs={SPENDING_CATALOGS}
      credit={SPENDING_CREDIT}
      events={SPENDING_LEDGER}
      initialPeriod="day"
      now={SPENDING_REDESIGN_NOW}
    />
  );
}

function ThisWeekPage() {
  return (
    <SpendingRedesignPage
      catalogs={SPENDING_CATALOGS}
      credit={SPENDING_CREDIT}
      events={SPENDING_LEDGER}
      initialPeriod="week"
      now={SPENDING_REDESIGN_NOW}
    />
  );
}

function LastYearPage() {
  return (
    <SpendingRedesignPage
      catalogs={SPENDING_CATALOGS}
      credit={SPENDING_CREDIT}
      events={SPENDING_LEDGER}
      initialModelBreakdown="user"
      initialPeriod="year"
      now={SPENDING_REDESIGN_NOW}
    />
  );
}

function CustomRangePage() {
  return (
    <SpendingRedesignPage
      catalogs={SPENDING_CATALOGS}
      credit={SPENDING_CREDIT}
      events={SPENDING_LEDGER}
      initialCustomRange={rangeForPreset("week", SPENDING_REDESIGN_NOW)}
      initialPeriod="custom"
      now={SPENDING_REDESIGN_NOW}
    />
  );
}

function FilteredWorkspacePage() {
  return (
    <SpendingRedesignPage
      catalogs={SPENDING_CATALOGS}
      credit={SPENDING_CREDIT}
      events={SPENDING_LEDGER}
      initialModelBreakdown="model"
      initialModelFilters={{
        ...EMPTY_SPENDING_FILTERS,
        workspaceId: PRIMARY_FACTORY_ID,
        model: "anthropic/claude-sonnet-4-6",
      }}
      now={SPENDING_REDESIGN_NOW}
    />
  );
}

function EmptyPage() {
  return (
    <SpendingRedesignPage
      catalogs={SPENDING_CATALOGS}
      credit={SPENDING_CREDIT_WARNING}
      events={[]}
      now={SPENDING_REDESIGN_NOW}
    />
  );
}

function renderSpending(Page: ComponentType) {
  return (
    <FactorySettingsNavProvider groups={ORG_SPENDING_ONLY_NAV_GROUPS}>
      <FactoriesHarness
        enableOnboarding={false}
        factoriesFixture={defaultFactoriesFixture}
        pageOverrides={{ organizationSpending: Page }}
        pathSuffix={spendingPath}
      />
    </FactorySettingsNavProvider>
  );
}

/** Default explorer: last 30 days, all workspaces, stacked by workspace. */
export const LastThirtyDays: Story = {
  name: "Last 30 days",
  render: () => renderSpending(LastThirtyDaysPage),
};

/** Last 24 hours with hourly bars. */
export const LastDay: Story = {
  name: "Last day",
  render: () => renderSpending(LastDayPage),
};

/** Rolling 7-day window with daily bars. */
export const ThisWeek: Story = {
  name: "This week",
  render: () => renderSpending(ThisWeekPage),
};

/** Last 12 months with monthly bars, grouped by task owner on model usage. */
export const LastYear: Story = {
  name: "Last 12 months",
  render: () => renderSpending(LastYearPage),
};

/** Custom range selected. Use the calendar to change the start and end days. */
export const CustomRange: Story = {
  name: "Custom range",
  render: () => renderSpending(CustomRangePage),
};

/** Semaphore workspace and Claude Sonnet only, broken down by model. */
export const FilteredWorkspaceAndModel: Story = {
  name: "Filtered workspace and model",
  render: () => renderSpending(FilteredWorkspacePage),
};

/** No ledger rows. Hosted credit is empty so the wallet warning stays visible. */
export const Empty: Story = {
  render: () => renderSpending(EmptyPage),
};
