import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { FACTORIES_ORGANIZATION_ID, OPEN_WORK_ORDER, PRIMARY_FACTORY_KEY } from "./__fixtures__/factoryPageResponses";
import {
  BOOLEAN_CHECK_CI_PASS,
  BOOLEAN_CHECK_FLAKY_GATE_FAIL,
  BOOLEAN_CHECK_SECURITY_SCAN_FAIL,
  OPEN_WORK_ORDER_CHECKS,
} from "./__fixtures__/workOrderCheckFixtures";
import { presentWorkOrderChecks } from "./lib/workOrderChecks";
import { WorkOrderChecksSection } from "./WorkOrderChecksSection";

/**
 * Prototype: boolean (pass/fail) checks alongside the existing numeric
 * scores. A boolean card shows a Pass/Fail badge in place of the
 * score-and-meter; clicking either kind opens the same check dialog.
 *
 * These stories pass checks directly as props — the harnessed page story
 * (`Factories/Pages/Work Order Detail → Open`) is where boolean checks show
 * up through the real Checks section via the Storybook-only prototype slot.
 */
const meta = {
  title: "Factories/Components/WorkOrderChecksSection",
  component: WorkOrderChecksSection,
  parameters: { layout: "padded" },
  args: {
    organizationId: FACTORIES_ORGANIZATION_ID,
    factoryKey: PRIMARY_FACTORY_KEY,
    orderNumber: OPEN_WORK_ORDER.number,
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="max-w-3xl bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderChecksSection>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A passing boolean check next to a Fail-critical one. */
export const BooleanChecks: Story = {
  args: {
    checks: [BOOLEAN_CHECK_CI_PASS, BOOLEAN_CHECK_SECURITY_SCAN_FAIL],
  },
};

/** Fail decided as caution (amber), not critical (red) — the reporting automation's call. */
export const BooleanFailCaution: Story = {
  name: "Boolean — Fail (caution)",
  args: {
    checks: [BOOLEAN_CHECK_CI_PASS, BOOLEAN_CHECK_FLAKY_GATE_FAIL],
  },
};

/** The realistic view: scored checks (risk review, coverage, confidence) and boolean gates (CI, security scan) in one grid. */
export const MixedWithScoredChecks: Story = {
  args: {
    checks: [
      ...presentWorkOrderChecks(OPEN_WORK_ORDER_CHECKS),
      BOOLEAN_CHECK_CI_PASS,
      BOOLEAN_CHECK_SECURITY_SCAN_FAIL,
    ],
  },
};

export const Loading: Story = {
  args: {
    checks: [],
    isLoading: true,
  },
};

export const ErrorState: Story = {
  args: {
    checks: [],
    error: new Error("Failed to load checks"),
  },
};

/** No checks reported — the section renders nothing. */
export const Empty: Story = {
  args: {
    checks: [],
  },
};
