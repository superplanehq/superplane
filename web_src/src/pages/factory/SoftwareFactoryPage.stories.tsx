import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { factoryPageData, newFactoryPageData } from "./__fixtures__/factoryFixtures";
import { SoftwareFactoryPage } from "./SoftwareFactoryPage";

/**
 * Dedicated page for a Software Factory, designed from
 * `docs/prd/software-factory.md`.
 *
 * A Factory is a first-class resource, not a renamed App — so this is its own
 * page shell: Factory header, then the four tabs the PRD specifies (Overview,
 * Work Orders, Automations, Velocity) at the recommended ~1,600px cap.
 *
 * All data arrives via props; every story below is a fixture.
 */
const meta = {
  title: "Pages/Software Factory",
  component: SoftwareFactoryPage,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof SoftwareFactoryPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const handlers = {
  onOpenWorkOrder: fn(),
  onOpenAutomation: fn(),
  onCreateWorkOrder: fn(),
  onCreateAutomation: fn(),
  onSelectRepository: fn(),
};

/**
 * Overview is the default tab, and Work Orders needing attention are rendered
 * *above* the metric row — the PRD's explicit ordering requirement.
 */
export const Overview: Story = {
  args: { data: factoryPageData, defaultTab: "overview", ...handlers },
};

/** The complete queue: Needs attention → Running → Recently done → Unsuccessful. */
export const WorkOrders: Story = {
  args: { data: factoryPageData, defaultTab: "work-orders", ...handlers },
};

/** A finite operational list; opening one hands off to the existing Canvas editor. */
export const Automations: Story = {
  args: { data: factoryPageData, defaultTab: "automations", ...handlers },
};

/**
 * Three cohorts, identical indicators, no ranking — the PRD requires the
 * comparison read neutrally rather than as humans versus the Factory. Human
 * tracked cost shows "Not available" rather than $0.
 */
export const Velocity: Story = {
  args: { data: factoryPageData, defaultTab: "velocity", ...handlers },
};

/**
 * The PRD's "blank by default" creation outcome: a persisted Factory with zero
 * Automations and no Work Orders, which must still be a coherent page.
 */
export const NewFactory: Story = {
  args: { data: newFactoryPageData, defaultTab: "overview", ...handlers },
};

/** Operational trouble surfaced in the header, above the tabs. */
export const Degraded: Story = {
  args: {
    ...handlers,
    defaultTab: "overview",
    data: {
      ...factoryPageData,
      factory: {
        ...factoryPageData.factory,
        status: "degraded",
        statusDetail: "The GitHub connection expired — Automations can commit but cannot open pull requests.",
      },
    },
  },
};
