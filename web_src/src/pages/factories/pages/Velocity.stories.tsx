import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  type FactoriesFixture,
} from "../__fixtures__/factoryPageResponses";
import {
  factoriesFixtureWithSetupAnswers,
  GITHUB_CONNECTION_ID,
  GITHUB_SETUP_INTEGRATIONS,
} from "../__fixtures__/setupStoryFixtures";
import {
  DEFAULT_FACTORY_VELOCITY,
  EARLY_USAGE_FACTORY_VELOCITY,
  EMPTY_FACTORY_VELOCITY,
  PEOPLE_LOAD_MORE_FACTORY_VELOCITY,
  PEOPLE_SYNC_PENDING_FACTORY_VELOCITY,
  VELOCITY_REPOSITORY,
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

/**
 * The page reads the GitHub connection and the repository that workspace setup
 * stored, so the fixture workspace must name the repository the reports are
 * built from. Without it the page hides Refresh data and tells the reader to
 * connect GitHub, which a set-up workspace never shows.
 */
const velocityWorkspaceSetup = {
  ...REFUND_FACTORY.onboarding,
  vcsIntegrationId: GITHUB_CONNECTION_ID,
  appRepository: VELOCITY_REPOSITORY,
};

/** Serves one velocity report for every period the page can request. */
function renderVelocity(report: FactoriesFixture["velocityByFactoryId"]) {
  return (
    <FactoriesHarness
      pathSuffix={velocityPath}
      factoriesFixture={{ ...factoriesFixtureWithSetupAnswers(velocityWorkspaceSetup), velocityByFactoryId: report }}
      orgIntegrations={GITHUB_SETUP_INTEGRATIONS}
    />
  );
}

export const Default: Story = {
  render: () => renderVelocity({ [PRIMARY_FACTORY_ID]: DEFAULT_FACTORY_VELOCITY }),
};

/** New workspace with no merges and no spend — empty state that points at the board. */
export const ZeroState: Story = {
  name: "Zero state",
  render: () => renderVelocity({ [PRIMARY_FACTORY_ID]: { 14: EMPTY_FACTORY_VELOCITY, 30: EMPTY_FACTORY_VELOCITY } }),
};

/**
 * A few hours after setup: people merges fill the period, SuperPlane has one
 * day of output. There is no earlier period, so no metric shows a delta.
 */
export const EarlyUsage: Story = {
  name: "Early usage",
  render: () =>
    renderVelocity({ [PRIMARY_FACTORY_ID]: { 14: EARLY_USAGE_FACTORY_VELOCITY, 30: EARLY_USAGE_FACTORY_VELOCITY } }),
};

/**
 * A cohort of 14 people: the People table shows its first 5 rows and a
 * "Show more" control that fetches the next 20, sorted and paged by the backend.
 */
export const PeopleLoadMore: Story = {
  name: "People show more",
  render: () =>
    renderVelocity({
      [PRIMARY_FACTORY_ID]: PEOPLE_LOAD_MORE_FACTORY_VELOCITY,
    }),
};

/**
 * Right after a repository is connected: the background sync has not stored the
 * repository history yet, so the People series is withheld and the People table
 * explains why the Manual work column is empty.
 */
export const PeopleSyncPending: Story = {
  name: "People sync pending",
  render: () =>
    renderVelocity({
      [PRIMARY_FACTORY_ID]: {
        14: PEOPLE_SYNC_PENDING_FACTORY_VELOCITY,
        30: PEOPLE_SYNC_PENDING_FACTORY_VELOCITY,
      },
    }),
};
