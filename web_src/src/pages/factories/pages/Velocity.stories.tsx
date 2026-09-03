import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  type FactoriesFixture,
} from "../__fixtures__/factoryPageResponses";
import {
  DEFAULT_FACTORY_VELOCITY,
  EARLY_USAGE_FACTORY_VELOCITY,
  EMPTY_FACTORY_VELOCITY,
  PEOPLE_SYNC_PENDING_FACTORY_VELOCITY,
} from "../__fixtures__/velocityReportFixtures";
import { VelocityPage } from "./VelocityPage";

const meta = {
  title: "Factories/Pages/Velocity",
  component: VelocityPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof VelocityPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const velocityPath = `workspaces/${PRIMARY_FACTORY_KEY}/velocity`;

/** Serves one velocity report for every period the page can request. */
function fixtureWithVelocity(report: FactoriesFixture["velocityByFactoryId"]): FactoriesFixture {
  return { ...defaultFactoriesFixture, velocityByFactoryId: report };
}

export const Default: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={velocityPath}
      factoriesFixture={fixtureWithVelocity({ [PRIMARY_FACTORY_ID]: DEFAULT_FACTORY_VELOCITY })}
    />
  ),
};

/** New workspace with no merges and no spend — empty state that points at the board. */
export const ZeroState: Story = {
  name: "Zero state",
  render: () => (
    <FactoriesHarness
      pathSuffix={velocityPath}
      factoriesFixture={fixtureWithVelocity({
        [PRIMARY_FACTORY_ID]: { 14: EMPTY_FACTORY_VELOCITY, 30: EMPTY_FACTORY_VELOCITY },
      })}
    />
  ),
};

/**
 * A few hours after setup: people merges fill the period, SuperPlane has one
 * day of output. There is no earlier period, so no metric shows a delta.
 */
export const EarlyUsage: Story = {
  name: "Early usage",
  render: () => (
    <FactoriesHarness
      pathSuffix={velocityPath}
      factoriesFixture={fixtureWithVelocity({
        [PRIMARY_FACTORY_ID]: { 14: EARLY_USAGE_FACTORY_VELOCITY, 30: EARLY_USAGE_FACTORY_VELOCITY },
      })}
    />
  ),
};

/**
 * Right after a repository is connected: the background sync has not stored the
 * repository history yet, so the People series is withheld and the People table
 * explains why the Manual work column is empty.
 */
export const PeopleSyncPending: Story = {
  name: "People sync pending",
  render: () => (
    <FactoriesHarness
      pathSuffix={velocityPath}
      factoriesFixture={fixtureWithVelocity({
        [PRIMARY_FACTORY_ID]: {
          14: PEOPLE_SYNC_PENDING_FACTORY_VELOCITY,
          30: PEOPLE_SYNC_PENDING_FACTORY_VELOCITY,
        },
      })}
    />
  ),
};
