import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import {
  BOOLEAN_CHECK_CI_PASS,
  BOOLEAN_CHECK_FLAKY_GATE_FAIL,
  BOOLEAN_CHECK_SECURITY_SCAN_FAIL,
  OPEN_WORK_ORDER_CHECKS,
} from "./__fixtures__/workOrderCheckFixtures";
import { presentWorkOrderCheck } from "./lib/workOrderChecks";
import { WorkOrderCheckDialog } from "./WorkOrderCheckDialog";

/**
 * Expanded check dialog, opened. Boolean checks show a Pass/Fail badge
 * instead of the score-and-scale header and skip the trend line; everything
 * else (summary, analysis, attribution, "View run") is shared with scored
 * checks.
 */
const meta = {
  title: "Factories/Components/WorkOrderCheckDialog",
  component: WorkOrderCheckDialog,
  parameters: { layout: "padded" },
  args: {
    open: true,
    onClose: () => undefined,
  },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[420px] bg-white p-6 dark:bg-gray-900">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderCheckDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A scored check, for comparison — score, level badge, trend line, then analysis. */
export const Score: Story = {
  args: {
    check: presentWorkOrderCheck(OPEN_WORK_ORDER_CHECKS[0]),
    runHref: "#run",
  },
};

/** Passing boolean check — green Pass badge, no score line. */
export const BooleanPass: Story = {
  name: "Boolean — Pass",
  args: {
    check: BOOLEAN_CHECK_CI_PASS,
    runHref: "#run",
  },
};

/** Failing boolean check at critical level — red Fail badge. */
export const BooleanFailCritical: Story = {
  name: "Boolean — Fail (critical)",
  args: {
    check: BOOLEAN_CHECK_SECURITY_SCAN_FAIL,
    runHref: "#run",
  },
};

/** Failing boolean check at caution level — amber Fail badge, no run link. */
export const BooleanFailCaution: Story = {
  name: "Boolean — Fail (caution)",
  args: {
    check: BOOLEAN_CHECK_FLAKY_GATE_FAIL,
    runHref: null,
  },
};
