import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { ActionsPage } from "./consoleActionRuns";
import type { StepActionRun } from "./consoleStepActions";

const runs: StepActionRun[] = Array.from({ length: 3 }, (_, index) => {
  const runId = `confidence-run-${index + 1}`;
  return {
    runId,
    title: "Merge confidence",
    ordinal: index + 1,
    startedAt: `2026-11-05T${10 + index}:00:00Z`,
    rows: [
      {
        id: `${runId}-risk`,
        runId,
        title: `Reported Blast radius ${index + 2}/5`,
        detail:
          "The change updates webhook processing and connected service credentials. API and worker processes need the same configuration before rollout.\n\nReview the configuration for each process. Confirm that existing workflows can still use the connected services after deployment.",
      },
      {
        id: `${runId}-performance`,
        runId,
        title: "Reported Performance 5/5",
        detail:
          "The change saves matching events before external API calls. Workers process saved events asynchronously and retain the existing retry behavior.",
      },
      {
        id: `${runId}-risk-update`,
        runId,
        title: "Reported Blast radius 1/5",
        detail:
          "The updated configuration includes the required API and worker settings. Existing workflows remain available during rollout.",
      },
      {
        id: `${runId}-owner`,
        runId,
        title: "Assigned Ada as owner",
        detail: "Ada maintains the webhook processing code and can review the deployment configuration.",
      },
    ],
  };
});

const meta = {
  title: "Pages/Factories/Task console/Action runs",
  component: ActionsPage,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <div className="mx-auto max-w-3xl rounded-lg border border-border bg-background p-4 text-foreground">
        <h2 className="mb-3 text-sm font-medium">Merge confidence</h2>
        <Story />
      </div>
    ),
  ],
  args: { runs },
} satisfies Meta<typeof ActionsPage>;

export default meta;
type Story = StoryObj<typeof meta>;

export const RepeatedReports: Story = {};

export const PartialLoadFailure: Story = {
  render: (args) => {
    const [failed, setFailed] = useState(true);
    return (
      <ActionsPage
        {...args}
        failed={failed}
        onRetry={() => {
          console.log("Retry action history");
          setFailed(false);
        }}
      />
    );
  },
};
