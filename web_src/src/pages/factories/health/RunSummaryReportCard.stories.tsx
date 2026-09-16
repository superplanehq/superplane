import type { Meta, StoryObj } from "@storybook/react-vite";

import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";
import {
  RUN_SUMMARY_FAILED,
  RUN_SUMMARY_PASSED,
} from "@/pages/factories/verification/__fixtures__/verificationFixtures";

import { RunSummaryReportCard } from "./RunSummaryReportCard";

/**
 * Structured summary of one verification run: findings detected, fixed, and
 * remaining by severity, plus the gate result. Also shown as a Slack
 * message preview.
 */
const meta = {
  title: "Factories/Health/RunSummaryReportCard",
  component: RunSummaryReportCard,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-[260px] max-w-2xl bg-gray-50 p-6 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof RunSummaryReportCard>;

export default meta;

type Story = StoryObj<typeof meta>;

/** A failed gate on the work order timeline. */
export const GateFailed: Story = {
  args: { report: RUN_SUMMARY_FAILED },
};

/** A passed gate on the work order timeline. */
export const GatePassed: Story = {
  args: { report: RUN_SUMMARY_PASSED },
};

/** The same report as a Slack message preview. */
export const SlackPreview: Story = {
  args: { report: RUN_SUMMARY_FAILED, variant: "slack" },
};
