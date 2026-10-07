import type { Meta, StoryObj } from "@storybook/react-vite";

import { FactoriesHarness } from "../../__fixtures__/FactoriesHarness";
import { defaultFactoriesFixture, PRIMARY_FACTORY_KEY } from "../../__fixtures__/factoryPageResponses";
import { overviewRedesignFixture } from "./overviewRedesignFixture";
import { OverviewRedesignPage } from "./OverviewRedesignPage";
import { FRESH_OVERVIEW, POPULATED_OVERVIEW, QUIET_OVERVIEW } from "./overviewRedesignMocks";

/**
 * Workspace Overview redesign baseline. Storybook-only: mounted through the
 * real workspace chrome (sidebar + header) with the overview route swapped
 * for the redesigned page. All data is mocked; nothing touches the app.
 */
const meta = {
  title: "Factories/Pages/Overview Redesign",
  parameters: { layout: "fullscreen" },
} satisfies Meta;

export default meta;

type Story = StoryObj;

const overviewPath = `workspaces/${PRIMARY_FACTORY_KEY}/overview`;

function PopulatedOverview() {
  return <OverviewRedesignPage data={POPULATED_OVERVIEW} />;
}

function QuietOverview() {
  return <OverviewRedesignPage data={QUIET_OVERVIEW} />;
}

function FreshOverview() {
  return <OverviewRedesignPage data={FRESH_OVERVIEW} />;
}

/** Busy workspace: items in every section, all rail cards populated. */
export const Populated: Story = {
  render: () => (
    <FactoriesHarness
      pathSuffix={overviewPath}
      factoriesFixture={overviewRedesignFixture}
      enableOnboarding={false}
      pageOverrides={{ overview: PopulatedOverview }}
    />
  ),
};

/** Quiet workspace: empty attention queue is the best news this page can show. */
export const QuietWorkspace: Story = {
  name: "Quiet workspace",
  render: () => (
    <FactoriesHarness
      pathSuffix={overviewPath}
      factoriesFixture={overviewRedesignFixture}
      enableOnboarding={false}
      pageOverrides={{ overview: QuietOverview }}
    />
  ),
};

/** Fresh workspace: nothing ran yet, repository scan in progress. */
export const FreshWorkspace: Story = {
  name: "Fresh workspace",
  render: () => (
    <FactoriesHarness
      pathSuffix={overviewPath}
      factoriesFixture={defaultFactoriesFixture}
      enableOnboarding={false}
      pageOverrides={{ overview: FreshOverview }}
    />
  ),
};
