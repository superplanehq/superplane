import type { Meta, StoryObj } from "@storybook/react-vite";

import type { CanvasAppFixture } from "@/pages/app/__fixtures__/handlers";

import { FactoriesHarness } from "../__fixtures__/FactoriesHarness";
import {
  defaultFactoriesFixture,
  emptyFactoriesFixture,
  FACTORIES_ORGANIZATION_ID,
  HOUR_AGO,
  LAST_WEEK,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_APPS,
  TWO_HOURS_AGO,
} from "../__fixtures__/factoryPageResponses";
import { AutomationsPage } from "./AutomationsPage";

/**
 * Automations page: v3-style card list + detail (automation card + Runs / Velocity tabs).
 */
const meta = {
  title: "Factories/Pages/Automations",
  component: AutomationsPage,
  parameters: { layout: "fullscreen" },
} satisfies Meta<typeof AutomationsPage>;

export default meta;

type Story = StoryObj<typeof meta>;

const automationsPath = `workspaces/${PRIMARY_FACTORY_ID}/automations`;
const implementerAppId = REFUND_FACTORY_APPS[1]?.id ?? "app-refund-implementer";

const implementerCanvasRunsFixture: CanvasAppFixture = {
  canvasId: implementerAppId,
  organizationId: FACTORIES_ORGANIZATION_ID,
  runs: {
    runs: [
      {
        id: "run-implement",
        canvasId: implementerAppId,
        state: "STATE_STARTED",
        createdAt: HOUR_AGO,
        updatedAt: HOUR_AGO,
        rootEvent: { customName: "Add refund reconciliation test" },
      },
      {
        id: "run-implement-2",
        canvasId: implementerAppId,
        state: "STATE_FINISHED",
        result: "RESULT_FAILED",
        createdAt: TWO_HOURS_AGO,
        updatedAt: HOUR_AGO,
        rootEvent: { customName: "Ship idempotent refund retries" },
      },
      {
        id: "run-implement-3",
        canvasId: implementerAppId,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        createdAt: LAST_WEEK,
        updatedAt: LAST_WEEK,
        rootEvent: { customName: "Backfill refund audit trail" },
      },
    ],
    totalCount: 3,
    hasNextPage: false,
  },
};

export const Populated: Story = {
  render: () => <FactoriesHarness pathSuffix={automationsPath} factoriesFixture={defaultFactoriesFixture} />,
};

export const DetailWithRuns: Story = {
  name: "Detail with runs",
  render: () => (
    <FactoriesHarness
      pathSuffix={`${automationsPath}/${implementerAppId}`}
      factoriesFixture={defaultFactoriesFixture}
      appFixture={implementerCanvasRunsFixture}
    />
  ),
};

export const EmptyFactory: Story = {
  name: "Empty factory",
  render: () => <FactoriesHarness pathSuffix={automationsPath} factoriesFixture={emptyFactoriesFixture} />,
};
