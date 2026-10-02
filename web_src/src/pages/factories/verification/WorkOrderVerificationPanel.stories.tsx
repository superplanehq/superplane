import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import { WorkOrderVerificationPanel } from "./WorkOrderVerificationPanel";
import {
  FAILED_VERIFICATION_RUN,
  PASSED_VERIFICATION_RUN,
  RUNNING_VERIFICATION_RUN,
} from "./__fixtures__/verificationFixtures";

/**
 * Verification results on the work order detail: run status, per-check
 * outcomes with command checks apart from agent reviews, and findings
 * grouped by severity.
 */
const meta = {
  title: "Factories/Verification/WorkOrderVerificationPanel",
  component: WorkOrderVerificationPanel,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[480px] max-w-4xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkOrderVerificationPanel>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A failed run: blocking findings stop the line. */
export const Failed: Story = {
  args: { run: FAILED_VERIFICATION_RUN },
};

/** A passed run: advisory findings are recorded but do not stop the line. */
export const Passed: Story = {
  args: { run: PASSED_VERIFICATION_RUN },
};

/** A run in progress: checks run in parallel and results appear as they finish. */
export const Running: Story = {
  args: { run: RUNNING_VERIFICATION_RUN },
};
