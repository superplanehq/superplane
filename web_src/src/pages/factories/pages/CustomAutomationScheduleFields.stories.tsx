import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { CustomAutomationSchedule } from "../lib/customAutomation";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { CustomAutomationScheduleFields } from "./CustomAutomationScheduleFields";

const scheduleError = "Enter a valid cron expression and a UTC offset from -12 to 14.";

const meta = {
  title: "Factories/Components/CustomAutomationScheduleFields",
  component: CustomAutomationScheduleFields,
  parameters: { layout: "padded" },
  args: {
    schedule: { cronExpression: "0 9 * * MON-FRI", utcOffset: 5.75 },
    onChange: (schedule: CustomAutomationSchedule) => console.log("Schedule changed", schedule),
  },
  decorators: [
    withFactoriesTheme,
    (Story, { args }) => {
      const [schedule, setSchedule] = useState(args.schedule);
      return (
        <ComponentStoryShell className="max-w-lg bg-background p-6">
          <Story
            args={{
              schedule,
              onChange: (next: CustomAutomationSchedule) => {
                args.onChange(next);
                setSchedule(next);
              },
            }}
          />
        </ComponentStoryShell>
      );
    },
  ],
} satisfies Meta<typeof CustomAutomationScheduleFields>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SavedFractionalOffset: Story = {};

export const EmptyInputs: Story = {
  args: { schedule: { cronExpression: "", utcOffset: Number.NaN } },
};

export const ErrorRecovery: Story = {
  args: { schedule: { cronExpression: "invalid", utcOffset: 15 } },
  render: function ErrorRecoveryForm(args) {
    const [error, setError] = useState(scheduleError);
    return (
      <form
        className="space-y-4"
        onSubmit={(event) => {
          event.preventDefault();
          console.log("Save schedule", args.schedule);
          // Simulate the server response without a configured workspace.
          const fields = args.schedule.cronExpression.trim().split(/\s+/);
          setError(fields.length === 5 || fields.length === 6 ? "" : scheduleError);
        }}
      >
        {error ? (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        ) : (
          <p role="status">Schedule saved.</p>
        )}
        <CustomAutomationScheduleFields {...args} />
        <Button type="submit">Save</Button>
      </form>
    );
  },
};
